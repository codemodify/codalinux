#!/usr/bin/env bash
# Host-safe checks for coda-install-lib.sh (no root, no pacstrap).
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=coda-install-lib.sh
. "${root}/scripts/coda-install-lib.sh"

fail=0

# Stale env guard must not skip reload when helpers are missing.
unset -f coda_need_root
CODA_INSTALL_LIB_SOURCED=1
# shellcheck source=coda-install-lib.sh
. "${root}/scripts/coda-install-lib.sh"
if ! declare -F coda_need_root >/dev/null 2>&1; then
  echo "coda-install-lib_test: stale CODA_INSTALL_LIB_SOURCED skipped helper reload" >&2
  fail=1
fi
if ! CODA_INSTALL_LIB_SOURCED=1 bash "${root}/scripts/coda-slot" --help >/dev/null; then
  echo "coda-install-lib_test: coda-slot --help failed with stale CODA_INSTALL_LIB_SOURCED" >&2
  fail=1
fi
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
cp -a "${src}/usr/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst" \
  "${dest}/usr/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst"

# Stale dest tree must be replaced (not hardlink-merged) on resync.
if command -v rsync >/dev/null 2>&1; then
  mkdir -p "${dest}/usr/lib/modules/${kver}/kernel/fs/fat"
  printf 'STALE\n' >"${dest}/usr/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst"
  printf 'FRESH\n' >"${src}/usr/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst"
  coda_sync_kernel_modules "${src}" "${dest}"
  if ! grep -qx 'FRESH' "${dest}/usr/lib/modules/${kver}/kernel/fs/fat/vfat.ko.zst"; then
    echo "coda-install-lib_test: resync left a stale vfat.ko.zst" >&2
    fail=1
  fi
fi

# Arch linux 7.2 ships ext4 built-in (no ext4.ko). Assert must accept modules.builtin.
rm -f "${dest}/usr/lib/modules/${kver}/kernel/fs/ext4/ext4.ko.zst"
printf 'kernel/fs/ext4/ext4.ko\n' >"${dest}/usr/lib/modules/${kver}/modules.builtin"
if ! coda_assert_kernel_modules "${dest}" "${kver}"; then
  echo "coda-install-lib_test: assert should accept built-in ext4" >&2
  fail=1
fi
rm -f "${dest}/usr/lib/modules/${kver}/modules.builtin"
if ( coda_assert_kernel_modules "${dest}" "${kver}" ) >/dev/null 2>&1; then
  echo "coda-install-lib_test: assert should fail when ext4 is neither a .ko nor built-in" >&2
  fail=1
fi
rm -rf "${src}"

# Live ISO /boot/loader is not the disk ESP unless coda-*.conf is there.
esp_work="$(mktemp -d)"
mkdir -p "${esp_work}/boot/loader/entries"
if coda_pick_esp "${esp_work}" >/dev/null 2>&1; then
  echo "coda-install-lib_test: coda_pick_esp must fail without coda-*.conf" >&2
  fail=1
fi
: >"${esp_work}/boot/loader/entries/coda-b.conf"
picked="$(coda_pick_esp "${esp_work}")"
if [[ "${picked}" != "${esp_work}/boot" ]]; then
  echo "coda-install-lib_test: coda_pick_esp=${picked} (want ${esp_work}/boot)" >&2
  fail=1
fi
rm -rf "${esp_work}"

# greetd unit + PAM on the slot (not a dangling /usr/lib symlink).
coda_install_slot_greetd "${dest}"
if [[ ! -f "${dest}/etc/systemd/system/greetd.service" ]]; then
  echo "coda-install-lib_test: greetd.service was not written to the slot" >&2
  fail=1
fi
if [[ -L "${dest}/etc/systemd/system/greetd.service" ]]; then
  echo "coda-install-lib_test: greetd.service must not be a symlink" >&2
  fail=1
fi
if ! grep -q '^ExecStart=/usr/bin/greetd' "${dest}/etc/systemd/system/greetd.service"; then
  echo "coda-install-lib_test: greetd.service missing ExecStart" >&2
  fail=1
fi
if [[ ! -f "${dest}/etc/systemd/system/greetd.service.d/coda-desktop-mount.conf" ]]; then
  echo "coda-install-lib_test: greetd drop-in missing" >&2
  fail=1
fi
if ! grep -q 'After=coda-desktop-mount.service' \
    "${dest}/etc/systemd/system/greetd.service.d/coda-desktop-mount.conf"; then
  echo "coda-install-lib_test: greetd drop-in must After=coda-desktop-mount" >&2
  fail=1
fi
if grep -q 'ConditionPathExists' \
    "${dest}/etc/systemd/system/greetd.service.d/coda-desktop-mount.conf"; then
  echo "coda-install-lib_test: greetd drop-in must not ConditionPathExists (skips before merge)" >&2
  fail=1
fi

# Confext must not re-apply live dangling /usr/lib wants or coda-live-setup.
desk="$(mktemp -d)"
mkdir -p "${desk}/etc/systemd/system/greetd.service.d" \
  "${desk}/etc/systemd/system/multi-user.target.wants" \
  "${desk}/etc/systemd/system/graphical.target.wants"
printf '[Unit]\nWants=coda-live-setup.service\n' \
  >"${desk}/etc/systemd/system/greetd.service.d/coda.conf"
ln -sfn /usr/lib/systemd/system/greetd.service \
  "${desk}/etc/systemd/system/display-manager.service"
ln -sfn /usr/lib/systemd/system/greetd.service \
  "${desk}/etc/systemd/system/multi-user.target.wants/greetd.service"
coda_wipe_live_bits "${desk}"
if [[ -e "${desk}/etc/systemd/system/greetd.service.d/coda.conf" ]]; then
  echo "coda-install-lib_test: live greetd drop-in must be wiped from desktop" >&2
  fail=1
fi
if [[ -L "${desk}/etc/systemd/system/display-manager.service" ]]; then
  echo "coda-install-lib_test: dangling /usr/lib display-manager must be wiped from desktop" >&2
  fail=1
fi
rm -rf "${desk}"
if [[ ! -f "${dest}/etc/pam.d/greetd" ]]; then
  echo "coda-install-lib_test: /etc/pam.d/greetd missing on the slot" >&2
  fail=1
fi
want_link="$(readlink "${dest}/etc/systemd/system/multi-user.target.wants/greetd.service")"
if [[ "${want_link}" != /etc/systemd/system/greetd.service ]]; then
  echo "coda-install-lib_test: greetd wants is ${want_link} (want /etc/systemd/system/greetd.service)" >&2
  fail=1
fi
dm_link="$(readlink "${dest}/etc/systemd/system/display-manager.service")"
if [[ "${dm_link}" != /etc/systemd/system/greetd.service ]]; then
  echo "coda-install-lib_test: display-manager.service is ${dm_link}" >&2
  fail=1
fi
if [[ -x "${dest}/usr/bin/greetd" || -x "${dest}/usr/local/bin/coda-hyprland" ]]; then
  echo "coda-install-lib_test: slot greetd helper must not copy session binaries" >&2
  fail=1
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "coda-install-lib_test: FAILED" >&2
  exit 1
fi
echo "coda-install-lib_test: OK"
exit 0
