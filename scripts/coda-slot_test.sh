#!/usr/bin/env bash
# Host-safe regression: coda-slot must load helpers after a stub/empty
# first candidate (ISO layout after install). No root, no QEMU.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
slot="${root}/scripts/coda-slot"
lib="${root}/scripts/coda-install-lib.sh"
fail=0

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

good="${work}/good-lib.sh"
cp -a "${lib}" "${good}"

stub_return="${work}/stub-return.sh"
printf '%s\n' 'return 0' >"${stub_return}"

stub_guard="${work}/stub-guard.sh"
cat >"${stub_guard}" <<'EOF'
if declare -F coda_need_root >/dev/null 2>&1; then
  return 0
fi
CODA_INSTALL_LIB_SOURCED=1
EOF

empty="${work}/empty-lib.sh"
: >"${empty}"

dir_as_file="${work}/dir-as-lib.sh"
mkdir -p "${dir_as_file}"

iso_bin="${work}/iso/usr/local/bin"
mkdir -p "${iso_bin}"
cp -a "${slot}" "${iso_bin}/coda-slot"
chmod 0755 "${iso_bin}/coda-slot"

run_slot() {
  CODA_SLOT_LIB_PATHS="$1" bash "${iso_bin}/coda-slot" --help >/dev/null 2>"${work}/err"
}

if ! run_slot "${good}"; then
  echo "coda-slot_test: good lib alone must load" >&2
  fail=1
fi

if ! run_slot "${stub_return} ${good}"; then
  echo "coda-slot_test: return-0 stub must not skip a later good lib" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi

if ! run_slot "${stub_guard} ${good}"; then
  echo "coda-slot_test: guard-only stub must not skip a later good lib" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi

if ! run_slot "${empty} ${good}"; then
  echo "coda-slot_test: empty first file must not skip a later good lib" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi

if ! run_slot "${dir_as_file} ${good}"; then
  echo "coda-slot_test: directory at first path must not skip a later good lib" >&2
  cat "${work}/err" >&2 || true
  fail=1
fi

# ISO after install: first path emptied, share fallback still good.
if ! run_slot "${empty} ${good}"; then
  echo "coda-slot_test: post-install empty live path + fallback must load" >&2
  fail=1
fi

if run_slot "${empty} ${stub_return} ${stub_guard}"; then
  echo "coda-slot_test: no-good-candidate must fail" >&2
  fail=1
else
  if ! grep -q 'did not provide coda_need_root' "${work}/err"; then
    echo "coda-slot_test: missing load-failure message" >&2
    cat "${work}/err" >&2 || true
    fail=1
  fi
fi

# Simulate sourcing from a function (the old loader). A return-0 stub
# must not be treated as success when a later file is good — we check
# the top-level loop by running coda-slot, already covered above.
# Also assert the loader is not a for-loop inside a function that sources.
if awk '
  /coda_slot_load_lib\(\)/ { in_fn=1 }
  in_fn && /^\s*\.\s+"/ { src_in_fn=1 }
  in_fn && /^}/ { in_fn=0 }
  END { exit src_in_fn ? 0 : 1 }
' "${slot}"; then
  echo "coda-slot_test: coda-slot must not source the lib from inside a function" >&2
  fail=1
fi

# shellcheck source=coda-install-lib.sh
. "${lib}"

# Same-inode copy must not empty the live helper.
same="${work}/same-inode"
mkdir -p "${same}"
printf 'coda_need_root() { :; }\n' >"${same}/lib.sh"
ln -f "${same}/lib.sh" "${same}/alias.sh"
coda_copy_file_safe "${same}/lib.sh" "${same}/alias.sh"
if ! grep -q 'coda_need_root()' "${same}/lib.sh"; then
  echo "coda-slot_test: same-inode copy emptied the source helper" >&2
  fail=1
fi

# Restore after install emptied the live path (override via env by
# copying into a temp "live" tree and calling the restore helper).
snap="${work}/snap"
mkdir -p "${snap}" "${work}/live"
cp -a "${lib}" "${snap}/coda-install-lib.sh"
: >"${work}/emptied.sh"
# Restore writes to /usr/local/... which we cannot touch. Exercise
# coda_lib_file_has_need_root + copy_file_safe instead.
if coda_lib_file_has_need_root "${work}/emptied.sh"; then
  echo "coda-slot_test: empty file must not count as a helper" >&2
  fail=1
fi
if ! coda_lib_file_has_need_root "${snap}/coda-install-lib.sh"; then
  echo "coda-slot_test: snapshot of real lib must count as a helper" >&2
  fail=1
fi
coda_copy_file_safe "${snap}/coda-install-lib.sh" "${work}/emptied.sh"
if ! coda_lib_file_has_need_root "${work}/emptied.sh"; then
  echo "coda-slot_test: restore copy did not bring helpers back" >&2
  fail=1
fi

# Ground truth from abox e2e on 4dd58cfa: step 5 used source `/`
# (no /run/archiso/airootfs) then desktop --delete. Live helper must stay.
if command -v rsync >/dev/null 2>&1; then
  live="${work}/live-root"
  mkdir -p \
    "${live}/usr/local/lib/codalinux" \
    "${live}/usr/bin" \
    "${live}/mnt/coda-slot/coda/data/desktop/usr/local/lib/codalinux"
  cp -a "${lib}" "${live}/usr/local/lib/codalinux/coda-install-lib.sh"
  printf 'hypr\n' >"${live}/usr/bin/Hyprland"
  # Leftover from first-install copy_tree of /usr/local/lib onto desktop.
  cp -a "${lib}" \
    "${live}/mnt/coda-slot/coda/data/desktop/usr/local/lib/codalinux/coda-install-lib.sh"
  desk_list="${work}/desktop.list"
  printf '%s\n' usr/bin/Hyprland >"${desk_list}"
  coda_rsync_filelist \
    "${live}" \
    "${live}/mnt/coda-slot/coda/data/desktop" \
    "${desk_list}" \
    1
  if ! grep -q 'coda_need_root()' "${live}/usr/local/lib/codalinux/coda-install-lib.sh"; then
    echo "coda-slot_test: desktop --delete from live root emptied the live helper" >&2
    fail=1
  fi
  if [[ -e "${live}/mnt/coda-slot/coda/data/desktop/usr/local/lib/codalinux/coda-install-lib.sh" ]]; then
    echo "coda-slot_test: desktop --delete must drop leftover helper from dest" >&2
    fail=1
  fi
fi

src_got="$(coda_find_source)"
if [[ "${src_got}" != / ]]; then
  echo "coda-slot_test: host find_source=${src_got} (want / when archiso is absent)" >&2
  fail=1
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "coda-slot_test: FAILED" >&2
  exit 1
fi
echo "coda-slot_test: OK"
exit 0
