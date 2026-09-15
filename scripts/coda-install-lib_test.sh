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

# usr-merge compat links (kmod resolves /lib/modules/$kver).
mkdir -p "${dest}/usr/lib" "${dest}/usr/bin" "${dest}/usr/sbin"
coda_ensure_usr_merge "${dest}"
for link in bin lib lib64 sbin; do
  if [[ ! -L "${dest}/${link}" ]]; then
    echo "coda-install-lib_test: missing usr-merge symlink ${link}" >&2
    fail=1
  fi
done
if [[ "$(readlink "${dest}/lib")" != "usr/lib" ]]; then
  echo "coda-install-lib_test: /lib should point at usr/lib" >&2
  fail=1
fi

# Directory rsync of the kernel module tree (not the files-only list).
src="$(mktemp -d)"
kver="7.2.6-arch2-1"
mkdir -p \
  "${src}/usr/lib/modules/${kver}/kernel/fs/fat" \
  "${src}/usr/lib/modules/${kver}/kernel/fs/ext4" \
  "${src}/usr/lib/modules/${kver}/kernel/drivers/virtio"
: >"${src}/usr/lib/modules/${kver}/vmlinuz"
: >"${src}/usr/lib/modules/${kver}/modules.dep"
: >"${src}/usr/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst"
: >"${src}/usr/lib/modules/${kver}/kernel/fs/fat/fat.ko.zst"
: >"${src}/usr/lib/modules/${kver}/kernel/fs/ext4/ext4.ko.zst"
: >"${src}/usr/lib/modules/${kver}/kernel/drivers/virtio/virtio_blk.ko.zst"
if command -v rsync >/dev/null 2>&1; then
  coda_sync_kernel_modules "${src}" "${dest}"
else
  mkdir -p "${dest}/usr/lib/modules"
  cp -a "${src}/usr/lib/modules/." "${dest}/usr/lib/modules/"
fi
coda_assert_kernel_modules "${dest}" "${kver}"
if [[ ! -f "${dest}/usr/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst" ]]; then
  echo "coda-install-lib_test: vfat.ko.zst was not copied onto the slot" >&2
  fail=1
fi
if [[ ! -e "${dest}/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst" ]]; then
  echo "coda-install-lib_test: /lib/modules does not resolve vfat via usr-merge" >&2
  fail=1
fi
rm -f "${dest}/usr/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst"
if ( coda_assert_kernel_modules "${dest}" "${kver}" ) >/dev/null 2>&1; then
  echo "coda-install-lib_test: assert should fail when vfat is missing" >&2
  fail=1
fi
rm -rf "${src}"

if [[ "${fail}" -ne 0 ]]; then
  echo "coda-install-lib_test: FAILED" >&2
  exit 1
fi
echo "coda-install-lib_test: OK"
exit 0
