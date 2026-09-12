#!/usr/bin/env bash
# Build the CodaLinux live ISO from archiso/.
#
# Unattended: never prompts for sudo or pacman providers.
# Paths (in order):
#   1. Native/rootless mkarchiso when it is in PATH
#   2. Docker/Podman as the current user (must already work without sudo)
#   3. Exit with a docker-group / setup message
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

fail_no_sudo() {
  cat >&2 <<'EOF'
This script never calls sudo. Unattended abox builds must not prompt.

If you need elevated tools:
  - Prefer native/rootless mkarchiso (archiso 89+ can unshare as a regular user).
  - For Docker: add your user to the docker group, then re-login:
        # one-time, interactive — not this script
        usermod -aG docker "$USER"
    docker info must work without sudo. Do not grant passwordless root.
  - Optional sudoers (host admin, not this repo) may allow only
    /usr/bin/mkarchiso and /usr/bin/docker — never ALL=(ALL) NOPASSWD: ALL.
EOF
  exit 1
}

engine_usable() {
  local bin="$1"
  command -v "${bin}" >/dev/null 2>&1 || return 1
  if [[ "${bin}" == docker ]]; then
    docker info >/dev/null 2>&1
  else
    "${bin}" info >/dev/null 2>&1
  fi
}

prepare_overlay() {
  "${root}/scripts/compose-package-lists.sh"
  "${root}/scripts/check-package-lists.sh"

  local overlay="${profile}/airootfs"

  install -d "${overlay}/etc/pacman.d/hooks"
  install -m 0644 "${root}/branding/hooks/codalinux-os-release.hook" \
    "${overlay}/etc/pacman.d/hooks/codalinux-os-release.hook"
  install -m 0644 "${root}/branding/hooks/codalinux-locale.hook" \
    "${overlay}/etc/pacman.d/hooks/codalinux-locale.hook"
  install -d "${overlay}/usr/local/lib/codalinux"
  install -m 0755 "${root}/branding/hooks/apply-os-release.sh" \
    "${overlay}/usr/local/lib/codalinux/apply-os-release.sh"
  install -m 0755 "${root}/branding/hooks/apply-locale.sh" \
    "${overlay}/usr/local/lib/codalinux/apply-locale.sh"
  install -m 0755 "${root}/scripts/coda-live-setup.sh" \
    "${overlay}/usr/local/lib/codalinux/coda-live-setup.sh"
  install -m 0755 "${root}/scripts/coda-install-config.py" \
    "${overlay}/usr/local/lib/codalinux/coda-install-config.py"
  install -d "${overlay}/usr/local/share/codalinux"
  install -m 0644 "${root}/branding/os-release" \
    "${overlay}/usr/local/share/codalinux/os-release"
  install -m 0644 "${root}/branding/issue" \
    "${overlay}/usr/local/share/codalinux/issue"
  install -m 0644 "${root}/branding/issue.net" \
    "${overlay}/usr/local/share/codalinux/issue.net"
  # Do not write /usr/lib/os-release or /etc/issue* here — filesystem owns them.

  install -d "${overlay}/usr/share/wayland-sessions"
  install -m 0644 "${root}/sessions/wayland/codalinux-hyprland.desktop" \
    "${overlay}/usr/share/wayland-sessions/codalinux-hyprland.desktop"

  install_hypr_configs() {
    local dest="$1"
    install -d "${dest}"
    # Drop leftover hyprlang compositor config (removed in Hyprland 0.57).
    rm -f "${dest}/hyprland.conf"
    local src
    for src in "${root}/desktop/hypr/"*; do
      [[ -f "${src}" ]] || continue
      case "${src}" in
        *.md) continue ;;
      esac
      install -m 0644 "${src}" "${dest}/"
    done
  }
  install_hypr_configs "${overlay}/etc/skel/.config/hypr"
  install_hypr_configs "${overlay}/etc/xdg/hypr"
  # Live root session uses the same Hyprland configs.
  install_hypr_configs "${overlay}/root/.config/hypr"

  # Drop leftover interim-shell configs (Waybar was rejected).
  rm -rf \
    "${overlay}/etc/xdg/waybar" "${overlay}/etc/skel/.config/waybar" \
    "${overlay}/etc/xdg/fuzzel" "${overlay}/etc/skel/.config/fuzzel" \
    "${overlay}/etc/xdg/mako" "${overlay}/etc/skel/.config/mako"

  install -d "${overlay}/usr/share/backgrounds/codalinux"
  # Never overwrite a committed wallpaper. Generator is last-resort only.
  if [[ ! -f "${root}/branding/wallpapers/default.png" ]]; then
    python3 "${root}/scripts/gen-wallpaper.py"
  fi
  install -m 0644 "${root}/branding/wallpapers/default.png" \
    "${overlay}/usr/share/backgrounds/codalinux/default.png"
  install -d "${overlay}/usr/share/applications"
  install -m 0644 "${root}/desktop/applications/"*.desktop \
    "${overlay}/usr/share/applications/"
  install -d "${overlay}/usr/local/share/codalinux"
  install -m 0644 "${root}/desktop/share/input-help.txt" \
    "${overlay}/usr/local/share/codalinux/input-help.txt"
  install -d "${overlay}/usr/local/share/codalinux/ags"
  rm -rf "${overlay}/usr/local/share/codalinux/ags"
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
  install -m 0755 "${root}/scripts/coda-hyprland" \
    "${overlay}/usr/local/bin/coda-hyprland"
  install -m 0755 "${root}/scripts/coda-settings" \
    "${overlay}/usr/local/bin/coda-settings"
  install -m 0755 "${root}/scripts/coda-sandbox" \
    "${overlay}/usr/local/bin/coda-sandbox"
  install -m 0755 "${root}/scripts/coda-ags" \
    "${overlay}/usr/local/bin/coda-ags"
  install -m 0755 "${root}/scripts/coda-hypr-ws" \
    "${overlay}/usr/local/bin/coda-hypr-ws"
  install -m 0755 "${root}/scripts/coda-hyprlock" \
    "${overlay}/usr/local/bin/coda-hyprlock"
  install -m 0755 "${root}/scripts/coda-hyprpaper" \
    "${overlay}/usr/local/bin/coda-hyprpaper"
  install -m 0755 "${root}/scripts/coda-wallpaper" \
    "${overlay}/usr/local/bin/coda-wallpaper"
  install -m 0755 "${root}/scripts/coda-sync-desktop-from-host.sh" \
    "${overlay}/usr/local/bin/coda-sync-desktop-from-host"

  # mkarchiso file_permissions is the squashfs source of truth; keep
  # airootfs +x anyway and fail the build if a wrapper is not executable.
  chmod 0755 "${overlay}/usr/local/bin/coda-"* || true
  local wrap
  for wrap in coda-hyprpaper coda-wallpaper coda-ags coda-hyprland \
              coda-hyprlock coda-hypr-ws coda-install coda-settings \
              coda-sandbox coda-sync-desktop-from-host; do
    if [[ ! -x "${overlay}/usr/local/bin/${wrap}" ]]; then
      echo "build-iso: ${overlay}/usr/local/bin/${wrap} is not executable" >&2
      exit 1
    fi
  done
  "${root}/scripts/check-wrapper-modes.sh"

  mkdir -p "${work}" "${out}"
}

# A leftover db.lck from a killed mkarchiso run makes pacstrap fail.
clean_build_dirs() {
  if [[ -d "${work}" ]]; then
    if ! rm -rf "${work}"; then
      echo "Cannot remove ${work} (often a root-owned leftover from an old sudo build)." >&2
      echo "chown -R \"\$USER:\$USER\" ${work} and retry. This script will not sudo." >&2
      exit 1
    fi
  fi
  mkdir -p "${work}" "${out}"
}

run_mkarchiso() {
  prepare_overlay
  log "Vendoring AGS/Astal into airootfs /usr/local (official-repo build deps only)"
  "${root}/scripts/vendor-ags.sh" "${profile}/airootfs"
  log "Vendoring hyprbars into airootfs /usr/local/lib/hyprland (official-repo build deps only)"
  "${root}/scripts/vendor-hyprbars.sh" "${profile}/airootfs"
  clean_build_dirs
  log "Running mkarchiso -v -w ${work} -o ${out} ${profile}"
  # mkarchiso drives pacstrap; lists pin providers so pacman stays noninteractive.
  mkarchiso -v -w "${work}" -o "${out}" "${profile}"
  log "ISO output:"
  ls -lh "${out}"/*.iso 2>/dev/null || ls -lh "${out}"
}

run_in_arch_container() {
  local engine="$1"
  local -a cmd
  cmd=("${engine}" run --rm --privileged --name coda-iso-build -e CODA_ISO_INNER=1)
  if [[ -n "${SOURCE_DATE_EPOCH:-}" ]]; then
    cmd+=(-e "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}")
  fi
  cmd+=(
    -v "${root}:${root}"
    -w "${root}"
    docker.io/library/archlinux:latest
    bash -lc "set -euo pipefail
      pacman-key --init
      pacman-key --populate archlinux
      pacman -Sy --noconfirm archlinux-keyring
      pacman -Syu --noconfirm archiso python
      # vendor-ags.sh installs the rest of packages/ags-build-deps.txt
      exec ./scripts/build-iso.sh $(printf '%q' "${work}") $(printf '%q' "${out}")
    "
  )
  log "Building inside ${engine} archlinux:latest (privileged, no sudo)"
  "${cmd[@]}"
}

if [[ -n "${CODA_ISO_INNER:-}" ]] || [[ "${CODA_ISO_ENGINE:-}" == native ]]; then
  if ! command -v mkarchiso >/dev/null 2>&1; then
    echo "CODA_ISO_INNER/native set but mkarchiso is missing. pacman -S --noconfirm --needed archiso" >&2
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
if [[ -n "${engine}" ]]; then
  if ! engine_usable "${engine}"; then
    echo "${engine} is not usable without sudo." >&2
    fail_no_sudo
  fi
  run_in_arch_container "${engine}"
  exit 0
fi

if engine_usable docker; then
  run_in_arch_container docker
  exit 0
fi
if engine_usable podman; then
  run_in_arch_container podman
  exit 0
fi

if command -v docker >/dev/null 2>&1; then
  echo "docker is installed but 'docker info' failed without sudo." >&2
  fail_no_sudo
fi

cat >&2 <<'EOF'
mkarchiso not found, and docker/podman are not usable without sudo.

CodaLinux ISOs are built with official archiso, unattended:

  # On Arch (preferred; archiso 89+ can run mkarchiso without root via unshare):
  pacman -S --noconfirm --needed archiso
  ./scripts/build-iso.sh

  # On other hosts: docker must work as your user (docker group), then:
  docker pull archlinux:latest
  ./scripts/build-iso.sh

See README.md (Unattended local builds) and DESIGN.md.
EOF
exit 1
