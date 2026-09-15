#!/usr/bin/env bash
# Host-safe checks for coda-install-lib.sh (no root, no pacstrap).
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=coda-install-lib.sh
. "${root}/scripts/coda-install-lib.sh"

fail=0
dest="$(mktemp -d)"
trap 'rm -rf "${dest}"' EXIT

# Core-only slots have /etc but not the empty mkinitcpio.d package dir.
mkdir -p "${dest}/etc"
if [[ -d "${dest}/etc/mkinitcpio.d" ]]; then
  echo "coda-install-lib_test: fixture must start without mkinitcpio.d" >&2
  exit 1
fi

coda_write_linux_preset "${dest}" a
if [[ ! -f "${dest}/etc/mkinitcpio.d/linux.preset" ]]; then
  echo "coda-install-lib_test: linux.preset was not written" >&2
  fail=1
fi
if ! grep -q "ALL_kver=\"/boot/coda/a/vmlinuz-linux\"" "${dest}/etc/mkinitcpio.d/linux.preset"; then
  echo "coda-install-lib_test: linux.preset contents wrong" >&2
  fail=1
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "coda-install-lib_test: FAILED" >&2
  exit 1
fi
echo "coda-install-lib_test: OK"
exit 0
