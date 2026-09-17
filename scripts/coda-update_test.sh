#!/usr/bin/env bash
# Host-safe CLI tests for coda-update: usage, parsing, refuses running slot.
# No root, no pacman, no QEMU, no network.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
upd="${root}/scripts/coda-update"
lib="${root}/scripts/coda-install-lib.sh"
fail=0

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

good="${work}/good-lib.sh"
cp -a "${lib}" "${good}"

run_upd() {
  local rc=0
  # shellcheck disable=SC2086
  CODA_UPDATE_LIB_PATHS="${CODA_UPDATE_LIB_PATHS:-${good}}" \
    CODA_PACKAGES_DIR="${root}/packages" \
    CODA_UPDATE_DRY_RUN="${CODA_UPDATE_DRY_RUN:-0}" \
    CODA_UPDATE_TEST_ACTIVE="${CODA_UPDATE_TEST_ACTIVE:-}" \
    bash "${upd}" "$@" >"${work}/out" 2>"${work}/err" || rc=$?
  return "${rc}"
}

expect_ok() {
  if ! "$@"; then
    echo "coda-update_test: expected success: $*" >&2
    cat "${work}/out" >&2 || true
    cat "${work}/err" >&2 || true
    fail=1
  fi
}

expect_fail() {
  if "$@"; then
    echo "coda-update_test: expected failure: $*" >&2
    cat "${work}/out" >&2 || true
    cat "${work}/err" >&2 || true
    fail=1
    return 1
  fi
  return 0
}

# --- usage / help ---
expect_ok run_upd --help
if ! grep -q 'coda-update core' "${work}/out"; then
  echo "coda-update_test: --help missing coda-update core" >&2
  fail=1
fi
if ! grep -q 'coda-update desktop' "${work}/out"; then
  echo "coda-update_test: --help missing desktop" >&2
  fail=1
fi
if ! grep -q -- '--promote' "${work}/out"; then
  echo "coda-update_test: --help missing --promote" >&2
  fail=1
fi
if ! grep -q 'status' "${work}/out"; then
  echo "coda-update_test: --help missing status" >&2
  fail=1
fi
if ! grep -q 'Do not' "${work}/out" || ! grep -q 'pacman -Syu' "${work}/out"; then
  echo "coda-update_test: --help must reject host pacman -Syu" >&2
  fail=1
fi
if ! grep -q 'oneshot' "${work}/out"; then
  echo "coda-update_test: --help must explain oneshot vs promote" >&2
  fail=1
fi

expect_ok run_upd -h
expect_ok run_upd help
expect_ok run_upd

# --- argument parsing ---
expect_fail run_upd core core
if ! grep -q 'duplicate command' "${work}/err"; then
  echo "coda-update_test: duplicate command must be reported" >&2
  fail=1
fi

expect_fail run_upd --not-a-flag
if ! grep -q 'unknown argument' "${work}/err"; then
  echo "coda-update_test: unknown flag must be reported" >&2
  fail=1
fi

expect_fail run_upd frobnicate
if ! grep -q 'unknown argument' "${work}/err"; then
  echo "coda-update_test: unknown command must be reported" >&2
  fail=1
fi

# --- refuses running slot (dry-run, no root) ---
if CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_TEST_ACTIVE=a \
    CODA_UPDATE_LIB_PATHS="${good}" \
    bash "${upd}" core --slot a >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: core --slot a while running a must fail" >&2
  fail=1
else
  if ! grep -q 'refusing to write the running slot (a)' "${work}/err"; then
    echo "coda-update_test: missing refuse-running-slot message" >&2
    cat "${work}/err" >&2 || true
    fail=1
  fi
fi

# Inactive slot is allowed in dry-run.
if ! CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_TEST_ACTIVE=a \
    CODA_UPDATE_LIB_PATHS="${good}" \
    bash "${upd}" core --slot b >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: core --slot b while running a must dry-run OK" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi
if ! grep -q 'inactive slot b' "${work}/out"; then
  echo "coda-update_test: dry-run core must name slot b" >&2
  cat "${work}/out" >&2 || true
  fail=1
fi
if ! grep -q 'coda-update core --promote' "${work}/out"; then
  echo "coda-update_test: dry-run core must point at --promote" >&2
  fail=1
fi

# Flag order: options before command.
if ! CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_TEST_ACTIVE=a \
    CODA_UPDATE_LIB_PATHS="${good}" \
    bash "${upd}" --slot b --disk /dev/vda core >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: --slot/--disk before core must parse" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi

# --from-iso dry-run
if ! CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_TEST_ACTIVE=a \
    CODA_UPDATE_LIB_PATHS="${good}" \
    bash "${upd}" core --from-iso --slot b >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: --from-iso dry-run failed" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi
if ! grep -q 'from-iso' "${work}/out"; then
  echo "coda-update_test: --from-iso dry-run must mention ISO path" >&2
  fail=1
fi

# --no-boot-test
if ! CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_TEST_ACTIVE=b \
    CODA_UPDATE_LIB_PATHS="${good}" \
    bash "${upd}" core --no-boot-test >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: --no-boot-test dry-run failed" >&2
  fail=1
fi
if ! grep -q 'skip oneshot' "${work}/out"; then
  echo "coda-update_test: --no-boot-test must skip oneshot" >&2
  cat "${work}/out" >&2 || true
  fail=1
fi

# --- promote parsing ---
if CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_TEST_ACTIVE=live \
    CODA_UPDATE_LIB_PATHS="${good}" \
    bash "${upd}" core --promote >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: --promote from live ISO must fail" >&2
  fail=1
else
  if ! grep -qi 'reboot into the new slot' "${work}/err"; then
    echo "coda-update_test: --promote from live must explain reboot" >&2
    cat "${work}/err" >&2 || true
    fail=1
  fi
fi

if ! CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_TEST_ACTIVE=b \
    CODA_UPDATE_LIB_PATHS="${good}" \
    bash "${upd}" core --promote >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: --promote while running on b must dry-run OK" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi
if ! grep -q 'would promote running slot b' "${work}/out"; then
  echo "coda-update_test: --promote dry-run must name slot b" >&2
  cat "${work}/out" >&2 || true
  fail=1
fi

if CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_TEST_ACTIVE=a \
    CODA_UPDATE_LIB_PATHS="${good}" \
    bash "${upd}" core --promote --slot b >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: --promote --slot b while on a must fail" >&2
  fail=1
else
  if ! grep -q 'refusing to promote' "${work}/err"; then
    echo "coda-update_test: missing refuse-promote-other-slot message" >&2
    cat "${work}/err" >&2 || true
    fail=1
  fi
fi

# --- desktop dry-run ---
if ! CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_LIB_PATHS="${good}" \
    CODA_PACKAGES_DIR="${root}/packages" \
    bash "${upd}" desktop >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: desktop --dry-run failed" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi
if ! grep -q 'desktop packages' "${work}/out"; then
  echo "coda-update_test: desktop dry-run must mention desktop packages" >&2
  cat "${work}/out" >&2 || true
  fail=1
fi
if ! grep -q '/home' "${work}/out"; then
  echo "coda-update_test: desktop dry-run must say /home is left alone" >&2
  fail=1
fi

if CODA_UPDATE_DRY_RUN=1 CODA_UPDATE_LIB_PATHS="${good}" \
    CODA_PACKAGES_DIR="${root}/packages" \
    bash "${upd}" desktop --promote >"${work}/out" 2>"${work}/err"; then
  echo "coda-update_test: desktop --promote must fail" >&2
  fail=1
fi

# --- lib loader (same stub story as coda-slot) ---
stub_return="${work}/stub-return.sh"
printf '%s\n' 'return 0' >"${stub_return}"
empty="${work}/empty-lib.sh"
: >"${empty}"

if ! CODA_UPDATE_LIB_PATHS="${stub_return} ${good}" bash "${upd}" --help >/dev/null 2>"${work}/err"; then
  echo "coda-update_test: return-0 stub must not skip a later good lib" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi
if ! CODA_UPDATE_LIB_PATHS="${empty} ${good}" bash "${upd}" --help >/dev/null 2>"${work}/err"; then
  echo "coda-update_test: empty first file must not skip a later good lib" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi
if CODA_UPDATE_LIB_PATHS="${empty} ${stub_return}" bash "${upd}" --help >/dev/null 2>"${work}/err"; then
  echo "coda-update_test: no-good-candidate must fail" >&2
  fail=1
else
  if ! grep -q 'did not provide coda_need_root' "${work}/err"; then
    echo "coda-update_test: missing load-failure message" >&2
    cat "${work}/err" >&2 || true
    fail=1
  fi
fi

if awk '
  /coda_update_load_lib\(\)/ { in_fn=1 }
  in_fn && /^\s*\.\s+"/ { src_in_fn=1 }
  in_fn && /^}/ { in_fn=0 }
  END { exit src_in_fn ? 0 : 1 }
' "${upd}"; then
  echo "coda-update_test: coda-update must not source the lib from inside a function" >&2
  fail=1
fi

# --- package lists via lib (no pacman) ---
# shellcheck source=coda-install-lib.sh
. "${lib}"
core_names="$(coda_core_package_names | tr '\n' ' ')"
if [[ "${core_names}" != *linux* || "${core_names}" != *rsync* ]]; then
  echo "coda-update_test: core package names missing linux/rsync: ${core_names}" >&2
  fail=1
fi
if [[ "${core_names}" == *hyprland* || "${core_names}" == *firefox* ]]; then
  echo "coda-update_test: core package names must not include desktop pkgs" >&2
  fail=1
fi
desk_names="$(coda_desktop_package_names | tr '\n' ' ')"
if [[ "${desk_names}" != *hyprland* || "${desk_names}" != *bubblewrap* ]]; then
  echo "coda-update_test: desktop package names missing hyprland/bubblewrap" >&2
  fail=1
fi

if ( coda_refuse_running_slot a a ) >/dev/null 2>"${work}/err"; then
  echo "coda-update_test: coda_refuse_running_slot a a must fail" >&2
  fail=1
else
  if ! grep -q 'refusing to write the running slot (a)' "${work}/err"; then
    echo "coda-update_test: coda_refuse_running_slot message missing" >&2
    fail=1
  fi
fi
if ! ( coda_refuse_running_slot b a ); then
  echo "coda-update_test: coda_refuse_running_slot b a must succeed" >&2
  fail=1
fi

if ( coda_pacman_into_root / -Q ) >/dev/null 2>"${work}/err"; then
  echo "coda-update_test: pacman --root / must be refused" >&2
  fail=1
else
  if ! grep -q 'running root' "${work}/err"; then
    echo "coda-update_test: pacman --root / must mention running root" >&2
    cat "${work}/err" >&2 || true
    fail=1
  fi
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "coda-update_test: FAILED" >&2
  exit 1
fi
echo "coda-update_test: OK"
exit 0
