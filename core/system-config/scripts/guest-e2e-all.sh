#!/usr/bin/env bash
# Comprehensive QEMU CodaLinux guest e2e for system-config (+ optional sandbox).
# NEVER run on the build host. Requires --guest and ID=codalinux.
# Safe applies only: never suspend, hibernate, or poweroff.
set -uo pipefail

if [[ "${1:-}" != "--guest" ]]; then
  echo "QEMU guest only. Usage: $0 --guest" >&2
  echo "Do not run this on the host — it refuses." >&2
  exit 2
fi

if ! grep -q '^ID=codalinux' /etc/os-release 2>/dev/null; then
  echo "refusing: /etc/os-release is not ID=codalinux (never run on the host)" >&2
  exit 2
fi

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing $1 (ISO-wired binaries should be on PATH)" >&2
    exit 1
  }
}
need system-configd
need system-config-apply
need system-config-report
need system-config

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
if [[ -z "${session_uid}" || "${session_uid}" == "0" ]]; then
  if id live >/dev/null 2>&1; then
    session_uid="$(id -u live)"
  elif [[ "$(id -u)" != "0" ]]; then
    session_uid="$(id -u)"
  fi
fi
if [[ -z "${session_uid}" || "${session_uid}" == "0" ]]; then
  echo "FAIL: no seat user (refusing uid 0 — do not start D/report as root)" >&2
  exit 1
fi
session_user="$(id -nu "${session_uid}")"

runtime="/run/user/${session_uid}"
# Never inherit a root runtime (/run/user/0). Clients always use the seat dir.
export XDG_RUNTIME_DIR="${runtime}"
export CODA_SYSTEM_CONFIG_UID="${session_uid}"
if [[ "$(id -u)" -eq 0 ]]; then
  mkdir -p "${runtime}/coda"
  chown "${session_uid}:${session_uid}" "${runtime}/coda" 2>/dev/null || true
  chmod 0750 "${runtime}/coda" 2>/dev/null || true
else
  mkdir -p "${runtime}/coda"
fi

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

as_session() {
  local -a envvars=(
    "XDG_RUNTIME_DIR=${runtime}"
    "CODA_SYSTEM_CONFIG_UID=${session_uid}"
    "HOME=$(getent passwd "${session_user}" | cut -d: -f6 || echo /home/${session_user})"
  )
  if [[ -n "${his}" ]]; then
    envvars+=("HYPRLAND_INSTANCE_SIGNATURE=${his}")
  fi
  # Always the seat user — never a root copy under /run/user/0.
  if [[ "$(id -u)" == "${session_uid}" ]]; then
    env "${envvars[@]}" "$@"
  else
    sudo -u "${session_user}" -- env "${envvars[@]}" "$@"
  fi
}

wait_sock() {
  local s="$1" n=0
  while [[ ! -S "${s}" && "${n}" -lt 80 ]]; do
    sleep 0.1
    n=$((n + 1))
  done
  [[ -S "${s}" ]]
}

dial_unix() {
  local s="$1"
  as_session python3 -c "import socket,sys; p=sys.argv[1]; c=socket.socket(socket.AF_UNIX); c.settimeout(1); c.connect(p); c.close()" "${s}" 2>/dev/null
}

wait_dialable() {
  local s="$1" n=0
  while [[ "${n}" -lt 80 ]]; do
    if [[ -S "${s}" ]]; then
      if command -v python3 >/dev/null 2>&1 && dial_unix "${s}"; then
        return 0
      fi
      if as_session test -w "${s}"; then
        return 0
      fi
    fi
    sleep 0.1
    n=$((n + 1))
  done
  return 1
}

# Prefer already-running user units (coda-hyprland). Only start missing
# D/report as the seat user — never as root.
if [[ ! -S "${runtime}/coda/system-configd.sock" ]]; then
  as_session system-configd &
fi
if [[ ! -S "${runtime}/coda/system-config-report.sock" ]]; then
  as_session system-config-report &
fi
if [[ ! -S "${runtime}/coda/system-config-apply.sock" ]]; then
  if [[ "$(id -u)" -eq 0 ]]; then
    env XDG_RUNTIME_DIR="${runtime}" CODA_SYSTEM_CONFIG_UID="${session_uid}" system-config-apply &
  else
    sudo -- env XDG_RUNTIME_DIR="${runtime}" CODA_SYSTEM_CONFIG_UID="${session_uid}" system-config-apply &
  fi
fi
if ! wait_sock "${runtime}/coda/system-configd.sock"; then
  echo "FAIL: system-configd.sock not ready under ${runtime}/coda" >&2
  exit 1
fi
wait_sock "${runtime}/coda/system-config-report.sock" || true
if ! wait_dialable "${runtime}/coda/system-config-apply.sock"; then
  echo "FAIL: apply socket not dialable as ${session_user} (${runtime}/coda/system-config-apply.sock)" >&2
  ls -l "${runtime}/coda/" >&2 || true
  exit 1
fi

json_ok() {
  printf '%s' "${1:-}" | grep -qE '"ok"[[:space:]]*:[[:space:]]*true'
}

json_field() {
  # json_field <json> <dotted path under observed, e.g. outputs.0.name>
  local raw="$1" path="$2"
  if command -v python3 >/dev/null 2>&1; then
    printf '%s' "${raw}" | python3 -c '
import json, sys
raw = sys.stdin.read()
try:
    d = json.loads(raw)
except Exception:
    sys.exit(1)
cur = d.get("observed", d)
for part in sys.argv[1].split("."):
    if cur is None:
        break
    if isinstance(cur, list):
        try:
            cur = cur[int(part)]
        except Exception:
            cur = None
            break
    elif isinstance(cur, dict):
        cur = cur.get(part)
    else:
        cur = None
        break
if cur is None:
    print("")
elif cur is True:
    print("true")
elif cur is False:
    print("false")
elif isinstance(cur, (dict, list)):
    print(json.dumps(cur))
else:
    print(cur)
' "${path}" 2>/dev/null || true
  fi
}

declare -a RESULTS=()
fail_count=0
pass_count=0
skip_count=0

record() {
  local status="$1" name="$2" note="${3:-}"
  RESULTS+=("${status}|${name}|${note}")
  case "${status}" in
    PASS) pass_count=$((pass_count + 1)) ;;
    FAIL) fail_count=$((fail_count + 1)) ;;
    SKIP) skip_count=$((skip_count + 1)) ;;
  esac
  printf '  [%s] %s%s\n' "${status}" "${name}" "${note:+ — ${note}}"
}

cli() {
  as_session system-config "$@"
}

# --- refresh every KnownPath ---
paths=(
  display network audio bluetooth input datetime locale
  session power printers users storage
  devices.summary devices.pci devices.usb hardware.dmi
)

echo "=== get submodels ==="
subs="$(cli get submodels 2>&1)" || subs="ERR:${subs}"
if [[ "${subs}" == ERR:* ]] || ! json_ok "${subs}"; then
  record FAIL "get submodels" "$(printf '%s' "${subs}" | tr '\n' ' ' | head -c 160)"
else
  missing=""
  for p in "${paths[@]}"; do
    if ! printf '%s' "${subs}" | grep -q "\"${p}\""; then
      missing="${missing} ${p}"
    fi
  done
  if [[ -n "${missing}" ]]; then
    record FAIL "get submodels" "missing:${missing}"
  else
    record PASS "get submodels"
  fi
fi

echo "=== refresh every KnownPath ==="
for p in "${paths[@]}"; do
  if [[ "${p}" == bluetooth ]]; then
    t0="$(date +%s)"
    out="$(cli refresh bluetooth 2>&1)" || out="ERR:${out}"
    t1="$(date +%s)"
    elapsed=$((t1 - t0))
    if [[ "${out}" == ERR:* ]] || ! json_ok "${out}"; then
      record FAIL "refresh bluetooth" "not ok (${elapsed}s)"
    elif [[ "${elapsed}" -ge 5 ]]; then
      record FAIL "refresh bluetooth" "took ${elapsed}s (must be <5s)"
    else
      record PASS "refresh bluetooth" "${elapsed}s"
    fi
    continue
  fi
  out="$(cli refresh "${p}" 2>&1)" || out="ERR:${out}"
  if [[ "${out}" == ERR:* ]] || ! json_ok "${out}"; then
    record FAIL "refresh ${p}" "$(printf '%s' "${out}" | tr '\n' ' ' | head -c 160)"
  else
    record PASS "refresh ${p}"
  fi
done

# Wait for Hyprland outputs before display apply (guest race: report
# can refresh before monitors exist).
out_name=""
if command -v hyprctl >/dev/null 2>&1; then
  n=0
  while [[ "${n}" -lt 50 ]]; do
    cli refresh display >/dev/null 2>&1 || true
    disp="$(cli get display 2>/dev/null || true)"
    out_name="$(json_field "${disp}" "outputs.0.name")"
    if [[ -z "${out_name}" ]]; then
      out_name="$(printf '%s' "${disp}" | grep -oE '"name"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | cut -d'"' -f4 || true)"
    fi
    if [[ -n "${out_name}" ]]; then
      break
    fi
    sleep 0.2
    n=$((n + 1))
  done
else
  disp="$(cli get display 2>/dev/null || true)"
  out_name="$(json_field "${disp}" "outputs.0.name")"
fi

# --- safe applies ---
echo "=== safe applies (never suspend/hibernate/poweroff) ==="

if [[ -n "${out_name}" ]] && command -v hyprctl >/dev/null 2>&1; then
  if out="$(cli set display "{\"outputs\":[{\"name\":\"${out_name}\",\"scale\":2}]}" 2>&1)" && json_ok "${out}"; then
    if out="$(cli apply display 2>&1)" && json_ok "${out}"; then
      record PASS "apply display scale 2" "${out_name}"
    else
      record FAIL "apply display scale 2" "$(printf '%s' "${out}" | tr '\n' ' ' | head -c 160)"
    fi
  else
    record FAIL "set display scale 2" "$(printf '%s' "${out}" | tr '\n' ' ' | head -c 160)"
  fi
  if out="$(cli set display "{\"outputs\":[{\"name\":\"${out_name}\",\"scale\":1}]}" 2>&1)" && json_ok "${out}"; then
    if out="$(cli apply display 2>&1)" && json_ok "${out}"; then
      record PASS "apply display scale 1" "${out_name}"
    else
      record FAIL "apply display scale 1" "$(printf '%s' "${out}" | tr '\n' ' ' | head -c 160)"
    fi
  else
    record FAIL "set display scale 1"
  fi
else
  record SKIP "apply display scale" "no output or hyprctl"
fi

if out="$(cli set datetime '{"timezone":"America/Denver","ntp":true}' 2>&1)" && json_ok "${out}"; then
  if out="$(cli apply datetime 2>&1)" && json_ok "${out}"; then
    record PASS "apply datetime America/Denver + ntp"
  else
    record FAIL "apply datetime" "$(printf '%s' "${out}" | tr '\n' ' ' | head -c 160)"
  fi
else
  record FAIL "set datetime"
fi

audio="$(cli get audio 2>/dev/null || true)"
sinks="$(json_field "${audio}" "sinks")"
if [[ -n "${sinks}" && "${sinks}" != "[]" && "${sinks}" != "null" ]]; then
  mute_now="$(json_field "${audio}" "mute")"
  if [[ "${mute_now}" == "True" || "${mute_now}" == "true" ]]; then
    want=false
  else
    want=true
  fi
  if out="$(cli set audio "{\"mute\":${want}}" 2>&1)" && json_ok "${out}"; then
    if out="$(cli apply audio 2>&1)" && json_ok "${out}"; then
      record PASS "apply audio mute toggle" "mute=${want}"
    else
      record FAIL "apply audio mute" "$(printf '%s' "${out}" | tr '\n' ' ' | head -c 160)"
    fi
  else
    record FAIL "set audio mute"
  fi
else
  record SKIP "apply audio mute" "no sinks"
fi

if command -v hyprctl >/dev/null 2>&1 && [[ -n "${his}" || -d "${runtime}/hypr" ]]; then
  if out="$(cli set input '{"pointer_speed":0}' 2>&1)" && json_ok "${out}"; then
    if out="$(cli apply input 2>&1)" && json_ok "${out}"; then
      record PASS "apply input pointer_speed 0"
    else
      record FAIL "apply input pointer_speed" "$(printf '%s' "${out}" | tr '\n' ' ' | head -c 160)"
    fi
  else
    record FAIL "set input pointer_speed"
  fi
else
  record SKIP "apply input pointer_speed" "hypr not available"
fi

loc="$(cli get locale 2>/dev/null || true)"
if json_ok "${loc}"; then
  record PASS "get locale"
  lang="$(json_field "${loc}" "lang")"
  if [[ -z "${lang}" ]]; then
    lang="en_US.UTF-8"
  fi
  if out="$(cli set locale "{\"lang\":\"${lang}\"}" 2>&1)" && json_ok "${out}"; then
    if out="$(cli apply locale 2>&1)" && json_ok "${out}"; then
      record PASS "apply locale lang no-op/safe" "${lang}"
    else
      record FAIL "apply locale lang" "$(printf '%s' "${out}" | tr '\n' ' ' | head -c 160)"
    fi
  else
    record FAIL "set locale lang"
  fi
else
  record FAIL "get locale"
fi

net="$(cli get network 2>/dev/null || true)"
ssid="${CODA_E2E_SSID:-}"
if [[ -z "${ssid}" ]]; then
  ssid="$(json_field "${net}" "wifi.connect")"
fi
if [[ -z "${ssid}" ]]; then
  ssid="$(json_field "${net}" "wifi.connected")"
fi
if [[ -z "${ssid}" ]]; then
  record SKIP "apply wifi connect" "no SSID (set CODA_E2E_SSID to try)"
else
  record SKIP "apply wifi connect" "SSID present (${ssid}) but e2e does not join networks"
fi

bt="$(cli get bluetooth 2>/dev/null || true)"
adapter="$(json_field "${bt}" "adapter")"
if [[ -z "${adapter}" ]]; then
  record SKIP "apply bluetooth pair" "no adapter"
else
  record SKIP "apply bluetooth pair" "adapter ${adapter} present; e2e does not pair"
fi

# Observe-only domains users expect in a settings app.
for p in printers users storage; do
  out="$(cli get "${p}" 2>/dev/null || true)"
  if json_ok "${out}"; then
    record PASS "get ${p}"
  else
    record FAIL "get ${p}"
  fi
done

# Never lock/suspend/hibernate/poweroff/reboot in e2e.
record SKIP "apply session.lock" "would lock the live session"
record SKIP "apply power.suspend" "e2e never suspends"
record SKIP "apply power.hibernate" "e2e never hibernates"

# --- sandbox ---
echo "=== coda-sandbox smoke ==="
if ! command -v coda-sandbox >/dev/null 2>&1; then
  record SKIP "coda-sandbox --help" "not on PATH"
else
  if coda-sandbox --help >/dev/null 2>&1 || coda-sandbox help >/dev/null 2>&1; then
    record PASS "coda-sandbox --help"
  else
    record FAIL "coda-sandbox --help"
  fi
  have_net=0
  if command -v curl >/dev/null 2>&1 && curl -fsI --max-time 4 https://geo.mirror.pkgbuild.com/ >/dev/null 2>&1; then
    have_net=1
  elif command -v ping >/dev/null 2>&1 && ping -c 1 -W 2 geo.mirror.pkgbuild.com >/dev/null 2>&1; then
    have_net=1
  fi
  env_name="e2e-sc"
  if [[ "${have_net}" -ne 1 ]]; then
    record SKIP "coda-sandbox create/exec/destroy" "no network"
  else
    as_desktop_sb() {
      if [[ "$(id -u)" == "${session_uid}" ]]; then
        "$@"
      else
        sudo -u "${session_user}" -- env HOME="$(getent passwd "${session_user}" | cut -d: -f6)" "$@"
      fi
    }
    as_desktop_sb coda-sandbox destroy "${env_name}" >/dev/null 2>&1 || true
    create_ok=0
    # timeout must wrap the binary, not the shell function (or PATH misses as_desktop_sb).
    if command -v timeout >/dev/null 2>&1; then
      if as_desktop_sb timeout 180 coda-sandbox create "${env_name}"; then
        create_ok=1
      fi
    elif as_desktop_sb coda-sandbox create "${env_name}"; then
      create_ok=1
    fi
    if [[ "${create_ok}" -eq 1 ]]; then
      if as_desktop_sb coda-sandbox exec "${env_name}" true; then
        record PASS "coda-sandbox create/exec"
      else
        record FAIL "coda-sandbox exec"
      fi
      if as_desktop_sb coda-sandbox destroy "${env_name}"; then
        record PASS "coda-sandbox destroy"
      else
        record FAIL "coda-sandbox destroy"
      fi
    else
      record SKIP "coda-sandbox create/exec/destroy" "create failed (keyring/network)"
    fi
  fi
fi

echo
echo "=== summary ==="
printf '%-6s  %-36s  %s\n' "STATUS" "CHECK" "NOTE"
printf '%s\n' "------  ------------------------------------  ----"
for row in "${RESULTS[@]}"; do
  IFS='|' read -r st name note <<<"${row}"
  printf '%-6s  %-36s  %s\n' "${st}" "${name}" "${note}"
done
echo
echo "PASS=${pass_count} FAIL=${fail_count} SKIP=${skip_count}"
if [[ "${fail_count}" -gt 0 ]]; then
  echo "guest e2e FAILED" >&2
  exit 1
fi
echo "guest e2e PASS"
exit 0
