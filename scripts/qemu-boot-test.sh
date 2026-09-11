#!/usr/bin/env bash
# Boot the CodaLinux UEFI live ISO under QEMU/KVM (preferred automated
# test path). VirtualBox remains valid for manual VMSVGA checks; do not
# remove those docs. Serial + QMP screenshots replace VBoxManage inject
# / screenshotpng for Horos wallpaper and idle-lock smoke tests.
#
# virtio-vga without xres/yres comes up at 640x480. Both QEMU helpers
# pin 1920x1080 so the live desktop (and screenshots) are usable.
#
# Arch host packages (abox):
#   pacman -S --needed qemu-system-x86 edk2-ovmf
#   # optional GTK window:
#   pacman -S --needed qemu-ui-gtk
#   # optional Spice window:
#   pacman -S --needed qemu-ui-spice-app
# Needs /dev/kvm (user in group kvm). OVMF is required (UEFI-only ISO).
#
# Usage:
#   ./scripts/qemu-boot-test.sh [--headless|--display|--spice] [ISO]
#   ./scripts/qemu-boot-test.sh --help
#   CODA_ISO=/path/to/codalinux-*.iso ./scripts/qemu-boot-test.sh --wait
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

iso="${CODA_ISO:-}"
out_dir="${CODA_QEMU_OUT:-${root}/out/qemu-boot-test}"
ram="${CODA_QEMU_RAM:-4G}"
cpus="${CODA_QEMU_CPUS:-}"
display_mode="headless"
timeout_s="${CODA_QEMU_TIMEOUT:-90}"
wait_forever=0
shot_delay_s="${CODA_QEMU_SHOT_DELAY:-40}"
shot_interval_s="${CODA_QEMU_SHOT_INTERVAL:-0}"
dry_run=0
qemu_pid=""
cleaned=0

usage() {
  cat <<'EOF'
Usage: qemu-boot-test.sh [options] [ISO]

Boot the CodaLinux UEFI live ISO with QEMU/KVM + OVMF. Default is
headless: serial log, QMP socket, screenshot(s) after the desktop
should have settled. Ctrl-C stops QEMU and leaves artifacts.
Guest GPU is virtio-vga at 1920x1080 (bare virtio-vga is 640x480).

ISO is the first non-option argument, or $CODA_ISO, or the newest
out/codalinux-*.iso under the repo root.

Options:
  -h, --help                 Show this help
  --headless                 No window (default; automation)
  --display                  GTK window (needs qemu-ui-gtk)
  --spice                    Spice window (needs qemu-ui-spice-app)
  --timeout SECONDS          QMP quit after N seconds (default 90)
  --wait                     Keep the VM until Ctrl-C (timeout only
                             ends scheduled screenshots, not the VM)
  --screenshot-delay SECONDS First screenshot (default 40)
  --screenshot-interval SECONDS
                             Extra shots until timeout; 0 = one shot
                             plus a final shot on exit (default 0)
  --ram SIZE                 QEMU -m (default 4G; $CODA_QEMU_RAM)
  --cpus N                   vCPUs (default: min(nproc, 4), at least 2)
  --out DIR                  Artifacts (default out/qemu-boot-test
                             or $CODA_QEMU_OUT)
  --dry-run                  Print the QEMU command; do not start it

Artifacts (created at run time):
  serial.log     guest serial (kernel logs need console=ttyS0)
  ovmf-debug.log OVMF debugcon (port 0x402)
  qmp.sock       QMP unix socket
  qmp.sh         Helper: ./qmp.sh screendump [file.png]
  qemu.pid       QEMU pid
  qemu.cmd       Exact command line
  shot-*.png     QMP screendumps

Examples:
  ./scripts/qemu-boot-test.sh
  ./scripts/qemu-boot-test.sh --display --wait out/codalinux-2026.09.11-x86_64.iso
  ./scripts/qemu-boot-test.sh --headless --timeout 120 --screenshot-delay 50
EOF
}

log() { printf 'qemu-boot-test: %s\n' "$*"; }
die() { printf 'qemu-boot-test: %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --headless) display_mode="headless"; shift ;;
    --display) display_mode="gtk"; shift ;;
    --spice) display_mode="spice"; shift ;;
    --wait) wait_forever=1; shift ;;
    --dry-run) dry_run=1; shift ;;
    --timeout)
      [[ $# -ge 2 ]] || die "--timeout needs a value"
      timeout_s="$2"
      shift 2
      ;;
    --screenshot-delay)
      [[ $# -ge 2 ]] || die "--screenshot-delay needs a value"
      shot_delay_s="$2"
      shift 2
      ;;
    --screenshot-interval)
      [[ $# -ge 2 ]] || die "--screenshot-interval needs a value"
      shot_interval_s="$2"
      shift 2
      ;;
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
  # Prefer split CODE+VARS (writable NVRAM copy). Combined -bios is fallback.
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
        self.read_reply()  # greeting
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

screendump() {
  local dest="$1"
  local attempt
  for attempt in 1 2 3; do
    if qmp_cmd "${qmp_sock}" "{\"execute\":\"screendump\",\"arguments\":{\"filename\":\"${dest}\",\"format\":\"png\"}}" >/dev/null; then
      log "screenshot ${dest}"
      return 0
    fi
    log "QMP screendump attempt ${attempt} failed; retrying"
    sleep 2
  done
  dest="${dest%.png}.ppm"
  if qmp_cmd "${qmp_sock}" "{\"execute\":\"screendump\",\"arguments\":{\"filename\":\"${dest}\"}}" >/dev/null; then
    log "screenshot ${dest}"
    return 0
  fi
  log "warning: screendump failed (QEMU not answering QMP). serial/ovmf logs still saved."
  return 1
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
  cp -f "${ovmf_vars}" "${vars_copy}"
  chmod u+w "${vars_copy}"
  fw_args=(
    -drive "if=pflash,format=raw,readonly=on,file=${ovmf_code}"
    -drive "if=pflash,format=raw,file=${vars_copy}"
  )
else
  fw_args=(-bios "${ovmf_code}")
fi

cmd=(
  qemu-system-x86_64
  -name coda-qemu-boot-test
  "${accel_args[@]}"
  "${cpu_args[@]}"
  -smp "${cpus}"
  -m "${ram}"
  "${fw_args[@]}"
  -cdrom "${iso}"
  -boot order=d,menu=on
  -device virtio-vga,xres=1920,yres=1080
  "${display_args[@]}"
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
# Session helper written by qemu-boot-test.sh
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

log "ISO ${iso}"
log "OVMF ${ovmf_mode} ${ovmf_code}${ovmf_vars:+ ${ovmf_vars}}"
log "artifacts ${out_dir} (${display_mode}, ${ram}, ${cpus} cpu)"

if [[ "${dry_run}" -eq 1 ]]; then
  cat "${cmd_file}"
  trap - EXIT INT TERM
  exit 0
fi

: >"${serial_log}"
"${cmd[@]}" &
qemu_pid=$!
# pidfile can lag a moment
sleep 0.2
if [[ -f "${pid_file}" ]]; then
  qemu_pid="$(<"${pid_file}")"
fi
if ! qemu_alive; then
  die "QEMU exited immediately; see ${serial_log}"
fi
log "QEMU pid ${qemu_pid}  qmp ${qmp_sock}"

deadline=$((SECONDS + timeout_s))
next_shot=$((SECONDS + shot_delay_s))
shot_n=0

take_shot() {
  shot_n=$((shot_n + 1))
  local dest
  dest="$(printf '%s/shot-%02d.png' "${out_dir}" "${shot_n}")"
  if ! qemu_alive; then
    die "QEMU died before screenshot; see ${serial_log}"
  fi
  # QMP socket appears after QEMU starts accepting
  local waited=0
  while [[ ! -S "${qmp_sock}" && "${waited}" -lt 20 ]]; do
    sleep 0.25
    waited=$((waited + 1))
    qemu_alive || die "QEMU died before QMP; see ${serial_log}"
  done
  [[ -S "${qmp_sock}" ]] || die "QMP socket missing: ${qmp_sock}"
  screendump "${dest}" || true
}

while qemu_alive; do
  now="${SECONDS}"
  if [[ "${now}" -ge "${next_shot}" && "${shot_n}" -eq 0 ]]; then
    take_shot
    if [[ "${shot_interval_s}" -gt 0 ]]; then
      next_shot=$((now + shot_interval_s))
    else
      next_shot=999999
    fi
  elif [[ "${shot_interval_s}" -gt 0 && "${shot_n}" -gt 0 && "${now}" -ge "${next_shot}" ]]; then
    take_shot
    next_shot=$((now + shot_interval_s))
  fi
  if [[ "${wait_forever}" -eq 0 && "${now}" -ge "${deadline}" ]]; then
    break
  fi
  if [[ "${wait_forever}" -eq 1 && "${now}" -ge "${deadline}" && "${shot_interval_s}" -gt 0 ]]; then
    # Stop repeating shots after timeout but keep the VM.
    next_shot=999999
  fi
  sleep 1
done

if qemu_alive && [[ "${shot_n}" -eq 0 ]]; then
  take_shot
elif qemu_alive && [[ "${wait_forever}" -eq 0 ]]; then
  take_shot
fi

if [[ "${wait_forever}" -eq 1 ]]; then
  log "waiting until Ctrl-C (QEMU pid ${qemu_pid})"
  wait "${qemu_pid}" || true
  qemu_pid=""
else
  log "timeout ${timeout_s}s reached; stopping QEMU"
fi
# EXIT trap runs cleanup
