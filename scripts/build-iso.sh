#!/usr/bin/env bash
# Build the CodaLinux live ISO from the archiso/ profile.
#
# This is a documented placeholder, not a production builder:
#   - requires an Arch Linux host with archiso (mkarchiso)
#   - copies branding/desktop/session overlays into airootfs
#   - does not implement NVIDIA detection or AGS compilation
#
# Usage (from the repo root, as root on Arch):
#   ./scripts/build-iso.sh
#   ./scripts/build-iso.sh /path/to/work /path/to/out
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
profile="${root}/archiso"
work="${1:-${root}/work}"
out="${2:-${root}/out}"

if ! command -v mkarchiso >/dev/null 2>&1; then
  cat >&2 <<'EOF'
mkarchiso not found.

CodaLinux ISOs are built with official archiso on an Arch Linux host:

  pacman -S --needed archiso
  sudo ./scripts/build-iso.sh

CI is not expected to produce a bootable image until an Arch builder exists.
See docs/TODO.md (Live ISO) and DESIGN.md.
EOF
  exit 1
fi

"${root}/scripts/compose-package-lists.sh"
"${root}/scripts/check-package-lists.sh"

overlay="${profile}/airootfs"

# Branding + session files are sourced from their trees so airootfs stays thin.
install -d "${overlay}/usr/lib"
install -m 0644 "${root}/branding/os-release" "${overlay}/usr/lib/os-release"
install -d "${overlay}/etc/pacman.d/hooks"
install -m 0644 "${root}/branding/hooks/codalinux-os-release.hook" \
  "${overlay}/etc/pacman.d/hooks/codalinux-os-release.hook"
install -d "${overlay}/usr/local/lib/codalinux"
install -m 0755 "${root}/branding/hooks/apply-os-release.sh" \
  "${overlay}/usr/local/lib/codalinux/apply-os-release.sh"
install -d "${overlay}/usr/local/share/codalinux"
install -m 0644 "${root}/branding/os-release" \
  "${overlay}/usr/local/share/codalinux/os-release"
install -m 0644 "${root}/branding/issue" "${overlay}/etc/issue"
install -m 0644 "${root}/branding/issue.net" "${overlay}/etc/issue.net"

install -d "${overlay}/usr/share/wayland-sessions"
install -m 0644 "${root}/sessions/wayland/codalinux-hyprland.desktop" \
  "${overlay}/usr/share/wayland-sessions/codalinux-hyprland.desktop"

install -d "${overlay}/etc/skel/.config/hypr"
install -m 0644 "${root}/desktop/hypr/"*.conf "${overlay}/etc/skel/.config/hypr/"
install -d "${overlay}/etc/xdg/hypr"
install -m 0644 "${root}/desktop/hypr/"*.conf "${overlay}/etc/xdg/hypr/"

install -d "${overlay}/usr/share/backgrounds/codalinux"
# Wallpaper assets land here later; directory documents the path.

install -d "${overlay}/usr/local/share/codalinux/ags"
cp -a "${root}/desktop/ags/." "${overlay}/usr/local/share/codalinux/ags/"

mkdir -p "${work}" "${out}"

echo "Running mkarchiso -v -w ${work} -o ${out} ${profile}"
mkarchiso -v -w "${work}" -o "${out}" "${profile}"
