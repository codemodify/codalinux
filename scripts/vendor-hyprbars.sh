#!/usr/bin/env bash
# Build hyprbars from pinned hyprland-plugins and install the plugin .so
# into DESTDIR /usr/local/lib/hyprland/. Official Arch repos only.
#
# Must run on Arch (the ISO builder). Compiles against the builder's
# hyprland package so the ABI matches the live compositor. Arch hyprland
# already ships the plugin header tree under /usr/include/hyprland/src;
# there is no separate headers package and we do not vendor Hyprland source.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Official hyprland-plugins hyprpm.toml pin for Hyprland 0.56.2
# (hyprland commit efb5099… → plugin 7644cec…). Keep in sync with
# desktop/hypr/README.md. Do not track hyprland-plugins main: later
# chases include hyprland/src/desktop/view/window/Window.hpp, which
# Arch 0.56.2 installs as hyprland/src/desktop/view/Window.hpp.
HYPRBARS_COMMIT="${CODA_HYPRBARS_COMMIT:-7644cecdb947060682891a0db2a0cdc5c0b9e704}"
HYPRBARS_URL="https://github.com/hyprwm/hyprland-plugins/archive/${HYPRBARS_COMMIT}.tar.gz"
# pkg-config --modversion hyprland series this pin matches.
HYPRBARS_HYPRLAND_SERIES="${CODA_HYPRBARS_HYPRLAND_SERIES:-0.56}"

PREFIX="/usr/local"
DESTDIR="${1:-${root}/archiso/airootfs}"
CACHE_ROOT="${CODA_HYPRBARS_CACHE:-${root}/.cache/coda-hyprbars}"
CACHE="${CACHE_ROOT}/${HYPRBARS_COMMIT}"
JOBS="${CODA_HYPRBARS_JOBS:-$(nproc 2>/dev/null || echo 4)}"

# Arch 0.56.2 layout (confirmed from extra/hyprland file list).
HYPRLAND_WINDOW_HPP="/usr/include/hyprland/src/desktop/view/Window.hpp"

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
  local stamp="${dest}/.coda-hyprbars-commit"
  # Old cache layout reused whatever tarball was unpacked first (722f15a).
  if [[ -d "${CACHE_ROOT}/src" ]]; then
    log "removing stale unversioned cache ${CACHE_ROOT}/src"
    rm -rf "${CACHE_ROOT}/src"
  fi
  if [[ -d "${dest}" && -f "${dest}/hyprbars/Makefile" && -f "${stamp}" && "$(<"${stamp}")" == "${HYPRBARS_COMMIT}" ]]; then
    log "reusing hyprland-plugins ${HYPRBARS_COMMIT} in ${dest}"
    return 0
  fi
  rm -rf "${dest}"
  install -d "${dest}"
  log "fetching hyprland-plugins ${HYPRBARS_COMMIT}"
  curl -fsSL --retry 5 --retry-all-errors --retry-delay 2 "${HYPRBARS_URL}" \
    | tar -xz -C "${dest}" --strip-components=1
  printf '%s\n' "${HYPRBARS_COMMIT}" > "${stamp}"
}

assert_header_layout() {
  local src="${CACHE}/src/hyprbars"
  if [[ ! -f "${HYPRLAND_WINDOW_HPP}" ]]; then
    echo "vendor-hyprbars: official hyprland headers missing ${HYPRLAND_WINDOW_HPP}" >&2
    echo "Install extra/hyprland; there is no separate headers package." >&2
    exit 1
  fi
  if grep -Rqs 'hyprland/src/desktop/view/window/Window.hpp' "${src}"; then
    echo "vendor-hyprbars: plugin sources expect view/window/Window.hpp" >&2
    echo "Arch hyprland $(pkg-config --modversion hyprland 2>/dev/null || echo unknown) ships ${HYPRLAND_WINDOW_HPP}" >&2
    echo "Pin hyprland-plugins to the hyprpm.toml commit for this series (0.56.2 → ${HYPRBARS_COMMIT})." >&2
    exit 1
  fi
}

lua_cflags() {
  local flags=""
  if flags="$(pkg-config --cflags lua54 2>/dev/null)" && [[ -n "${flags}" ]]; then
    printf '%s' "${flags}"
    return 0
  fi
  if flags="$(pkg-config --cflags lua5.4 2>/dev/null)" && [[ -n "${flags}" ]]; then
    printf '%s' "${flags}"
    return 0
  fi
  echo "vendor-hyprbars: lua54.pc missing (hyprland.pc does not add <lua.h>)" >&2
  echo "Install official lua54, then retry." >&2
  exit 1
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
  assert_header_layout
  local extra
  extra="$(lua_cflags)"
  # Command-line CXXFLAGS replaces the plugin Makefile's flags (a68fddf
  # dropped -std=c++2b and Arch GCC 16 then rejected std::expected).
  # Always pass a complete C++23 set; -std=c++23 is required for
  # hyprland 0.56.2 headers on GCC 16.
  local cxxflags includes
  cxxflags="-O2 -shared -fPIC -std=c++23 -Wno-c++11-narrowing ${extra}"
  includes="$(pkg-config --cflags pixman-1 libdrm hyprland libinput libudev wayland-server xkbcommon) ${extra}"
  log "building hyprbars ${HYPRBARS_COMMIT} against ${hypr_ver} hyprland.pc (${HYPRLAND_WINDOW_HPP}) with -std=c++23"
  make -C "${src}" -j "${JOBS}" all CXXFLAGS="${cxxflags}" INCLUDES="${includes}"
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
