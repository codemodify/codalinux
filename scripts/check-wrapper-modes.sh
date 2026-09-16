#!/usr/bin/env bash
# Smoke assert: live wrappers stay executable in git, airootfs, and
# archiso file_permissions (mkarchiso sets unlisted files to 644).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
failed=0

need_bins=(
  coda-ags
  coda-hypr-ws
  coda-hyprland
  coda-hyprlock
  coda-hyprpaper
  coda-wallpaper
  coda-install
  coda-slot
  coda-settings
  coda-sandbox
  coda-sync-desktop-from-host
  system-config-gui
)

log_fail() { printf 'check-wrapper-modes: %s\n' "$*" >&2; failed=1; }

for bin in "${need_bins[@]}"; do
  overlay="${root}/archiso/airootfs/usr/local/bin/${bin}"
  if [[ ! -f "${overlay}" ]]; then
    log_fail "missing airootfs wrapper: ${overlay}"
    continue
  fi
  if [[ ! -x "${overlay}" ]]; then
    log_fail "not executable (airootfs): ${overlay}"
  fi
  mode="$(stat -c '%a' "${overlay}" 2>/dev/null || stat -f '%OLp' "${overlay}")"
  if [[ "${mode}" != 755 && "${mode}" != 0755 ]]; then
    log_fail "airootfs mode ${mode} (want 755): ${overlay}"
  fi
  if ! grep -qF "[\"/usr/local/bin/${bin}\"]=\"0:0:755\"" \
      "${root}/archiso/profiledef.sh"; then
    log_fail "archiso/profiledef.sh file_permissions missing 755 for /usr/local/bin/${bin}"
  fi
done

for src in coda-ags coda-hyprland coda-hyprlock coda-hyprpaper coda-wallpaper \
           coda-hypr-ws coda-install coda-slot coda-settings coda-sandbox system-config-gui; do
  f="${root}/scripts/${src}"
  if [[ ! -f "${f}" ]]; then
    log_fail "missing source wrapper: ${f}"
    continue
  fi
  if [[ ! -x "${f}" ]]; then
    log_fail "not executable (scripts/): ${f}"
  fi
done

for helper_name in coda-install-config.py coda-pacman-init.sh coda-install-post.sh \
                   coda-install-lib.sh coda-install-layout.py coda-install-ab.sh \
                   coda-install-verify.sh coda-install-split.py coda-desktop-mount; do
  helper_src="${root}/scripts/${helper_name}"
  helper_overlay="${root}/archiso/airootfs/usr/local/lib/codalinux/${helper_name}"
  if [[ ! -f "${helper_src}" || ! -x "${helper_src}" ]]; then
    log_fail "missing executable scripts/${helper_name}"
  fi
  if ! grep -qF "[\"/usr/local/lib/codalinux/${helper_name}\"]=\"0:0:755\"" \
      "${root}/archiso/profiledef.sh"; then
    log_fail "profiledef.sh missing 755 for ${helper_name}"
  fi
  # airootfs/usr/local/lib is gitignored for vendor-ags output. Authored
  # helpers are force-added when present; build-iso.sh always copies them.
  if [[ -e "${helper_overlay}" && ! -x "${helper_overlay}" ]]; then
    log_fail "airootfs ${helper_name} exists but is not executable"
  fi
done
if ! grep -qF "[\"/usr/local/lib/codalinux/coda-pacman-init.sh\"]=\"0:0:755\"" \
    "${root}/archiso/profiledef.sh"; then
  log_fail "profiledef.sh missing 755 for coda-pacman-init.sh"
fi
if ! grep -qF "[\"/usr/local/lib/codalinux/coda-install-post.sh\"]=\"0:0:755\"" \
    "${root}/archiso/profiledef.sh"; then
  log_fail "profiledef.sh missing 755 for coda-install-post.sh"
fi
if ! grep -qF 'user: ${CODA_INSTALL_USER:-user}' "${root}/scripts/coda-install"; then
  log_fail "coda-install must print default login user"
fi
if ! grep -qF 'password: ${CODA_INSTALL_PASSWORD:-1}' "${root}/scripts/coda-install"; then
  log_fail "coda-install must print default password 1"
fi
if ! grep -q 'CODA_INSTALL_DISK' "${root}/scripts/coda-install"; then
  log_fail "coda-install must honor CODA_INSTALL_DISK"
fi
if ! grep -q 'disk=auto\|--first-disk\|auto|first' "${root}/scripts/coda-install"; then
  log_fail "coda-install must accept auto/first disk for unattended e2e"
fi
if [[ ! -x "${root}/scripts/qemu-install-e2e.sh" ]]; then
  log_fail "missing executable scripts/qemu-install-e2e.sh (host A/B e2e)"
fi
if ! python3 "${root}/scripts/coda-install-layout_test.py" >/tmp/coda-layout-test.out 2>&1; then
  log_fail "coda-install-layout_test.py failed"
  cat /tmp/coda-layout-test.out >&2 || true
fi
if ! python3 "${root}/scripts/coda-install-split_test.py" >/tmp/coda-split-test.out 2>&1; then
  log_fail "coda-install-split_test.py failed"
  cat /tmp/coda-split-test.out >&2 || true
fi
if ! bash "${root}/scripts/coda-install-lib_test.sh" >/tmp/coda-lib-test.out 2>&1; then
  log_fail "coda-install-lib_test.sh failed"
  cat /tmp/coda-lib-test.out >&2 || true
fi
if ! bash "${root}/scripts/coda-slot_test.sh" >/tmp/coda-slot-test.out 2>&1; then
  log_fail "coda-slot_test.sh failed"
  cat /tmp/coda-slot-test.out >&2 || true
fi
if ! bash "${root}/scripts/coda-desktop-mount_test.sh" >/tmp/coda-mount-test.out 2>&1; then
  log_fail "coda-desktop-mount_test.sh failed"
  cat /tmp/coda-mount-test.out >&2 || true
fi
if ! bash "${root}/scripts/coda-install-post_test.sh" >/tmp/coda-post-test.out 2>&1; then
  log_fail "coda-install-post_test.sh failed"
  cat /tmp/coda-post-test.out >&2 || true
fi
if ! grep -q 'mkdir -p "${dest}/etc/mkinitcpio.d"' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must mkdir mkinitcpio.d before writing linux.preset"
fi
for sh in coda-install coda-install-ab.sh coda-install-lib.sh coda-slot \
          coda-install-verify.sh qemu-install-e2e.sh coda-desktop-mount \
          coda-desktop-mount_test.sh coda-install-post.sh \
          coda-install-post_test.sh coda-slot_test.sh; do
  if ! bash -n "${root}/scripts/${sh}"; then
    log_fail "bash -n failed: scripts/${sh}"
  fi
done
if [[ ! -L "${root}/archiso/airootfs/etc/systemd/system/multi-user.target.wants/qemu-guest-agent.service" ]]; then
  log_fail "live ISO must enable qemu-guest-agent.service (QEMU e2e)"
fi
if ! grep -q 'enable qemu-guest-agent.service' \
    "${root}/archiso/airootfs/etc/systemd/system-preset/80-codalinux.preset"; then
  log_fail "preset must enable qemu-guest-agent.service"
fi
if grep -q 'raise NotImplementedError' "${root}/install/profiles/codalinux.py"; then
  log_fail "install/profiles/codalinux.py must not be a NotImplementedError stub"
fi
if ! grep -qE '^timeout [12]$' "${root}/archiso/efiboot/loader/loader.conf"; then
  log_fail "archiso/efiboot/loader/loader.conf timeout must be 1 or 2"
fi
if ! grep -qE '(^|[[:space:]])cow_spacesize=4G([[:space:]]|$)' \
    "${root}/archiso/efiboot/loader/entries/01-codalinux-linux.conf"; then
  log_fail "live kernel cmdline must set cow_spacesize=4G (sandbox create needs >256M)"
fi
ldcfg="${root}/archiso/airootfs/etc/systemd/system/ldconfig.service.d/coda.conf"
if [[ ! -f "${ldcfg}" ]]; then
  log_fail "missing ldconfig.service.d/coda.conf"
fi
if ! grep -qE '^ConditionNeedsUpdate=$' "${ldcfg}"; then
  log_fail "ldconfig.service.d/coda.conf must reset all Condition* first"
fi
if ! grep -qE '^ConditionFileNotEmpty=!/etc/ld\.so\.cache$' "${ldcfg}"; then
  log_fail "ldconfig.service.d/coda.conf must require ConditionFileNotEmpty=!/etc/ld.so.cache"
fi
if grep -qE '^ConditionFileNotEmpty=\|' "${ldcfg}"; then
  log_fail "ldconfig.service.d/coda.conf FileNotEmpty must not use | (triggering OR)"
fi
if [[ -e "${root}/archiso/airootfs/etc/systemd/system/multi-user.target.wants/pacman-init.service" ]]; then
  log_fail "pacman-init.service must not be in multi-user.target.wants"
fi
cust="${root}/archiso/airootfs/root/customize_airootfs.sh"
if [[ ! -f "${cust}" || ! -x "${cust}" ]]; then
  log_fail "missing executable root/customize_airootfs.sh"
fi
if ! grep -qF "[\"/root/customize_airootfs.sh\"]=\"0:0:755\"" \
    "${root}/archiso/profiledef.sh"; then
  log_fail "profiledef.sh missing 755 for customize_airootfs.sh"
fi

if grep -qE '^WantedBy=multi-user\.target' \
    "${root}/archiso/airootfs/etc/systemd/system/pacman-init.service"; then
  log_fail "pacman-init.service must not WantedBy=multi-user.target (blocks greetd)"
fi

if ! grep -qx 'swaybg' "${root}/packages/desktop.txt"; then
  log_fail "packages/desktop.txt must list swaybg (live/VM wallpaper fallback)"
fi

if ! grep -qx 'bubblewrap' "${root}/packages/sandbox.txt"; then
  log_fail "packages/sandbox.txt must list bubblewrap (coda-sandbox backbone)"
fi
if [[ ! -f "${root}/packages/core-slot.txt" ]]; then
  log_fail "missing packages/core-slot.txt (OS-A/B seed extras)"
fi
if ! grep -qx 'rsync' "${root}/packages/core-slot.txt"; then
  log_fail "packages/core-slot.txt must list rsync"
fi
if grep -Eiq '^(hyprland|greetd|firefox)$' "${root}/packages/core-slot.txt"; then
  log_fail "packages/core-slot.txt must not list desktop packages"
fi
if ! grep -q 'SLOT_FLOOR_MIB = 4096' "${root}/scripts/coda-install-layout.py"; then
  log_fail "coda-install-layout.py SLOT_FLOOR_MIB must be 4096 (core-only slots)"
fi
if ! grep -q 'coda-hyprland must not be installed on the core slot' \
    "${root}/scripts/coda-install-post.sh"; then
  log_fail "coda-install-post.sh must keep the core-slot coda-hyprland guard"
fi
if ! grep -q 'coda_install_slot_greetd' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must install a real greetd.service on the slot"
fi
if ! grep -q 'coda-install-lib.sh' "${root}/scripts/coda-install-split.py"; then
  log_fail "coda-install-split.py must keep coda-install-lib.sh on the core list"
fi
if ! grep -q 'coda_slot_load_lib' "${root}/scripts/coda-slot"; then
  log_fail "coda-slot must load helpers via coda_slot_load_lib"
fi
if ! grep -q 'coda_slot_lib_candidates' "${root}/scripts/coda-slot"; then
  log_fail "coda-slot must list load paths via coda_slot_lib_candidates"
fi
if ! grep -q '/usr/share/codalinux/install/coda-install-lib.sh' \
    "${root}/scripts/coda-slot"; then
  log_fail "coda-slot must fall back to /usr/share/codalinux/install/coda-install-lib.sh"
fi
if ! grep -q 'coda_snapshot_live_helpers' "${root}/scripts/coda-slot"; then
  log_fail "coda-slot install must snapshot live helpers before mutating the tree"
fi
if ! grep -q 'coda_restore_live_helpers_if_broken' "${root}/scripts/coda-slot"; then
  log_fail "coda-slot install must restore live helpers if install emptied them"
fi
if ! grep -q 'coda-install-lib.sh' "${root}/scripts/build-iso.sh" \
    || ! grep -q '/usr/share/codalinux/install/coda-install-lib.sh' \
      "${root}/scripts/build-iso.sh"; then
  log_fail "build-iso.sh must install coda-install-lib.sh under /usr/share (ISO fallback)"
fi
if ! grep -qF "[\"/usr/share/codalinux/install/coda-install-lib.sh\"]=\"0:0:755\"" \
    "${root}/archiso/profiledef.sh"; then
  log_fail "profiledef.sh missing 755 for /usr/share/codalinux/install/coda-install-lib.sh"
fi
if ! grep -q 'coda_copy_file_safe' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must copy helpers via temp+rename (no same-inode truncate)"
fi
if ! grep -q 'usr/share/codalinux/install/coda-install-lib.sh' \
    "${root}/scripts/coda-install-split.py"; then
  log_fail "coda-install-split.py must keep the /usr/share helper fallback on core"
fi
if ! grep -q 'coda_pick_esp' "${root}/scripts/coda-slot"; then
  log_fail "coda-slot boot-test/promote must use coda_pick_esp (not live ISO /boot/loader)"
fi
if grep -q 'if \[\[ -d /boot/loader \]\]' "${root}/scripts/coda-slot"; then
  log_fail "coda-slot must not treat live ISO /boot/loader as the disk ESP"
fi
if ! grep -q 'coda-install-lib.sh missing on the core slot' \
    "${root}/scripts/coda-install-post.sh"; then
  log_fail "coda-install-post.sh must require coda-install-lib.sh on the slot"
fi
if ! grep -q 'coda_install_slot_greetd' "${root}/scripts/coda-install-post.sh"; then
  log_fail "coda-install-post.sh must call coda_install_slot_greetd"
fi
if grep -q 'ln -sfn /usr/lib/systemd/system/greetd.service' \
    "${root}/scripts/coda-install-post.sh"; then
  log_fail "coda-install-post.sh must not want greetd via a dangling /usr/lib symlink"
fi
if ! grep -q 'start --no-block greetd.service' "${root}/scripts/coda-desktop-mount"; then
  log_fail "coda-desktop-mount must start greetd --no-block after merge"
fi
if ! grep -q 'daemon-reload' "${root}/scripts/coda-desktop-mount"; then
  log_fail "coda-desktop-mount must daemon-reload after merge before starting greetd"
fi
if grep -q '^ConditionPathExists=' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must not emit a greetd ConditionPathExists drop-in"
fi
if ! grep -q 'coda_scrub_live_greetd' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must scrub live greetd drop-in and dangling /usr/lib wants"
fi
if ! grep -q 'greetd.service is not active after desktop merge' \
    "${root}/scripts/coda-install-verify.sh"; then
  log_fail "coda-install-verify.sh must fail when greetd is inactive"
fi
if ! grep -q 'start-hyprland missing after desktop merge' \
    "${root}/scripts/coda-install-verify.sh"; then
  log_fail "coda-install-verify.sh must require start-hyprland after merge"
fi
if ! grep -q 'is_directory_entry' "${root}/scripts/coda-install-split.py"; then
  log_fail "coda-install-split.py must skip directory nodes (rsync recurse leak)"
fi
if ! grep -q 'coda_purge_leaked_desktop' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must purge leaked desktop files from the slot"
fi
if ! grep -q 'coda_ensure_usr_merge' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must recreate usr-merge /lib so kmod sees modules"
fi
if ! grep -q 'coda_sync_kernel_modules' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must copy usr/lib/modules onto the core slot"
fi
if ! grep -q 'coda_assert_kernel_modules' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must assert ext4/vfat/virtio modules before mkinitcpio"
fi
if ! grep -q 'coda_module_is_builtin' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must accept modules.builtin (Arch 7.2 ext4 is built-in)"
fi
if ! grep -q 'replace, no hardlinks' "${root}/scripts/coda-install-lib.sh"; then
  log_fail "coda-install-lib.sh must replace the slot modules tree (no rsync -H merge)"
fi
if ! grep -q 'USR_MERGE_LINKS' "${root}/scripts/coda-install-split.py"; then
  log_fail "coda-install-split.py must keep usr-merge symlinks on the core list"
fi
if grep -qE 'arg.: \[.-lc' "${root}/scripts/qemu-install-e2e.sh"; then
  log_fail "qemu-install-e2e.sh qga_exec must use bash -c, not bash -lc"
fi
if ! grep -q '"-c", cmd' "${root}/scripts/qemu-install-e2e.sh"; then
  log_fail "qemu-install-e2e.sh qga_exec must pass bash -c"
fi
if grep -q 'status.get("exitcode") or 1' "${root}/scripts/qemu-install-e2e.sh"; then
  log_fail "qemu-install-e2e.sh must not treat guest-exec exitcode 0 as missing"
fi
if ! grep -q 'coda-slot install --disk' "${root}/scripts/qemu-install-e2e.sh"; then
  log_fail "qemu-install-e2e.sh must call coda-slot install --disk (subcommand first)"
fi
if ! grep -q 'coda-slot boot-test --disk' "${root}/scripts/qemu-install-e2e.sh"; then
  log_fail "qemu-install-e2e.sh must call coda-slot boot-test --disk"
fi

for gui_rt in wayland libxkbcommon libx11 libxext libxrandr libxcursor; do
  if ! grep -qx "${gui_rt}" "${root}/packages/desktop.txt"; then
    log_fail "packages/desktop.txt must list ${gui_rt} (system-config-gui CGO runtime)"
  fi
done
if grep -qE '^export CGO_ENABLED=0' "${root}/scripts/install-system-config.sh"; then
  log_fail "install-system-config.sh must not force CGO_ENABLED=0 for system-config-gui"
fi
if ! grep -q 'CGO_ENABLED=1 go build' "${root}/scripts/install-system-config.sh"; then
  log_fail "install-system-config.sh must CGO_ENABLED=1 build system-config-gui"
fi
if ! grep -qF "[\"/usr/local/lib/codalinux/system-config-gui\"]=\"0:0:755\"" \
    "${root}/archiso/profiledef.sh"; then
  log_fail "profiledef.sh missing 755 for CGO system-config-gui binary"
fi
if ! grep -q 'UITK_PAINT="${UITK_PAINT:-cpu}"' "${root}/scripts/system-config-gui"; then
  log_fail "system-config-gui wrapper must default UITK_PAINT=cpu (virtio EGL)"
fi
if ! grep -q 'UITK_BACKEND="${UITK_BACKEND:-wayland}"' "${root}/scripts/system-config-gui"; then
  log_fail "system-config-gui wrapper must prefer UITK_BACKEND=wayland"
fi

if [[ "${failed}" -ne 0 ]]; then
  echo "Wrapper modes / profiledef file_permissions / swaybg pin failed." >&2
  exit 1
fi

echo "Wrapper modes OK (airootfs +x, profiledef 755, swaybg listed)."
