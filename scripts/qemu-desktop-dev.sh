#!/usr/bin/env bash
# Interactive desktop-dev loop: boot the live ISO in a GTK QEMU window
# and expose the host tree over virtio-9p so hypr / AGS / wallpaper
# changes can be copied in without pacstrap+xz.
#
# This is not the ISO smoke path. Use scripts/qemu-boot-test.sh for
# headless serial + QMP screenshots. Use scripts/build-iso.sh when
# packages, vendor builds, or squashfs contents change.
#
# virtio-vga without xres/yres comes up at 640x480. Both QEMU helpers
# pin 1920x1080 so that mode is in the EDID. Hyprland still must not use
# mode = "preferred" — virtio lists 640x480@119.99 first. Session config
# pins 1920x1080@60 in desktop/hypr/hyprland.lua.
#
# Arch host packages (abox):
#   pacman -S --needed qemu-system-x86 edk2-ovmf qemu-ui-gtk
# Needs /dev/kvm (user in group kvm). OVMF is required (UEFI-only ISO).
#
# Usage:
#   ./scripts/qemu-desktop-dev.sh
#   ./scripts/qemu-desktop-dev.sh --help
#   CODA_ISO=/path/to/codalinux-*.iso ./scripts/qemu-desktop-dev.sh --share .
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

iso="${CODA_ISO:-}"
out_dir="${CODA_QEMU_DESKTOP_OUT:-${root}/out/qemu-desktop-dev}"
ram="${CODA_QEMU_RAM:-4G}"
cpus="${CODA_QEMU_CPUS:-}"
display_mode="gtk"
wait_forever=1
dry_run=0
share="${CODA_QEMU_SHARE:-${root}}"
mount_tag="${CODA_QEMU_MOUNT_TAG:-coda-host}"
guest_mount="${CODA_QEMU_GUEST_MOUNT:-/mnt/coda-host}"
share_writable=0
qemu_pid=""
cleaned=0

usage() {
  cat <<'EOF'
Usage: qemu-desktop-dev.sh [options] [ISO]

Boot the CodaLinux UEFI live ISO with QEMU/KVM + OVMF for desktop UX
trial/error. Default is a GTK window that stays up until Ctrl-C, with
the host repo shared into the guest over virtio-9p (tag coda-host).
Guest GPU is virtio-vga at 1920x1080 (bare virtio-vga is 640x480).
Hyprland must pin 1920x1080@60; preferred picks virtio 640x480@119.99.

ISO is the first non-option argument, or $CODA_ISO, or the newest
out/codalinux-*.iso under the repo root.

Options:
  -h, --help                 Show this help
  --display                  GTK window (default)
  --spice                    Spice window (needs qemu-ui-spice-app)
  --headless                 No window (serial/QMP only; still waits)
  --wait                     Keep the VM until Ctrl-C (default)
  --share DIR                Host directory to export (default: repo
                             root, or $CODA_QEMU_SHARE). Share at
                             least desktop/, branding/, and scripts/.
  --tag NAME                 9p mount tag (default: coda-host)
  --mount-tag NAME           Alias for --tag
  --guest-mount PATH         Printed guest mountpoint
                             (default: /mnt/coda-host)
  --writable                 Export the host --share as read-write
                             (default: readonly; guest-logs is always
                             writable)
  --ram SIZE                 QEMU -m (default 4G; $CODA_QEMU_RAM)
  --cpus N                   vCPUs (default: min(nproc, 4), at least 2)
  --out DIR                  Artifacts (default out/qemu-desktop-dev
                             or $CODA_QEMU_DESKTOP_OUT)
  --dry-run                  Print the QEMU command + guest steps;
                             do not start QEMU

Artifacts (created at run time):
  serial.log     guest serial
  ovmf-debug.log OVMF debugcon (port 0x402)
  qmp.sock       QMP unix socket
  qmp.sh         Helper: ./qmp.sh screendump [file.png]
  qemu.pid       QEMU pid
  qemu.cmd       Exact command line
  guest-logs/    Writable 9p target (tag coda-guest-logs)

Examples:
  ./scripts/qemu-desktop-dev.sh
  ./scripts/qemu-desktop-dev.sh --share . --wait
  ./scripts/qemu-desktop-dev.sh --dry-run
  CODA_ISO=out/codalinux-2026.09.11-x86_64.iso ./scripts/qemu-desktop-dev.sh
EOF
}

log() { printf 'qemu-desktop-dev: %s\n' "$*"; }
die() { printf 'qemu-desktop-dev: %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --headless) display_mode="headless"; shift ;;
    --display) display_mode="gtk"; shift ;;
    --spice) display_mode="spice"; shift ;;
    --wait) wait_forever=1; shift ;;
    --dry-run) dry_run=1; shift ;;
    --share)
      [[ $# -ge 2 ]] || die "--share needs a directory"
      share="$2"
      shift 2
      ;;
    --tag|--mount-tag)
      [[ $# -ge 2 ]] || die "$1 needs a value"
      mount_tag="$2"
      shift 2
      ;;
    --guest-mount)
      [[ $# -ge 2 ]] || die "--guest-mount needs a path"
      guest_mount="$2"
      shift 2
      ;;
    --writable) share_writable=1; shift ;;
    --ram)
      [[ $# -ge 2 ]] || die "--ram needs a value"
      ram="$2"
      shift 2
      ;;
    --cpus)
      [[ $# -ge 2 ]] || die "--cpus needs a value"
      cpus="$2"
      shift 2
      ;;
    --out)
      [[ $# -ge 2 ]] || die "--out needs a value"
      out_dir="$2"
      shift 2
      ;;
    --)
      shift
      break
      ;;
    -*)
      die "unknown option: $1 (try --help)"
      ;;
    *)
      if [[ -n "${iso}" ]]; then
        die "unexpected extra argument: $1"
      fi
      iso="$1"
      shift
      ;;
  esac
done

if [[ $# -gt 0 ]]; then
  if [[ -n "${iso}" ]]; then
    die "unexpected extra argument: $1"
  fi
  iso="$1"
  shift
fi
[[ $# -eq 0 ]] || die "unexpected extra argument: $1"

find_iso() {
  if [[ -n "${iso}" ]]; then
    if [[ ! -f "${iso}" && "${dry_run}" -ne 1 ]]; then
      die "ISO not found: ${iso}"
    fi
    printf '%s' "${iso}"
    return 0
  fi
  local newest=""
  local candidate
  shopt -s nullglob
  for candidate in "${root}/out"/codalinux-*.iso; do
    if [[ -z "${newest}" || "${candidate}" -nt "${newest}" ]]; then
      newest="${candidate}"
    fi
  done
  shopt -u nullglob
  if [[ -n "${newest}" ]]; then
    printf '%s' "${newest}"
    return 0
  fi
  if [[ "${dry_run}" -eq 1 ]]; then
    printf '%s' "${root}/out/codalinux-YYYY.MM.DD-x86_64.iso"
    return 0
  fi
  die "no ISO given and no ${root}/out/codalinux-*.iso (build with ./scripts/build-iso.sh)"
}

find_ovmf() {
  local code vars
  for code in \
    /usr/share/edk2/x64/OVMF_CODE.4m.fd \
    /usr/share/edk2/x64/OVMF_CODE.fd \
    /usr/share/edk2-ovmf/x64/OVMF_CODE.4m.fd \
    /usr/share/edk2-ovmf/x64/OVMF_CODE.fd \
    /usr/share/OVMF/OVMF_CODE_4M.fd \
    /usr/share/OVMF/OVMF_CODE.fd
  do
    [[ -f "${code}" ]] || continue
    vars="${code/OVMF_CODE/OVMF_VARS}"
    if [[ -f "${vars}" ]]; then
      printf 'split\t%s\t%s\n' "${code}" "${vars}"
      return 0
    fi
  done
  local f
  for f in \
    /usr/share/edk2/x64/OVMF.4m.fd \
    /usr/share/edk2/x64/OVMF.fd \
    /usr/share/edk2-ovmf/x64/OVMF.4m.fd \
    /usr/share/edk2-ovmf/x64/OVMF.fd \
    /usr/share/OVMF/OVMF.fd
  do
    if [[ -f "${f}" ]]; then
      printf 'bios\t%s\t\n' "${f}"
      return 0
    fi
  done
  if [[ "${dry_run}" -eq 1 ]]; then
    printf 'split\t/usr/share/edk2/x64/OVMF_CODE.4m.fd\t/usr/share/edk2/x64/OVMF_VARS.4m.fd\n'
    return 0
  fi
  die "OVMF firmware not found. On Arch: pacman -S --needed edk2-ovmf"
}

pick_cpus() {
  if [[ -n "${cpus}" ]]; then
    printf '%s' "${cpus}"
    return 0
  fi
  local n
  n="$(nproc 2>/dev/null || echo 2)"
  if [[ "${n}" -lt 2 ]]; then
    n=2
  elif [[ "${n}" -gt 4 ]]; then
    n=4
  fi
  printf '%s' "${n}"
}

qmp_cmd() {
  local sock="$1"
  local payload="$2"
  python3 - "$sock" "$payload" <<'PY'
import json, socket, sys

class Qmp:
    def __init__(self, path):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.settimeout(30)
        self.sock.connect(path)
        self.buf = ""
        self.read_reply()
        self.send('{"execute":"qmp_capabilities"}')
        cap = self.read_reply()
        if "error" in cap:
            raise SystemExit(cap["error"].get("desc", str(cap)))

    def send(self, payload):
        self.sock.sendall(payload.encode() + b"\r\n")

    def read_msg(self):
        dec = json.JSONDecoder()
        while True:
            text = self.buf.lstrip()
            if text:
                try:
                    obj, idx = dec.raw_decode(text)
                    self.buf = text[idx:]
                    return obj
                except json.JSONDecodeError:
                    pass
            chunk = self.sock.recv(4096)
            if not chunk:
                raise SystemExit("qmp: connection closed")
            self.buf += chunk.decode()

    def read_reply(self):
        while True:
            obj = self.read_msg()
            if "event" in obj:
                continue
            return obj

q = Qmp(sys.argv[1])
q.send(sys.argv[2])
reply = q.read_reply()
print(json.dumps(reply))
if "error" in reply:
    raise SystemExit(reply["error"].get("desc", str(reply)))
PY
}

qemu_alive() {
  [[ -n "${qemu_pid}" ]] && kill -0 "${qemu_pid}" 2>/dev/null
}

cleanup() {
  [[ "${cleaned}" -eq 1 ]] && return 0
  cleaned=1
  if [[ -z "${qemu_pid}" && -f "${pid_file:-}" ]]; then
    qemu_pid="$(<"${pid_file}")" || true
  fi
  if qemu_alive; then
    if [[ -S "${qmp_sock:-}" ]]; then
      qmp_cmd "${qmp_sock}" '{"execute":"quit"}' >/dev/null 2>&1 || true
    fi
    if qemu_alive; then
      kill -TERM "${qemu_pid}" 2>/dev/null || true
      sleep 0.3
    fi
    if qemu_alive; then
      kill -KILL "${qemu_pid}" 2>/dev/null || true
    fi
    wait "${qemu_pid}" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

if [[ "${dry_run}" -ne 1 ]]; then
  command -v qemu-system-x86_64 >/dev/null 2>&1 || die "qemu-system-x86_64 not found. On Arch: pacman -S --needed qemu-system-x86"
  command -v python3 >/dev/null 2>&1 || die "python3 not found (used for QMP)"
fi

if [[ ! -d "${share}" ]]; then
  if [[ "${dry_run}" -eq 1 ]]; then
    log "warning: --share directory does not exist yet: ${share}"
  else
    die "share directory not found: ${share}"
  fi
fi
share="$(cd "${share}" 2>/dev/null && pwd || printf '%s' "${share}")"

iso="$(find_iso)"
cpus="$(pick_cpus)"
ovmf_line="$(find_ovmf)"
ovmf_mode="${ovmf_line%%$'\t'*}"
ovmf_rest="${ovmf_line#*$'\t'}"
ovmf_code="${ovmf_rest%%$'\t'*}"
ovmf_vars="${ovmf_rest#*$'\t'}"

accel_args=()
cpu_args=(-cpu max)
if [[ -e /dev/kvm && -r /dev/kvm && -w /dev/kvm ]]; then
  accel_args=(-machine q35,accel=kvm -enable-kvm)
  cpu_args=(-cpu host)
else
  log "warning: /dev/kvm not usable; falling back to TCG (slow)"
  accel_args=(-machine q35,accel=tcg)
fi

display_args=()
case "${display_mode}" in
  headless) display_args=(-display none) ;;
  gtk) display_args=(-display gtk,gl=off,grab-on-hover=on) ;;
  spice) display_args=(-display spice-app) ;;
  *) die "unknown display mode: ${display_mode}" ;;
esac

install -d "${out_dir}"
serial_log="${out_dir}/serial.log"
qmp_sock="${out_dir}/qmp.sock"
mon_sock="${out_dir}/monitor.sock"
pid_file="${out_dir}/qemu.pid"
cmd_file="${out_dir}/qemu.cmd"
vars_copy="${out_dir}/OVMF_VARS.fd"
rm -f "${qmp_sock}" "${mon_sock}" "${pid_file}"

fw_args=()
if [[ "${ovmf_mode}" == split ]]; then
  if [[ "${dry_run}" -eq 1 && ! -f "${ovmf_vars}" ]]; then
    fw_args=(
      -drive "if=pflash,format=raw,readonly=on,file=${ovmf_code}"
      -drive "if=pflash,format=raw,file=${vars_copy}"
    )
  else
    cp -f "${ovmf_vars}" "${vars_copy}"
    chmod u+w "${vars_copy}"
    fw_args=(
      -drive "if=pflash,format=raw,readonly=on,file=${ovmf_code}"
      -drive "if=pflash,format=raw,file=${vars_copy}"
    )
  fi
else
  fw_args=(-bios "${ovmf_code}")
fi

# Host tree: readonly by default (edit on the host, copy into the live
# overlay). --writable makes it read-write. Always add a second writable
# virtfs for guest logs (hyprpaper.log / coda-wallpaper.log).
guest_logs="${out_dir}/guest-logs"
install -d "${guest_logs}"
chmod a+rwx "${guest_logs}" || true
virtfs_host="local,path=${share},mount_tag=${mount_tag},security_model=none"
if [[ "${share_writable}" -ne 1 ]]; then
  virtfs_host="${virtfs_host},readonly=on"
fi
virtfs_logs="local,path=${guest_logs},mount_tag=coda-guest-logs,security_model=none"

cmd=(
  qemu-system-x86_64
  -name coda-qemu-desktop-dev
  "${accel_args[@]}"
  "${cpu_args[@]}"
  -smp "${cpus}"
  -m "${ram}"
  "${fw_args[@]}"
  -cdrom "${iso}"
  -boot order=d,menu=on
  -device virtio-vga,xres=1920,yres=1080
  "${display_args[@]}"
  -virtfs "${virtfs_host}"
  -virtfs "${virtfs_logs}"
  -device virtio-net-pci,netdev=n0
  -netdev user,id=n0
  -device qemu-xhci
  -device usb-tablet
  -serial "file:${serial_log}"
  -debugcon "file:${out_dir}/ovmf-debug.log"
  -global isa-debugcon.iobase=0x402
  -qmp "unix:${qmp_sock},server,nowait"
  -monitor "unix:${mon_sock},server,nowait"
  -pidfile "${pid_file}"
)

{
  printf '%q ' "${cmd[@]}"
  printf '\n'
} >"${cmd_file}"

cat >"${out_dir}/qmp.sh" <<EOF
#!/usr/bin/env bash
# Session helper written by qemu-desktop-dev.sh
set -euo pipefail
sock=$(printf '%q' "${qmp_sock}")
if [[ \${1:-} == screendump ]]; then
  dest=\${2:-${out_dir}/shot-manual.png}
  payload=\$(python3 -c 'import json,sys; print(json.dumps({"execute":"screendump","arguments":{"filename":sys.argv[1],"format":"png"}}))' "\$dest")
else
  payload=\${1:-'{"execute":"query-status"}'}
fi
python3 - "\$sock" "\$payload" <<'PY'
import json, socket, sys

class Qmp:
    def __init__(self, path):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.settimeout(30)
        self.sock.connect(path)
        self.buf = ""
        self.read_reply()
        self.send('{"execute":"qmp_capabilities"}')
        cap = self.read_reply()
        if "error" in cap:
            raise SystemExit(cap["error"].get("desc", str(cap)))

    def send(self, payload):
        self.sock.sendall(payload.encode() + b"\\r\\n")

    def read_msg(self):
        dec = json.JSONDecoder()
        while True:
            text = self.buf.lstrip()
            if text:
                try:
                    obj, idx = dec.raw_decode(text)
                    self.buf = text[idx:]
                    return obj
                except json.JSONDecodeError:
                    pass
            chunk = self.sock.recv(4096)
            if not chunk:
                raise SystemExit("qmp closed")
            self.buf += chunk.decode()

    def read_reply(self):
        while True:
            obj = self.read_msg()
            if "event" in obj:
                continue
            return obj

q = Qmp(sys.argv[1])
q.send(sys.argv[2])
print(json.dumps(q.read_reply()))
PY
EOF
chmod +x "${out_dir}/qmp.sh"

print_guest_steps() {
  cat <<EOF

Guest 9p shares (copy configs / drop logs; do not rebuild the ISO)
  mount tag:    ${mount_tag}
  host path:    ${share}
  guest mount:  ${guest_mount}
  host writable: $([[ "${share_writable}" -eq 1 ]] && echo yes || echo no)

  guest-logs tag: coda-guest-logs
  host path:      ${guest_logs}
  guest mount:    /mnt/coda-guest-logs
  (always writable — drop /tmp/hyprpaper.log here)

  # As root (tty2 is a root console) after the desktop is up:
  modprobe 9pnet_virtio 9p || true
  mkdir -p ${guest_mount} /mnt/coda-guest-logs
  mount -t 9p -o trans=virtio,version=9p2000.L ${mount_tag} ${guest_mount}
  mount -t 9p -o trans=virtio,version=9p2000.L coda-guest-logs /mnt/coda-guest-logs

  # Preferred: helper from the share (always the host copy).
  # Run as root on tty2 after Hyprland is painted on tty1. Restart is
  # as user live — never as root — and waits/retries coda-ags.
  ${guest_mount}/scripts/coda-sync-desktop-from-host.sh ${guest_mount}
  # After the next ISO rebuild this is also: coda-sync-desktop-from-host
  # system-config binaries/units ship on the ISO; sync also copies
  # ${guest_mount}/core/system-config/bin/* if you built them on the host.
  # Verify AGS came back as live (not root):
  #   pgrep -u live -a ags
  # Sync logs "AGS running as live" or a clear "display not ready"
  # if Hyprland has no wayland socket yet. Retry after the session paints.

  # Or copy by hand, then restart from a Hyprland terminal:
  install -m 0755 ${guest_mount}/scripts/coda-wallpaper /usr/local/bin/coda-wallpaper
  install -m 0755 ${guest_mount}/scripts/coda-hyprpaper /usr/local/bin/coda-hyprpaper
  cp ${guest_mount}/desktop/hypr/hyprpaper.conf /etc/xdg/hypr/
  cp ${guest_mount}/desktop/hypr/hyprpaper.conf /home/live/.config/hypr/
  cp ${guest_mount}/desktop/hypr/hyprland.lua /etc/xdg/hypr/
  cp ${guest_mount}/desktop/hypr/hyprland.lua /home/live/.config/hypr/
  cp -a ${guest_mount}/desktop/ags/. /usr/local/share/codalinux/ags/
  # wallpaper (if you changed branding/wallpapers/default.png):
  cp ${guest_mount}/branding/wallpapers/default.png /usr/share/backgrounds/codalinux/default.png
  # then, as user live in a foot window:
  killall swaybg hyprpaper; coda-wallpaper
  coda-ags quit; coda-ags &
  hyprctl reload
  # If swaybg is missing on this ISO: pacman -S --noconfirm swaybg
  # Logs: /tmp/hyprpaper.log, /var/log/coda-wallpaper.log, /mnt/coda-guest-logs/

  If mount fails, check 9p modules on the live image:
    find /usr/lib/modules/\$(uname -r) -name '*9p*'
  Arch linux usually ships 9p / 9pnet / 9pnet_virtio. If that find is
  empty, adding those modules to the ISO is a follow-up (do not guess).

Ctrl-C on the host stops QEMU. Artifacts: ${out_dir}

EOF
}

log "ISO ${iso}"
log "OVMF ${ovmf_mode} ${ovmf_code}${ovmf_vars:+ ${ovmf_vars}}"
log "share ${share} tag=${mount_tag} → guest ${guest_mount} (writable=${share_writable})"
log "guest-logs ${guest_logs} tag=coda-guest-logs → /mnt/coda-guest-logs"
log "artifacts ${out_dir} (${display_mode}, ${ram}, ${cpus} cpu, wait)"

print_guest_steps

if [[ "${dry_run}" -eq 1 ]]; then
  cat "${cmd_file}"
  trap - EXIT INT TERM
  exit 0
fi

: >"${serial_log}"
"${cmd[@]}" &
qemu_pid=$!
sleep 0.2
if [[ -f "${pid_file}" ]]; then
  qemu_pid="$(<"${pid_file}")"
fi
if ! qemu_alive; then
  die "QEMU exited immediately; see ${serial_log}"
fi
log "QEMU pid ${qemu_pid}  qmp ${qmp_sock}"
log "waiting until Ctrl-C (QEMU pid ${qemu_pid})"
if [[ "${wait_forever}" -eq 1 ]]; then
  wait "${qemu_pid}" || true
  qemu_pid=""
fi
# EXIT trap runs cleanup
