#!/usr/bin/env bash
# Build the CodaLinux live ISO from archiso/.
#
# Paths (in order):
#   1. Native Arch host with mkarchiso in PATH
#   2. Privileged Docker/Podman using docker.io/library/archlinux
#   3. Print setup instructions and exit 1
#
# Usage (repo root):
#   ./scripts/build-iso.sh
#   ./scripts/build-iso.sh /path/to/work /path/to/out
#
# Environment:
#   CODA_ISO_INNER=1     already inside the Arch builder (do not re-enter Docker)
#   CODA_ISO_ENGINE=docker|podman|native
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
profile="${root}/archiso"
work="${1:-${root}/work}"
out="${2:-${root}/out}"

log() { printf '%s\n' "$*"; }

prepare_overlay() {
  "${root}/scripts/compose-package-lists.sh"
  "${root}/scripts/check-package-lists.sh"

  local overlay="${profile}/airootfs"

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
  # Live root session uses the same Hyprland configs.
  install -d "${overlay}/root/.config/hypr"
  install -m 0644 "${root}/desktop/hypr/"*.conf "${overlay}/root/.config/hypr/"

  install -d "${overlay}/usr/share/backgrounds/codalinux"
  install -d "${overlay}/usr/local/share/codalinux/ags"
  cp -a "${root}/desktop/ags/." "${overlay}/usr/local/share/codalinux/ags/"

  install -d "${overlay}/usr/share/codalinux/install"
  install -m 0644 "${root}/install/user_configuration.json" \
    "${overlay}/usr/share/codalinux/install/user_configuration.json"
  install -m 0644 "${root}/install/packages.txt" \
    "${overlay}/usr/share/codalinux/install/packages.txt"
  install -m 0644 "${root}/install/profiles/codalinux.py" \
    "${overlay}/usr/share/codalinux/install/codalinux.py"
  install -d "${overlay}/usr/local/bin"
  install -m 0755 "${root}/scripts/coda-install" \
    "${overlay}/usr/local/bin/coda-install"

  mkdir -p "${work}" "${out}"
}

run_mkarchiso() {
  prepare_overlay
  log "Running mkarchiso -v -w ${work} -o ${out} ${profile}"
  mkarchiso -v -w "${work}" -o "${out}" "${profile}"
  log "ISO output:"
  ls -lh "${out}"/*.iso 2>/dev/null || ls -lh "${out}"
}

run_in_arch_container() {
  local engine="$1"
  local -a cmd
  cmd=(
    "${engine}" run --rm --privileged
    --name coda-iso-build
    -e CODA_ISO_INNER=1
    -e SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-}"
    -v "${root}:${root}"
    -w "${root}"
    docker.io/library/archlinux:latest
    bash -lc "set -euo pipefail
      pacman-key --init
      pacman-key --populate archlinux
      pacman -Sy --noconfirm archlinux-keyring
      pacman -Syu --noconfirm archiso
      exec ./scripts/build-iso.sh $(printf '%q' "${work}") $(printf '%q' "${out}")
    "
  )
  if [[ "$(id -u)" -ne 0 ]] && [[ "${engine}" == docker ]]; then
    cmd=(sudo "${cmd[@]}")
  fi
  log "Building inside ${engine} archlinux:latest (privileged)"
  "${cmd[@]}"
}

if [[ -n "${CODA_ISO_INNER:-}" ]] || [[ "${CODA_ISO_ENGINE:-}" == native ]]; then
  if ! command -v mkarchiso >/dev/null 2>&1; then
    echo "CODA_ISO_INNER/native set but mkarchiso is missing. pacman -S archiso" >&2
    exit 1
  fi
  run_mkarchiso
  exit 0
fi

if command -v mkarchiso >/dev/null 2>&1; then
  run_mkarchiso
  exit 0
fi

engine="${CODA_ISO_ENGINE:-}"
if [[ -z "${engine}" ]]; then
  if command -v docker >/dev/null 2>&1; then
    engine=docker
  elif command -v podman >/dev/null 2>&1; then
    engine=podman
  fi
fi

if [[ -n "${engine}" ]]; then
  run_in_arch_container "${engine}"
  exit 0
fi

cat >&2 <<'EOF'
mkarchiso not found, and neither docker nor podman is available.

CodaLinux ISOs are built with official archiso:

  # On Arch:
  pacman -S --needed archiso
  sudo ./scripts/build-iso.sh

  # On other hosts with Docker:
  sudo docker pull archlinux:latest
  sudo ./scripts/build-iso.sh

See docs/TODO.md and DESIGN.md.
EOF
exit 1
