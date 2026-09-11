#!/usr/bin/env bash
# Build AGS + selected Astal libraries from pinned upstream tarballs and
# install them into a DESTDIR (the archiso airootfs) under /usr/local.
#
# Official Arch repos only. No AUR helper, no Coda pacman repo.
# Must run on an Arch system (the ISO builder). Never prompts for sudo.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Keep in sync with desktop/ags/README.md
ASTAL_COMMIT="${CODA_ASTAL_COMMIT:-ae8dc0acc66932171ec70d347a8cab9310ce74e4}"
AGS_COMMIT="${CODA_AGS_COMMIT:-bbee2f18939f1ec7ff720e717cf305e73635628f}"
ASTAL_URL="https://github.com/Aylur/astal/archive/${ASTAL_COMMIT}.tar.gz"
AGS_URL="https://github.com/Aylur/ags/archive/${AGS_COMMIT}.tar.gz"

PREFIX="/usr/local"
DESTDIR="${1:-${root}/archiso/airootfs}"
CACHE="${CODA_AGS_CACHE:-${root}/.cache/coda-ags}"
JOBS="${CODA_AGS_JOBS:-$(nproc 2>/dev/null || echo 4)}"

log() { printf 'vendor-ags: %s\n' "$*"; }

if ! command -v pacman >/dev/null 2>&1; then
  echo "vendor-ags.sh must run on Arch (ISO builder). pacman not found." >&2
  exit 1
fi

install_build_deps() {
  local -a deps
  mapfile -t deps < <(grep -vE '^\s*(#|$)' "${root}/packages/ags-build-deps.txt")
  if [[ "$(id -u)" -eq 0 ]]; then
    log "installing official-repo build deps"
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
      echo "Install official AGS build deps as root, then retry:" >&2
      echo "  pacman -S --noconfirm --needed ${deps[*]}" >&2
      exit 1
    fi
  fi
}

fetch_tarball() {
  local url="$1"
  local dest="$2"
  local name="$3"
  if [[ -d "${dest}" && -n "$(ls -A "${dest}" 2>/dev/null)" ]]; then
    log "reusing ${name} sources in ${dest}"
    return 0
  fi
  install -d "${dest}"
  log "fetching ${name} from ${url}"
  curl -fsSL --retry 5 --retry-all-errors --retry-delay 2 "${url}" \
    | tar -xz -C "${dest}" --strip-components=1
}

meson_install() {
  local src="$1"
  shift
  local bdir="${src}/build-coda"
  rm -rf "${bdir}"
  meson setup "${bdir}" "${src}" \
    --prefix="${PREFIX}" \
    --libdir=lib \
    --buildtype=release \
    --wrap-mode=nodownload \
    "$@"
  meson compile -C "${bdir}" -j "${JOBS}"
  if [[ -w "${PREFIX}" ]]; then
    meson install -C "${bdir}"
  fi
  DESTDIR="${DESTDIR}" meson install -C "${bdir}"
}

install_build_deps

export PKG_CONFIG_PATH="${PREFIX}/lib/pkgconfig:${DESTDIR}${PREFIX}/lib/pkgconfig${PKG_CONFIG_PATH:+:${PKG_CONFIG_PATH}}"
export LD_LIBRARY_PATH="${PREFIX}/lib:${DESTDIR}${PREFIX}/lib${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}"
export GI_TYPELIB_PATH="${PREFIX}/lib/girepository-1.0:${DESTDIR}${PREFIX}/lib/girepository-1.0${GI_TYPELIB_PATH:+:${GI_TYPELIB_PATH}}"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"
export GOFLAGS="${GOFLAGS:--mod=readonly}"
export npm_config_audit=false
export npm_config_fund=false

install -d "${CACHE}" "${DESTDIR}${PREFIX}"

fetch_tarball "${ASTAL_URL}" "${CACHE}/astal" "Astal ${ASTAL_COMMIT}"
fetch_tarball "${AGS_URL}" "${CACHE}/ags" "AGS ${AGS_COMMIT}"

log "building Astal io"
meson_install "${CACHE}/astal/lib/astal/io"
log "building Astal gtk4"
meson_install "${CACHE}/astal/lib/astal/gtk4"
log "building Astal apps"
meson_install "${CACHE}/astal/lib/apps" -Dcli=false
log "building Astal hyprland"
meson_install "${CACHE}/astal/lib/hyprland" -Dcli=false
log "building Astal notifd (lib only)"
meson_install "${CACHE}/astal/lib/notifd" -Dcli=false
log "building Astal bluetooth"
meson_install "${CACHE}/astal/lib/bluetooth"
log "building Astal wireplumber"
meson_install "${CACHE}/astal/lib/wireplumber"
log "building Astal battery"
meson_install "${CACHE}/astal/lib/battery" -Dcli=false

log "npm install AGS (gnim peer + lockfile)"
(
  cd "${CACHE}/ags"
  if [[ -f package-lock.json ]]; then
    npm install --no-fund --no-audit
  else
    npm install --no-fund --no-audit
  fi
  if [[ ! -d node_modules/gnim ]]; then
    npm install --no-fund --no-audit --no-save gnim@^1.8.0
  fi
)

log "building AGS CLI + JS package"
meson_install "${CACHE}/ags"

if [[ -d "${DESTDIR}${PREFIX}/share/glib-2.0/schemas" ]]; then
  glib-compile-schemas "${DESTDIR}${PREFIX}/share/glib-2.0/schemas"
fi
if [[ -d "${PREFIX}/share/glib-2.0/schemas" && -w "${PREFIX}/share/glib-2.0/schemas" ]]; then
  glib-compile-schemas "${PREFIX}/share/glib-2.0/schemas" || true
fi

install -d "${DESTDIR}/etc/ld.so.conf.d"
printf '%s\n' "${PREFIX}/lib" >"${DESTDIR}/etc/ld.so.conf.d/codalinux-usr-local.conf"

if [[ ! -x "${DESTDIR}${PREFIX}/bin/ags" ]]; then
  echo "vendor-ags: expected ${DESTDIR}${PREFIX}/bin/ags after install" >&2
  exit 1
fi

log "installed AGS ${AGS_COMMIT} and Astal ${ASTAL_COMMIT} into ${DESTDIR}${PREFIX}"
