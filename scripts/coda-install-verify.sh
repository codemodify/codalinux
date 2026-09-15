#!/usr/bin/env bash
# Verify ESP+A+B+data layout and/or an installed Hyprland session.
# Guest-safe: Hyprland checks refuse unless ID=codalinux.
# Hyprland must come from /coda/data/desktop (not the slot root payload).
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: coda-install-verify.sh [--layout] [--boot] [--slot a|b] [--disk DEV]

  --layout   GPT PARTLABELs, fstypes, size floors
  --boot     running root is the expected slot; /home and /var binds;
             desktop payload on coda-data; greetd user; Hyprland as
             that user from data (ID=codalinux only)
  default    both, when running on an installed slot
EOF
}

want_layout=0
want_boot=0
expect_slot=""
disk="${CODA_INSTALL_DISK:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --layout) want_layout=1; shift ;;
    --boot) want_boot=1; shift ;;
    --slot) expect_slot="$2"; shift 2 ;;
    --disk)
      disk="$2"
      shift 2
      ;;
    *)
      echo "coda-install-verify: unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if [[ "${want_layout}" -eq 0 && "${want_boot}" -eq 0 ]]; then
  want_layout=1
  if [[ -f /etc/coda/slot ]]; then
    want_boot=1
  fi
fi

failed=0
fail() { printf 'VERIFY FAIL: %s\n' "$*" >&2; failed=1; }
pass() { printf 'VERIFY PASS: %s\n' "$*"; }

check_layout() {
  local label dev fstype
  for label in coda-esp coda-a coda-b coda-data; do
    if [[ ! -e "/dev/disk/by-partlabel/${label}" ]]; then
      fail "missing PARTLABEL=${label}"
      continue
    fi
    dev="$(readlink -f "/dev/disk/by-partlabel/${label}")"
    fstype="$(lsblk -ndo FSTYPE "${dev}" 2>/dev/null || true)"
    case "${label}" in
      coda-esp)
        [[ "${fstype}" == vfat || "${fstype}" == fat32 ]] \
          || fail "${label} fstype=${fstype} (want vfat)"
        ;;
      *)
        [[ "${fstype}" == ext4 ]] || fail "${label} fstype=${fstype} (want ext4)"
        ;;
    esac
    pass "${label} ${dev} ${fstype}"
  done
  python3 - <<'PY' || fail "size floors"
import os, subprocess, sys

def size(label):
    path = f"/dev/disk/by-partlabel/{label}"
    if not os.path.exists(path):
        raise SystemExit(f"missing {label}")
    out = subprocess.check_output(["blockdev", "--getsize64", path], text=True)
    return int(out.strip())

mib = 1024 * 1024
checks = [
    ("coda-esp", 900 * mib),
    ("coda-a", 3500 * mib),
    ("coda-b", 3500 * mib),
    ("coda-data", 7 * 1024 * mib),
]
rc = 0
for label, floor in checks:
    have = size(label)
    if have < floor:
        print(f"VERIFY FAIL: {label} is {have} bytes (< {floor})", file=sys.stderr)
        rc = 1
    else:
        print(f"VERIFY PASS: {label} size {have} (>= {floor})")
sys.exit(rc)
PY
}

check_boot() {
  if ! grep -q '^ID=codalinux' /etc/os-release 2>/dev/null; then
    echo "VERIFY FAIL: --boot requires ID=codalinux (guest only)" >&2
    failed=1
    return
  fi
  local slot
  slot="$(tr -d '[:space:]' </etc/coda/slot 2>/dev/null || true)"
  [[ -n "${slot}" ]] || fail "missing /etc/coda/slot"
  if [[ -n "${expect_slot}" && "${slot}" != "${expect_slot}" ]]; then
    fail "running slot is ${slot} (want ${expect_slot})"
  else
    pass "running slot ${slot}"
  fi
  local src
  src="$(findmnt -n -o SOURCE / || true)"
  if [[ "${src}" != *coda-${slot}* && "${src}" != *"/dev/disk/by-partlabel/coda-${slot}"* ]]; then
    # Accept mapper/partlabel via lsblk.
    local pl
    pl="$(lsblk -ndo PARTLABEL "${src}" 2>/dev/null || true)"
    if [[ "${pl}" != "coda-${slot}" ]]; then
      fail "/ PARTLABEL is ${pl:-${src}} (want coda-${slot})"
    else
      pass "/ is PARTLABEL=coda-${slot}"
    fi
  else
    pass "/ is ${src}"
  fi
  findmnt -n /home | grep -q /coda/data/home || fail "/home is not bind from /coda/data/home"
  findmnt -n /var | grep -q /coda/data/var || fail "/var is not bind from /coda/data/var"
  pass "/home and /var bind coda-data"
  if [[ ! -d /coda/data/desktop/usr ]]; then
    fail "missing /coda/data/desktop/usr (desktop must live on coda-data)"
  else
    pass "/coda/data/desktop/usr present"
  fi
  if [[ ! -e /coda/data/desktop/.coda-desktop-payload ]]; then
    fail "missing /coda/data/desktop/.coda-desktop-payload marker"
  else
    pass "desktop payload marker on coda-data"
  fi
  hypr_data=""
  for cand in \
    /coda/data/desktop/usr/bin/Hyprland \
    /coda/data/desktop/usr/bin/hyprland \
    /coda/data/desktop/usr/local/bin/coda-hyprland
  do
    if [[ -e "${cand}" ]]; then
      hypr_data="${cand}"
      break
    fi
  done
  if [[ -z "${hypr_data}" ]]; then
    fail "Hyprland missing from /coda/data/desktop (must not live only on the slot)"
  else
    pass "Hyprland on data: ${hypr_data}"
  fi
  usr_opts="$(findmnt -n -o OPTIONS /usr 2>/dev/null || true)"
  usr_src="$(findmnt -n -o SOURCE /usr 2>/dev/null || true)"
  if echo "${usr_src} ${usr_opts}" | grep -Eq 'overlay|sysext|coda/data/desktop'; then
    pass "/usr is merged from coda-data desktop (${usr_src})"
  else
    fail "/usr is not overlay/sysext from /coda/data/desktop (${usr_src})"
  fi
  local user="${CODA_INSTALL_USER:-user}"
  id "${user}" >/dev/null 2>&1 || fail "user ${user} missing"
  if [[ -f /etc/greetd/config.toml ]]; then
    grep -q "user = \"${user}\"" /etc/greetd/config.toml \
      || fail "greetd does not autologin ${user}"
    grep -q 'user = "live"' /etc/greetd/config.toml \
      && fail "greetd still points at live"
    pass "greetd autologin ${user}"
  else
    fail "greetd config missing"
  fi
  local uid runtime his=""
  uid="$(id -u "${user}")"
  runtime="/run/user/${uid}"
  if [[ -d "${runtime}/hypr" ]]; then
    for d in "${runtime}/hypr"/*; do
      [[ -d "${d}" ]] || continue
      his="$(basename "${d}")"
      break
    done
  fi
  if [[ -z "${his}" ]]; then
    fail "no Hyprland instance under ${runtime}/hypr (user ${user})"
  else
    pass "Hyprland HIS=${his} as ${user} (uid ${uid})"
  fi
  if ! pgrep -u "${user}" -f 'Hyprland|start-hyprland' >/dev/null 2>&1; then
    fail "no Hyprland/start-hyprland process for ${user}"
  else
    pass "Hyprland process running as ${user}"
  fi
}

if [[ "${want_layout}" -eq 1 ]]; then
  check_layout
fi
if [[ "${want_boot}" -eq 1 ]]; then
  check_boot
fi

if [[ "${failed}" -ne 0 ]]; then
  echo "coda-install-verify: FAILED" >&2
  exit 1
fi
echo "coda-install-verify: OK"
exit 0
