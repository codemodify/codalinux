#!/usr/bin/env bash
# Host-side QEMU + QGA e2e for install layout + A/B update (abox).
# Local ISO only. No GitHub artifacts. No system-config host tests.
#
# One command, no prompts:
#   ./scripts/qemu-install-e2e.sh
#   ./scripts/qemu-install-e2e.sh --phase all     # steps 1–8 (default)
#   ./scripts/qemu-install-e2e.sh --phase install # steps 1–4 only
#
# In the guest, CODA_INSTALL_DISK=auto picks the first disk (/dev/vda).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

iso="${CODA_ISO:-}"
out_dir="${CODA_QEMU_INSTALL_OUT:-${root}/out/qemu-install-e2e}"
ram="${CODA_QEMU_RAM:-8G}"
cpus="${CODA_QEMU_CPUS:-}"
disk_size="${CODA_QEMU_DISK:-32G}"
phase="all"
dry_run=0
build_iso=0
keep_vm=0
timeout_qga="${CODA_QEMU_QGA_TIMEOUT:-180}"
timeout_install="${CODA_QEMU_INSTALL_TIMEOUT:-2700}"
timeout_hypr="${CODA_QEMU_HYPR_TIMEOUT:-240}"

qemu_pid=""
cleaned=0
declare -a SUMMARY=()

usage() {
  cat <<'EOF'
Usage: qemu-install-e2e.sh [options] [ISO]

Automated install + A/B update (QEMU/KVM + OVMF + qemu-guest-agent).
Uses the newest out/codalinux-*.iso unless ISO / $CODA_ISO is set.
Never downloads a GitHub ISO artifact.

Steps (--phase all, default):
  1. Pick first guest disk (CODA_INSTALL_DISK=auto)
  2. Auto-partition ESP + OS-A + OS-B + data
  3. Offline install into OS-A from the live ISO
  4. Reboot from disk → Hyprland as user/1 on slot A
  5. Write live payload + kernel/boot into inactive slot B
  6. Oneshot-boot slot B and verify Hyprland (default still A)
  7. Promote B as systemd-boot default
  8. Reboot and confirm running from B

  --phase install   stop after step 4
  --build           ./scripts/build-iso.sh if no local ISO exists
  --dry-run         print the plan and QEMU command; do not start
  --keep            leave the last VM running on success
  --disk-size SIZE  blank virtio disk (default 32G)
  --out DIR         artifacts (default out/qemu-install-e2e)

Success signal after a disk boot:
  /etc/coda/slot matches the expected slot, /home and /var bind
  coda-data, greetd autologin user, Hyprland instance under
  /run/user/<uid>/hypr and a Hyprland process for user.

Minimum disk: ~22 GiB (8G+8G slots for the full desktop). QEMU uses 32G.
EOF
}

log() { printf 'qemu-install-e2e: %s\n' "$*"; }
die() { printf 'qemu-install-e2e: %s\n' "$*" >&2; exit 1; }
step() {
  local name="$1"
  SUMMARY+=("${name}")
  log "==== ${name} ===="
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --phase)
      phase="$2"
      shift 2
      ;;
    --build) build_iso=1; shift ;;
    --dry-run) dry_run=1; shift ;;
    --keep) keep_vm=1; shift ;;
    --disk-size)
      disk_size="$2"
      shift 2
      ;;
    --ram)
      ram="$2"
      shift 2
      ;;
    --cpus)
      cpus="$2"
      shift 2
      ;;
    --out)
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

case "${phase}" in
  all|install) ;;
  *) die "--phase must be all or install" ;;
esac

find_iso() {
  if [[ -n "${iso}" ]]; then
    [[ -f "${iso}" || "${dry_run}" -eq 1 ]] || die "ISO not found: ${iso}"
    printf '%s' "${iso}"
    return 0
  fi
  local newest="" candidate
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
  if [[ "${build_iso}" -eq 1 && "${dry_run}" -ne 1 ]]; then
    log "no local ISO; running scripts/build-iso.sh"
    "${root}/scripts/build-iso.sh"
    find_iso
    return 0
  fi
  if [[ "${dry_run}" -eq 1 ]]; then
    printf '%s' "${root}/out/codalinux-YYYY.MM.DD-x86_64.iso"
    return 0
  fi
  die "no ${root}/out/codalinux-*.iso (build on abox with ./scripts/build-iso.sh or pass --build)"
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
  local sock="$1" payload="$2"
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

qga() {
  local payload="$1"
  python3 - "${qga_sock}" "${payload}" <<'PY'
import json, socket, sys

path, payload = sys.argv[1], sys.argv[2]
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.settimeout(60)
s.connect(path)
s.sendall(payload.encode() + b"\n")
buf = b""
dec = json.JSONDecoder()
while True:
    chunk = s.recv(65536)
    if not chunk:
        break
    buf += chunk
    text = buf.decode()
    try:
        obj, _ = dec.raw_decode(text.lstrip())
        print(json.dumps(obj))
        if "error" in obj:
            raise SystemExit(obj["error"].get("desc", str(obj)))
        sys.exit(0)
    except json.JSONDecodeError:
        continue
raise SystemExit("qga: no JSON reply")
PY
}

qga_exec() {
  local cmd="$1"
  local timeout_s="${2:-300}"
  python3 - "${qga_sock}" "${cmd}" "${timeout_s}" <<'PY'
import base64, json, socket, sys, time

path, cmd, timeout_s = sys.argv[1], sys.argv[2], int(sys.argv[3])

def rpc(payload, timeout=60):
    s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    s.settimeout(timeout)
    s.connect(path)
    s.sendall(json.dumps(payload).encode() + b"\n")
    buf = b""
    dec = json.JSONDecoder()
    while True:
        chunk = s.recv(65536)
        if not chunk:
            raise SystemExit("qga closed")
        buf += chunk
        try:
            obj, _ = dec.raw_decode(buf.decode().lstrip())
            break
        except json.JSONDecodeError:
            continue
    s.close()
    if "error" in obj:
        raise SystemExit(obj["error"].get("desc", str(obj)))
    return obj.get("return", obj)

start = rpc({
    "execute": "guest-exec",
    "arguments": {
        "path": "/bin/bash",
        "arg": ["-lc", cmd],
        "capture-output": True,
    },
})
pid = start["pid"]
deadline = time.time() + timeout_s
status = {}
while time.time() < deadline:
    status = rpc({"execute": "guest-exec-status", "arguments": {"pid": pid}})
    if status.get("exited"):
        break
    time.sleep(2)
else:
    raise SystemExit(f"guest-exec timed out after {timeout_s}s: {cmd}")

def dec(key):
    raw = status.get(key) or ""
    if not raw:
        return ""
    return base64.b64decode(raw).decode("utf-8", "replace")

out, err = dec("out-data"), dec("err-data")
sys.stdout.write(out)
sys.stderr.write(err)
rc = int(status.get("exitcode") or 1)
raise SystemExit(rc)
PY
}

qemu_alive() {
  [[ -n "${qemu_pid}" ]] && kill -0 "${qemu_pid}" 2>/dev/null
}

stop_qemu() {
  if qemu_alive; then
    qga '{"execute":"guest-shutdown"}' >/dev/null 2>&1 || true
    local n=0
    while qemu_alive && [[ "${n}" -lt 40 ]]; do
      sleep 1
      n=$((n + 1))
    done
    if qemu_alive && [[ -S "${qmp_sock:-}" ]]; then
      qmp_cmd "${qmp_sock}" '{"execute":"quit"}' >/dev/null 2>&1 || true
      sleep 1
    fi
    if qemu_alive; then
      kill -TERM "${qemu_pid}" 2>/dev/null || true
      sleep 0.5
    fi
    if qemu_alive; then
      kill -KILL "${qemu_pid}" 2>/dev/null || true
    fi
    wait "${qemu_pid}" 2>/dev/null || true
  fi
  qemu_pid=""
}

cleanup() {
  [[ "${cleaned}" -eq 1 ]] && return 0
  cleaned=1
  if [[ "${keep_vm}" -eq 1 && "${ok_exit:-0}" -eq 1 ]]; then
    log "leaving QEMU running (--keep) pid ${qemu_pid}"
    return 0
  fi
  stop_qemu
}

trap cleanup EXIT INT TERM

wait_qga() {
  local timeout_s="${1:-${timeout_qga}}"
  local n=0
  while [[ "${n}" -lt "${timeout_s}" ]]; do
    qemu_alive || die "QEMU died while waiting for QGA; see ${serial_log}"
    if [[ -S "${qga_sock}" ]] && qga '{"execute":"guest-ping"}' >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
    n=$((n + 2))
  done
  die "qemu-guest-agent did not answer in ${timeout_s}s (rebuild ISO so live enables qemu-guest-agent.service). serial: ${serial_log}"
}

start_qemu() {
  local mode="$1"  # live | disk
  rm -f "${qmp_sock}" "${qga_sock}" "${pid_file}"
  local -a cmd=(
    qemu-system-x86_64
    -name "coda-qemu-install-e2e"
    "${accel_args[@]}"
    "${cpu_args[@]}"
    -smp "${cpus}"
    -m "${ram}"
    "${fw_args[@]}"
    -drive "file=${disk_img},if=virtio,format=qcow2,cache=writeback"
    -device virtio-vga,xres=1920,yres=1080
    -display none
    -device virtio-serial-pci
    -chardev "socket,path=${qga_sock},server=on,wait=off,id=qga0"
    -device virtserialport,chardev=qga0,name=org.qemu.guest_agent.0
    -device qemu-xhci
    -device usb-tablet
    -serial "file:${serial_log}"
    -debugcon "file:${out_dir}/ovmf-debug.log"
    -global isa-debugcon.iobase=0x402
    -qmp "unix:${qmp_sock},server,nowait"
    -pidfile "${pid_file}"
  )
  # Offline at install time: no NIC. QGA is virtio-serial.
  if [[ "${mode}" == live ]]; then
    cmd+=(-cdrom "${iso}" -boot order=d,menu=on)
  else
    cmd+=(-boot order=c,menu=on)
  fi
  {
    printf '%q ' "${cmd[@]}"
    printf '\n'
  } >"${cmd_file}.${mode}"
  if [[ "${dry_run}" -eq 1 ]]; then
    log "dry-run ${mode}:"
    cat "${cmd_file}.${mode}"
    return 0
  fi
  : >"${serial_log}"
  "${cmd[@]}" &
  qemu_pid=$!
  sleep 0.3
  if [[ -f "${pid_file}" ]]; then
    qemu_pid="$(<"${pid_file}")"
  fi
  qemu_alive || die "QEMU exited immediately (${mode}); see ${serial_log}"
  log "QEMU ${mode} pid ${qemu_pid}"
}

print_summary() {
  local rc="$1"
  echo
  echo "======== qemu-install-e2e summary ========"
  local i=1 s
  for s in "${SUMMARY[@]}"; do
    printf '  %d. %s\n' "${i}" "${s}"
    i=$((i + 1))
  done
  if [[ "${rc}" -eq 0 ]]; then
    echo "RESULT: GREEN"
  else
    echo "RESULT: RED"
    echo "serial: ${serial_log}"
  fi
  echo "artifacts: ${out_dir}"
  echo "=========================================="
}

# --- setup ---
if [[ "${dry_run}" -ne 1 ]]; then
  command -v qemu-system-x86_64 >/dev/null || die "qemu-system-x86_64 missing"
  command -v qemu-img >/dev/null || die "qemu-img missing"
  command -v python3 >/dev/null || die "python3 missing"
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
  log "warning: /dev/kvm not usable; TCG will be slow"
  accel_args=(-machine q35,accel=tcg)
fi

install -d "${out_dir}"
serial_log="${out_dir}/serial.log"
qmp_sock="${out_dir}/qmp.sock"
qga_sock="${out_dir}/qga.sock"
pid_file="${out_dir}/qemu.pid"
cmd_file="${out_dir}/qemu.cmd"
disk_img="${out_dir}/coda-install.qcow2"
vars_copy="${out_dir}/OVMF_VARS.fd"

fw_args=()
if [[ "${ovmf_mode}" == split ]]; then
  if [[ "${dry_run}" -ne 1 ]]; then
    cp -f "${ovmf_vars}" "${vars_copy}"
    chmod u+w "${vars_copy}"
  fi
  fw_args=(
    -drive "if=pflash,format=raw,readonly=on,file=${ovmf_code}"
    -drive "if=pflash,format=raw,file=${vars_copy}"
  )
else
  die "need split OVMF CODE+VARS so UEFI boot entries persist across reboots"
fi

log "ISO ${iso}"
log "phase ${phase}  disk ${disk_size}  ram ${ram}  cpus ${cpus}"
log "artifacts ${out_dir}"
log "offline: QEMU has no NIC (install + slot write use the ISO only)"

if [[ "${dry_run}" -eq 1 ]]; then
  start_qemu live
  start_qemu disk
  echo "dry-run OK (no VM started)"
  trap - EXIT INT TERM
  exit 0
fi

qemu-img create -f qcow2 "${disk_img}" "${disk_size}" >/dev/null

guest_need_new_iso() {
  if qga_exec 'test -x /usr/local/bin/coda-slot && test -x /usr/local/lib/codalinux/coda-install-ab.sh && test -x /usr/local/lib/codalinux/coda-install-layout.py' 30; then
    return 0
  fi
  die "live ISO is missing the A/B installer (coda-slot / coda-install-ab.sh). Rebuild on abox: ./scripts/build-iso.sh"
}

# 1–3 live install
step "1 pick first disk (CODA_INSTALL_DISK=auto)"
start_qemu live
wait_qga
guest_need_new_iso
first_disk="$(qga_exec 'lsblk -dnpo NAME,TYPE | awk "\$2==\"disk\"{print \$1; exit}"' 30)"
first_disk="$(printf '%s' "${first_disk}" | tr -d '[:space:]')"
[[ -n "${first_disk}" ]] || die "guest has no disk"
log "guest first disk: ${first_disk}"
SUMMARY[-1]="1 pick first disk (${first_disk})"

step "2 space-check + auto layout ESP+OS-A+OS-B+data"
qga_exec "python3 /usr/local/lib/codalinux/coda-install-layout.py check ${first_disk}" 60

step "3 offline install into OS-A (no NIC, no pacstrap)"
qga_exec "export CODA_INSTALL_DISK=${first_disk}; /usr/local/bin/coda-install --disk ${first_disk} --yes" "${timeout_install}"
qga_exec "umount -R /mnt 2>/dev/null || true; /usr/local/lib/codalinux/coda-install-verify.sh --layout" 120

step "4 reboot from disk into OS-A (Hyprland user/1)"
stop_qemu
start_qemu disk
wait_qga
qga_exec 'for i in $(seq 1 60); do test -d /run/user/1000/hypr -o -d /run/user/$(id -u user 2>/dev/null)/hypr && break; sleep 2; done; /usr/local/lib/codalinux/coda-install-verify.sh --boot --slot a' "${timeout_hypr}"
log "OS-A desktop OK"

if [[ "${phase}" == install ]]; then
  ok_exit=1
  print_summary 0
  exit 0
fi

# 5 write inactive slot from live ISO
step "5 upgrade inactive slot B from live ISO (kernel + root, not running A)"
stop_qemu
start_qemu live
wait_qga
qga_exec "export CODA_INSTALL_DISK=${first_disk}; /usr/local/bin/coda-slot --disk ${first_disk} install --slot b" "${timeout_install}"
qga_exec "test -f /mnt/coda-slot/boot/coda/b/vmlinuz-linux && test -f /mnt/coda-slot/boot/coda/b/initramfs-linux.img" 30 \
  || qga_exec "mkdir -p /mnt/coda-esp && mount /dev/disk/by-partlabel/coda-esp /mnt/coda-esp && test -f /mnt/coda-esp/coda/b/vmlinuz-linux && test -f /mnt/coda-esp/coda/b/initramfs-linux.img && umount /mnt/coda-esp" 60

step "6 oneshot-boot slot B (default remains A) and verify desktop"
qga_exec "export CODA_INSTALL_DISK=${first_disk}; /usr/local/bin/coda-slot --disk ${first_disk} boot-test --slot b" 60
# Proof that failure would keep A: default is still coda-a.conf after oneshot.
qga_exec 'esp=/mnt/coda-slot/boot; if [[ ! -f $esp/loader/loader.conf ]]; then mkdir -p /mnt/coda-esp; mount /dev/disk/by-partlabel/coda-esp /mnt/coda-esp; esp=/mnt/coda-esp; fi; grep -q "default coda-a.conf" $esp/loader/loader.conf' 30
stop_qemu
start_qemu disk
wait_qga
qga_exec 'for i in $(seq 1 60); do test -f /etc/coda/slot && break; sleep 2; done; /usr/local/lib/codalinux/coda-install-verify.sh --boot --slot b' "${timeout_hypr}"
# Default must still be A until promote.
qga_exec 'grep -q "default coda-a.conf" /boot/loader/loader.conf' 30
log "slot B boot-test OK; default still A"

step "7 promote slot B (systemd-boot default)"
qga_exec '/usr/local/bin/coda-slot promote --slot b' 60
qga_exec 'grep -q "default coda-b.conf" /boot/loader/loader.conf' 30

step "8 reboot and confirm promoted slot B"
stop_qemu
start_qemu disk
wait_qga
qga_exec 'for i in $(seq 1 60); do test -f /etc/coda/slot && break; sleep 2; done; /usr/local/lib/codalinux/coda-install-verify.sh --boot --slot b && grep -q "default coda-b.conf" /boot/loader/loader.conf' "${timeout_hypr}"
log "promoted slot B desktop OK"

ok_exit=1
print_summary 0
exit 0
