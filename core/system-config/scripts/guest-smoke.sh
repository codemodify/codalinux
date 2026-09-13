#!/usr/bin/env bash
# QEMU CodaLinux guest smoke for system-config display apply/report.
# NEVER run on the build host. Requires --guest and ID=codalinux.
set -euo pipefail

if [[ "${1:-}" != "--guest" ]]; then
  echo "QEMU guest only. Usage: $0 --guest" >&2
  echo "Do not run this on the host or it will not touch Hyprland there anyway — it refuses." >&2
  exit 2
fi

if ! grep -q '^ID=codalinux' /etc/os-release 2>/dev/null; then
  echo "refusing: /etc/os-release is not ID=codalinux (never run on the host)" >&2
  exit 2
fi

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing $1 (copy core/system-config binaries into the guest PATH)" >&2
    exit 1
  }
}
need system-configd
need system-config-apply
need system-config-report
need system-config
need hyprctl

# Graphical session: loginctl wayland user, else live, else current.
session_uid=""
session_user=""
if command -v loginctl >/dev/null 2>&1; then
  while read -r sid rest; do
    [[ -n "${sid:-}" ]] || continue
    typ="$(loginctl show-session "${sid}" -p Type --value 2>/dev/null || true)"
    class="$(loginctl show-session "${sid}" -p Class --value 2>/dev/null || true)"
    uid="$(loginctl show-session "${sid}" -p UID --value 2>/dev/null || true)"
    name="$(loginctl show-session "${sid}" -p Name --value 2>/dev/null || true)"
    if [[ "${typ}" == wayland && "${class}" == user && -n "${uid}" ]]; then
      session_uid="${uid}"
      session_user="${name}"
      break
    fi
  done < <(loginctl list-sessions --no-legend --no-pager 2>/dev/null | awk '{print $1}')
fi
if [[ -z "${session_uid}" ]]; then
  if id live >/dev/null 2>&1; then
    session_uid="$(id -u live)"
  else
    session_uid="$(id -u)"
  fi
fi
session_user="$(id -nu "${session_uid}")"

runtime="/run/user/${session_uid}"
if [[ -d "${XDG_RUNTIME_DIR:-}" && "${XDG_RUNTIME_DIR}" == "${runtime}" ]]; then
  runtime="${XDG_RUNTIME_DIR}"
fi
export XDG_RUNTIME_DIR="${runtime}"
export CODA_SYSTEM_CONFIG_UID="${session_uid}"
mkdir -p "${runtime}/coda"

his=""
if [[ -d "${runtime}/hypr" ]]; then
  for d in "${runtime}/hypr"/*; do
    [[ -d "${d}" ]] || continue
    his="$(basename "${d}")"
    break
  done
fi
export HYPRLAND_INSTANCE_SIGNATURE="${his}"

echo "session ${session_user} uid=${session_uid} XDG_RUNTIME_DIR=${runtime} HIS=${his:-<discover>}"
echo "hypr instances:"
ls -d "${runtime}/hypr"/* 2>/dev/null || echo "  (none under ${runtime}/hypr — apply/report will scan /run/user/*/hypr)"

as_session() {
  local -a envvars=(
    "XDG_RUNTIME_DIR=${runtime}"
    "CODA_SYSTEM_CONFIG_UID=${session_uid}"
  )
  if [[ -n "${his}" ]]; then
    envvars+=("HYPRLAND_INSTANCE_SIGNATURE=${his}")
  fi
  if [[ "$(id -u)" == "${session_uid}" ]]; then
    env "${envvars[@]}" "$@"
  else
    sudo -u "${session_user}" -- env "${envvars[@]}" "$@"
  fi
}

# D + report as the session user; apply as root with the same socket dir so D can connect.
if [[ ! -S "${runtime}/coda/system-configd.sock" ]]; then
  as_session system-configd &
  sleep 0.2
fi
if [[ ! -S "${runtime}/coda/system-config-report.sock" ]]; then
  as_session system-config-report &
  sleep 0.2
fi
if [[ ! -S "${runtime}/coda/system-config-apply.sock" ]]; then
  if [[ "$(id -u)" -eq 0 ]]; then
    env XDG_RUNTIME_DIR="${runtime}" CODA_SYSTEM_CONFIG_UID="${session_uid}" system-config-apply &
  else
    sudo -- env XDG_RUNTIME_DIR="${runtime}" CODA_SYSTEM_CONFIG_UID="${session_uid}" system-config-apply &
  fi
  sleep 0.2
fi

echo "=== refresh display (expect Virtual-1 in observed.outputs) ==="
as_session system-config refresh display
disp="$(as_session system-config get display)"
printf '%s\n' "${disp}"
printf '%s\n' "${disp}" | grep -q Virtual-1 || {
  echo "FAIL: observed.outputs missing Virtual-1" >&2
  exit 1
}

echo "=== set scale 2 + apply ==="
as_session system-config set display '{"outputs":[{"name":"Virtual-1","scale":2}]}'
as_session system-config apply display

echo "=== hyprctl monitors (expect scale 2) ==="
mon="$(as_session hyprctl -j monitors)"
printf '%s\n' "${mon}"
printf '%s\n' "${mon}" | grep -E '"scale"[[:space:]]*:[[:space:]]*2' >/dev/null || {
  echo "FAIL: hyprctl monitors scale is not 2" >&2
  exit 1
}

echo "=== other domains (observe; do not fail smoke if tools missing) ==="
for p in network audio bluetooth input datetime locale devices.usb hardware.dmi session power; do
  as_session system-config refresh "${p}" || true
  as_session system-config get "${p}" || true
done

echo "guest smoke ok"
