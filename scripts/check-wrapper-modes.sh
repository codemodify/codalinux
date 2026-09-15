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
for sh in coda-install coda-install-ab.sh coda-install-lib.sh coda-slot \
          coda-install-verify.sh qemu-install-e2e.sh coda-desktop-mount; do
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
