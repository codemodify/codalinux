#!/usr/bin/env bash
# Host-safe checks for coda-desktop-mount session start after merge.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=coda-desktop-mount
. "${root}/scripts/coda-desktop-mount"

fail=0
work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

log_fail() { printf 'coda-desktop-mount_test: %s\n' "$*" >&2; fail=1; }

mock_bin="${work}/bin"
mkdir -p "${mock_bin}" "${work}/data" "${work}/greetd-ok"
printf '#!/bin/sh\nexit 0\n' >"${work}/greetd-ok/greetd"
chmod 0755 "${work}/greetd-ok/greetd"

cat >"${mock_bin}/systemctl" <<'EOF'
#!/bin/sh
log="${CODA_SYSTEMCTL_LOG:?}"
printf '%s\n' "$*" >>"${log}"
cmd="$1"
case "${cmd}" in
  is-active) exit 1 ;;
  cat)
    if [ "${CODA_SYSTEMCTL_CAT_FAIL:-0}" = 1 ]; then
      exit 1
    fi
    exit 0
    ;;
  daemon-reload) exit 0 ;;
  start) exit 0 ;;
  *) exit 0 ;;
esac
EOF
chmod 0755 "${mock_bin}/systemctl"

export PATH="${mock_bin}:${PATH}"
export CODA_DATA_DIR="${work}/data"
export CODA_GREETD_BIN="${work}/greetd-ok/greetd"

# Happy path: greetd unit known, binary present → start --no-block.
export CODA_SYSTEMCTL_LOG="${work}/start.log"
: >"${CODA_SYSTEMCTL_LOG}"
coda_start_merged_session
if ! grep -qx 'start --no-block greetd.service' "${CODA_SYSTEMCTL_LOG}"; then
  log_fail "expected systemctl start --no-block greetd.service"
  cat "${CODA_SYSTEMCTL_LOG}" >&2 || true
fi
if grep -q 'daemon-reload' "${CODA_SYSTEMCTL_LOG}"; then
  log_fail "must not daemon-reload when greetd.service is already known"
fi

# Late unit: cat fails → daemon-reload then start.
export CODA_SYSTEMCTL_CAT_FAIL=1
export CODA_SYSTEMCTL_LOG="${work}/reload.log"
: >"${CODA_SYSTEMCTL_LOG}"
coda_start_merged_session
if ! grep -qx 'daemon-reload' "${CODA_SYSTEMCTL_LOG}"; then
  log_fail "expected daemon-reload when greetd.service is unknown"
fi
if ! grep -qx 'start --no-block greetd.service' "${CODA_SYSTEMCTL_LOG}"; then
  log_fail "expected start after daemon-reload"
fi
unset CODA_SYSTEMCTL_CAT_FAIL

# Missing binary: do not start greetd (core continues).
export CODA_GREETD_BIN="${work}/missing-greetd"
export CODA_SYSTEMCTL_LOG="${work}/missing.log"
: >"${CODA_SYSTEMCTL_LOG}"
coda_start_merged_session
if grep -q 'start --no-block' "${CODA_SYSTEMCTL_LOG}"; then
  log_fail "must not start greetd when the binary is missing"
fi

# Skip flag.
export CODA_SKIP_SESSION_START=1
export CODA_GREETD_BIN="${work}/greetd-ok/greetd"
export CODA_SYSTEMCTL_LOG="${work}/skip.log"
: >"${CODA_SYSTEMCTL_LOG}"
coda_start_merged_session
if [[ -s "${CODA_SYSTEMCTL_LOG}" ]]; then
  log_fail "CODA_SKIP_SESSION_START must not call systemctl"
fi
unset CODA_SKIP_SESSION_START

# Live ISO / no coda-data: no-op.
export CODA_DATA_DIR="${work}/no-such-data"
export CODA_SYSTEMCTL_LOG="${work}/nodata.log"
: >"${CODA_SYSTEMCTL_LOG}"
coda_start_merged_session
if [[ -s "${CODA_SYSTEMCTL_LOG}" ]]; then
  log_fail "missing /coda/data must not start greetd"
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "coda-desktop-mount_test: FAILED" >&2
  exit 1
fi
echo "coda-desktop-mount_test: OK"
exit 0
