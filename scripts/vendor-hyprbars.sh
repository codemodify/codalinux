#!/usr/bin/env bash
# Build hyprbars from pinned hyprland-plugins and install the plugin .so
# into DESTDIR /usr/local/lib/hyprland/. Official Arch repos only.
#
# Must run on Arch (the ISO builder). Compiles against the builder's
# hyprland package so the ABI matches the live compositor.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Official hyprland-plugins hyprpm.toml pin for Hyprland 0.56.2
# (hyprland commit efb5099… → plugin 7644cec…). Keep in sync with
# desktop/hypr/README.md. Do not track hyprland-plugins main: later
# chases expect headers Arch 0.56.2 does not ship
# (hyprland/src/desktop/view/window/Window.hpp).
HYPRBARS_COMMIT="${CODA_HYPRBARS_COMMIT:-7644cecdb947060682891a0db2a0cdc5c0b9e704}"
HYPRBARS_URL="https://github.com/hyprwm/hyprland-plugins/archive/${HYPRBARS_COMMIT}.tar.gz"
# pkg-config --modversion hyprland series this pin matches.
HYPRBARS_HYPRLAND_SERIES="${CODA_HYPRBARS_HYPRLAND_SERIES:-0.56}"

PREFIX="/usr/local"
DESTDIR="${1:-${root}/archiso/airootfs}"
CACHE_ROOT="${CODA_HYPRBARS_CACHE:-${root}/.cache/coda-hyprbars}"
CACHE="${CACHE_ROOT}/${HYPRBARS_COMMIT}"
JOBS="${CODA_HYPRBARS_JOBS:-$(nproc 2>/dev/null || echo 4)}"

log() { printf 'vendor-hyprbars: %s\n' "$*"; }

if ! command -v pacman >/dev/null 2>&1; then
  echo "vendor-hyprbars.sh must run on Arch (ISO builder). pacman not found." >&2
  exit 1
fi

install_build_deps() {
  local -a deps
  mapfile -t deps < <(grep -vE '^\s*(#|$)' "${root}/packages/hyprbars-build-deps.txt")
  if [[ "$(id -u)" -eq 0 ]]; then
    log "installing official-repo build deps (includes hyprland headers)"
    pacman -S --noconfirm --needed "${deps[@]}"
  else
    local missing=0
    local pkg
    for pkg in "${deps[@]}"; do
      if ! pacman -Q "${pkg}" >/dev/null 2>&1; then
        echo "missing build dep: ${pkg}" >&2
        missing=1
      fi
    done
    if [[ "${missing}" -ne 0 ]]; then
      echo "Install official hyprbars build deps as root, then retry:" >&2
      echo "  pacman -S --noconfirm --needed ${deps[*]}" >&2
      exit 1
    fi
  fi
}

fetch_tarball() {
  local dest="${CACHE}/src"
  if [[ -d "${dest}" && -f "${dest}/hyprbars/Makefile" ]]; then
    log "reusing hyprland-plugins sources in ${dest}"
    return 0
  fi
  rm -rf "${dest}"
  install -d "${dest}"
  log "fetching hyprland-plugins ${HYPRBARS_COMMIT}"
  curl -fsSL --retry 5 --retry-all-errors --retry-delay 2 "${HYPRBARS_URL}" \
    | tar -xz -C "${dest}" --strip-components=1
}

build_and_install() {
  local src="${CACHE}/src/hyprbars"
  [[ -f "${src}/Makefile" ]] || {
    echo "vendor-hyprbars: missing ${src}/Makefile" >&2
    exit 1
  }
  local hypr_ver
  hypr_ver="$(pkg-config --modversion hyprland 2>/dev/null || echo unknown)"
  case "${hypr_ver}" in
    "${HYPRBARS_HYPRLAND_SERIES}".*)
      ;;
    *)
      echo "vendor-hyprbars: hyprland ${hypr_ver} is not ${HYPRBARS_HYPRLAND_SERIES}.x" >&2
      echo "Update HYPRBARS_COMMIT from hyprland-plugins hyprpm.toml commit_pins, then retry." >&2
      exit 1
      ;;
  esac
  log "building hyprbars ${HYPRBARS_COMMIT} against ${hypr_ver} hyprland.pc"
  make -C "${src}" -j "${JOBS}" all
  local so=""
  if [[ -f "${src}/hyprbars.so" ]]; then
    so="${src}/hyprbars.so"
  elif [[ -f "${src}/libhyprbars.so" ]]; then
    so="${src}/libhyprbars.so"
  else
    echo "vendor-hyprbars: build produced no hyprbars.so" >&2
    exit 1
  fi
  install -d "${DESTDIR}${PREFIX}/lib/hyprland"
  install -m 0755 "${so}" "${DESTDIR}${PREFIX}/lib/hyprland/libhyprbars.so"
  log "installed ${DESTDIR}${PREFIX}/lib/hyprland/libhyprbars.so"
}

install_build_deps
install -d "${CACHE}"
fetch_tarball
build_and_install
