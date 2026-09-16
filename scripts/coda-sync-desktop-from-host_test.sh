#!/usr/bin/env bash
# Host-safe checks for AGS restart env discovery (no guest, no sudo).
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=coda-sync-desktop-from-host.sh
. "${root}/scripts/coda-sync-desktop-from-host.sh"

fail=0
log_fail() { printf 'coda-sync-desktop-from-host_test: %s\n' "$*" >&2; fail=1; }

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

export CODA_RUNTIME_ROOT="${work}/run"
export CODA_TMP_HYPR="${work}/tmp-hypr"
export CODA_TEST_UID=1000
export CODA_TEST_HOME="${work}/home/live"
export CODA_AGS_POLL_S=0.01
export CODA_AGS_RETRY_S=0
export CODA_AGS_START_WAIT_TICKS=50
export CODA_AGS_QUIT_WAIT_S=0
export CODA_AGS_KILL_WAIT_S=0
mkdir -p "${work}/run/0" "${work}/run/1000/hypr/live-sig-abc" \
  "${work}/home/live" "${work}/tmp-hypr"
touch "${work}/run/1000/wayland-1"
touch "${work}/run/1000/wayland-1.lock"

# tty2 root: XDG_RUNTIME_DIR=/run/user/0 must not win over the seat user.
export XDG_RUNTIME_DIR="${work}/run/0"
export WAYLAND_DISPLAY=wayland-0
unset HYPRLAND_INSTANCE_SIGNATURE || true

coda_desktop_session_env live
if [[ "${CODA_DS_RUNTIME}" != "${work}/run/1000" ]]; then
  log_fail "runtime must be seat /run/user/<uid>, got ${CODA_DS_RUNTIME}"
fi
if [[ "${CODA_DS_WAYLAND}" != "wayland-1" ]]; then
  log_fail "wayland must come from seat runtime, got ${CODA_DS_WAYLAND:-<none>}"
fi
if [[ "${CODA_DS_SIG}" != "live-sig-abc" ]]; then
  log_fail "HIS must come from seat hypr dir, got ${CODA_DS_SIG:-<none>}"
fi
if ! coda_session_display_ready; then
  log_fail "display should be ready when seat wayland socket exists"
fi

env_lines="$(coda_as_desktop_env_lines live)"
if printf '%s\n' "${env_lines}" | grep -qF "XDG_RUNTIME_DIR=${work}/run/0"; then
  log_fail "as_desktop env must not export root XDG_RUNTIME_DIR"
fi
if ! printf '%s\n' "${env_lines}" | grep -qx "XDG_RUNTIME_DIR=${work}/run/1000"; then
  log_fail "as_desktop env missing seat XDG_RUNTIME_DIR"
fi
if ! printf '%s\n' "${env_lines}" | grep -qx "WAYLAND_DISPLAY=wayland-1"; then
  log_fail "as_desktop env missing seat WAYLAND_DISPLAY"
fi
if ! printf '%s\n' "${env_lines}" | grep -qx "HYPRLAND_INSTANCE_SIGNATURE=live-sig-abc"; then
  log_fail "as_desktop env missing seat HIS"
fi

# Matching caller runtime may keep inherited HIS/wayland.
export XDG_RUNTIME_DIR="${work}/run/1000"
export WAYLAND_DISPLAY=wayland-inherited
export HYPRLAND_INSTANCE_SIGNATURE=from-shell
coda_desktop_session_env live
if [[ "${CODA_DS_WAYLAND}" != "wayland-inherited" || "${CODA_DS_SIG}" != "from-shell" ]]; then
  log_fail "matching seat runtime should keep inherited Wayland/HIS"
fi
export XDG_RUNTIME_DIR="${work}/run/0"
unset WAYLAND_DISPLAY HYPRLAND_INSTANCE_SIGNATURE || true

# Display not ready: no wayland socket.
rm -f "${work}/run/1000/wayland-1"
coda_desktop_session_env live
if coda_session_display_ready; then
  log_fail "display must not be ready without a wayland socket"
fi

# wait_ags_gone: already gone.
export CODA_TEST_AGS_PIDFILE="${work}/ags.pids"
: >"${CODA_TEST_AGS_PIDFILE}"
if ! coda_wait_ags_gone live 1; then
  log_fail "wait_ags_gone should succeed when no pids"
fi
printf '123\n' >"${CODA_TEST_AGS_PIDFILE}"
if coda_wait_ags_gone live 0; then
  log_fail "wait_ags_gone should time out when pids remain"
fi

# Restart: display not ready → clear error, no start.
touch "${work}/as_desktop.log"
as_desktop() {
  printf '%s\n' "$*" >>"${work}/as_desktop.log"
}
if coda_restart_ags live >/tmp/coda-ags-restart-noready.out 2>&1; then
  log_fail "restart must fail when display is not ready"
fi
if ! grep -q 'display not ready' /tmp/coda-ags-restart-noready.out; then
  log_fail "restart must log a clear display-not-ready error"
  cat /tmp/coda-ags-restart-noready.out >&2 || true
fi
if grep -q 'coda-ags' "${work}/as_desktop.log"; then
  log_fail "must not invoke coda-ags when display is not ready"
fi

# Restart: quit, wait, retry, then verify process as seat user.
touch "${work}/run/1000/wayland-1"
: >"${work}/as_desktop.log"
: >"${CODA_TEST_AGS_PIDFILE}"
printf '0\n' >"${work}/start.tries"
as_desktop() {
  local user="$1"
  shift
  printf '%s\n' "${user} $*" >>"${work}/as_desktop.log"
  case "${1:-}" in
    coda-ags)
      if [[ "${2:-}" == quit ]]; then
        : >"${CODA_TEST_AGS_PIDFILE}"
      else
        n="$(cat "${work}/start.tries")"
        n=$((n + 1))
        printf '%s\n' "${n}" >"${work}/start.tries"
        if [[ "${n}" -ge 2 ]]; then
          printf '4242\n' >"${CODA_TEST_AGS_PIDFILE}"
        fi
      fi
      ;;
    killall) : >"${CODA_TEST_AGS_PIDFILE}" ;;
  esac
}
if ! coda_restart_ags live >/tmp/coda-ags-restart-ok.out 2>&1; then
  log_fail "restart should succeed after retry"
  cat /tmp/coda-ags-restart-ok.out >&2 || true
fi
if ! grep -q 'coda-ags quit' "${work}/as_desktop.log"; then
  log_fail "restart must send coda-ags quit"
fi
starts="$(grep -cE '^live coda-ags$' "${work}/as_desktop.log" || true)"
if [[ "${starts}" -lt 2 ]]; then
  log_fail "expected at least two start attempts, got ${starts}"
fi
if ! grep -q 'AGS running as live' /tmp/coda-ags-restart-ok.out; then
  log_fail "restart must verify AGS as the desktop user"
fi
if [[ "$(cat "${CODA_TEST_AGS_PIDFILE}")" != "4242" ]]; then
  log_fail "verified pid file should list the seat-user AGS pid"
fi

# Restart exhausts retries when start never stays up.
: >"${work}/as_desktop.log"
: >"${CODA_TEST_AGS_PIDFILE}"
export CODA_AGS_START_TRIES=2
as_desktop() {
  local user="$1"
  shift
  printf '%s\n' "${user} $*" >>"${work}/as_desktop.log"
  case "${1:-}" in
    coda-ags)
      if [[ "${2:-}" == quit ]]; then
        : >"${CODA_TEST_AGS_PIDFILE}"
      fi
      ;;
  esac
  return 0
}
if coda_restart_ags live >/tmp/coda-ags-restart-fail.out 2>&1; then
  log_fail "restart must fail when AGS never stays up"
fi
if ! grep -q 'failed to stay up as live' /tmp/coda-ags-restart-fail.out; then
  log_fail "exhausted restart must log the seat user and failure"
fi
unset CODA_AGS_START_TRIES

# coda-ags run: clear error without a wayland socket; ok with one.
mkdir -p "${work}/rt" "${work}/bin"
printf '#!/bin/sh\necho ags-ran\n' >"${work}/bin/ags"
chmod 0755 "${work}/bin/ags"
export XDG_RUNTIME_DIR="${work}/rt"
export WAYLAND_DISPLAY=wayland-1
if CODA_AGS_BIN="${work}/bin/ags" HOME="${work}/home/live" \
    bash "${root}/scripts/coda-ags" run >/tmp/coda-ags-run.out 2>/tmp/coda-ags-run.err; then
  log_fail "coda-ags run must fail when the wayland socket is missing"
fi
if ! grep -q 'Wayland display wayland-1 not ready' /tmp/coda-ags-run.err; then
  log_fail "coda-ags run must name the missing display"
  cat /tmp/coda-ags-run.err >&2 || true
fi
touch "${work}/rt/wayland-1"
if ! CODA_AGS_BIN="${work}/bin/ags" HOME="${work}/home/live" \
    bash "${root}/scripts/coda-ags" run >/tmp/coda-ags-run-ok.out 2>/tmp/coda-ags-run-ok.err; then
  log_fail "coda-ags run should exec ags when the wayland socket exists"
fi
if ! grep -q 'ags-ran' /tmp/coda-ags-run-ok.out; then
  log_fail "coda-ags run did not reach the fake ags binary"
fi
# quit must not require a wayland socket (IPC to a running instance).
rm -f "${work}/rt/wayland-1"
if ! CODA_AGS_BIN="${work}/bin/ags" HOME="${work}/home/live" \
    bash "${root}/scripts/coda-ags" quit >/tmp/coda-ags-quit.out 2>/tmp/coda-ags-quit.err; then
  log_fail "coda-ags quit should not require a wayland socket"
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "coda-sync-desktop-from-host_test: FAILED" >&2
  exit 1
fi
echo "coda-sync-desktop-from-host_test: OK"
