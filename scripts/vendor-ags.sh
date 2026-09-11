#!/usr/bin/env bash
# Build AGS + selected Astal libraries from pinned upstream tarballs and
# install them into a DESTDIR (the archiso airootfs) under /usr/local.
#
# Official Arch repos only. No AUR helper, no Coda pacman repo.
# Must run on an Arch system (the ISO builder). Never prompts for sudo.
#
# Unprivileged builds cannot write /usr/local. Each library is installed
# into a writable staging sysroot ($CACHE/stage) with --prefix=/usr/local
# so g-ir-compiler records live /usr/local soname paths, then copied to
# DESTDIR. Staging .pc files are rewritten so later meson/valac invocations
# see headers, VAPIs, and GIRs without installing to the host.
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
STAGE="${CODA_AGS_STAGE:-${CACHE}/stage}"
JOBS="${CODA_AGS_JOBS:-$(nproc 2>/dev/null || echo 4)}"
HOST_PKG_CONFIG_PATH="${PKG_CONFIG_PATH:-}"
HOST_LD_LIBRARY_PATH="${LD_LIBRARY_PATH:-}"
HOST_LIBRARY_PATH="${LIBRARY_PATH:-}"
HOST_GI_TYPELIB_PATH="${GI_TYPELIB_PATH:-}"
HOST_XDG_DATA_DIRS="${XDG_DATA_DIRS:-/usr/local/share:/usr/share}"
HOST_C_INCLUDE_PATH="${C_INCLUDE_PATH:-}"

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

# Point later compiles at the staged prefix without using PKG_CONFIG_SYSROOT_DIR
# (that would also prefix system glib -I/-L paths and break host deps).
rewrite_stage_pkgconfig() {
  local pcdir="${STAGE}${PREFIX}/lib/pkgconfig"
  local pc
  [[ -d "${pcdir}" ]] || return 0
  for pc in "${pcdir}"/*.pc; do
    [[ -f "${pc}" ]] || continue
    if grep -q "^prefix=${PREFIX}$" "${pc}"; then
      sed -i "s|^prefix=${PREFIX}$|prefix=${STAGE}${PREFIX}|" "${pc}"
    fi
  done
}

install_valac_wrapper() {
  local real_valac
  real_valac="$(command -v valac)"
  if [[ -z "${real_valac}" ]]; then
    echo "vendor-ags: valac not found (install official vala)" >&2
    exit 1
  fi
  install -d "${CACHE}/bin"
  cat >"${CACHE}/bin/valac" <<EOF
#!/usr/bin/env bash
# Meson/valac do not search DESTDIR vapidirs. Extra --vapidir/--girdir
# after the staged Astal installs makes --pkg astal-io-0.1 resolve.
args=()
if [[ -d "${STAGE}${PREFIX}/share/vala/vapi" ]]; then
  args+=(--vapidir="${STAGE}${PREFIX}/share/vala/vapi")
fi
if [[ -d "${STAGE}${PREFIX}/share/gir-1.0" ]]; then
  args+=(--girdir="${STAGE}${PREFIX}/share/gir-1.0")
fi
exec "${real_valac}" "\${args[@]}" "\$@"
EOF
  chmod +x "${CACHE}/bin/valac"
}

expose_stage() {
  rewrite_stage_pkgconfig
  export PKG_CONFIG_PATH="${STAGE}${PREFIX}/lib/pkgconfig${HOST_PKG_CONFIG_PATH:+:${HOST_PKG_CONFIG_PATH}}"
  export LD_LIBRARY_PATH="${STAGE}${PREFIX}/lib${HOST_LD_LIBRARY_PATH:+:${HOST_LD_LIBRARY_PATH}}"
  export LIBRARY_PATH="${STAGE}${PREFIX}/lib${HOST_LIBRARY_PATH:+:${HOST_LIBRARY_PATH}}"
  export GI_TYPELIB_PATH="${STAGE}${PREFIX}/lib/girepository-1.0${HOST_GI_TYPELIB_PATH:+:${HOST_GI_TYPELIB_PATH}}"
  export XDG_DATA_DIRS="${STAGE}${PREFIX}/share:${HOST_XDG_DATA_DIRS}"
  export C_INCLUDE_PATH="${STAGE}${PREFIX}/include${HOST_C_INCLUDE_PATH:+:${HOST_C_INCLUDE_PATH}}"
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
    --pkg-config-path="${STAGE}${PREFIX}/lib/pkgconfig" \
    "$@"
  meson compile -C "${bdir}" -j "${JOBS}"
  # ISO image first: .pc / typelibs keep live prefix=/usr/local.
  DESTDIR="${DESTDIR}" meson install -C "${bdir}"
  # Writable stage for the next library's meson/valac (may be unprivileged).
  DESTDIR="${STAGE}" meson install -C "${bdir}"
  expose_stage
}

install_build_deps

rm -rf "${STAGE}"
install -d "${CACHE}" "${STAGE}${PREFIX}/lib/pkgconfig" "${DESTDIR}${PREFIX}"
install_valac_wrapper
export PATH="${CACHE}/bin:${PATH}"

# Do not use PKG_CONFIG_SYSROOT_DIR: it prefixes host -I/-L as well.
unset PKG_CONFIG_SYSROOT_DIR || true
export GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"
export GOFLAGS="${GOFLAGS:--mod=readonly}"
export npm_config_audit=false
export npm_config_fund=false
expose_stage

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
  npm install --no-fund --no-audit
  if [[ ! -d node_modules/gnim ]]; then
    npm install --no-fund --no-audit --no-save gnim@^1.8.0
  fi
)

log "building AGS CLI + JS package"
meson_install "${CACHE}/ags"

if [[ -d "${DESTDIR}${PREFIX}/share/glib-2.0/schemas" ]]; then
  glib-compile-schemas "${DESTDIR}${PREFIX}/share/glib-2.0/schemas"
fi

install -d "${DESTDIR}/etc/ld.so.conf.d"
printf '%s\n' "${PREFIX}/lib" >"${DESTDIR}/etc/ld.so.conf.d/codalinux-usr-local.conf"

if [[ ! -x "${DESTDIR}${PREFIX}/bin/ags" ]]; then
  echo "vendor-ags: expected ${DESTDIR}${PREFIX}/bin/ags after install" >&2
  exit 1
fi
if [[ ! -e "${DESTDIR}${PREFIX}/lib/pkgconfig/astal-io-0.1.pc" ]]; then
  echo "vendor-ags: expected ${DESTDIR}${PREFIX}/lib/pkgconfig/astal-io-0.1.pc" >&2
  exit 1
fi
if grep -q "^prefix=${STAGE}${PREFIX}$" "${DESTDIR}${PREFIX}/lib/pkgconfig/"*.pc 2>/dev/null; then
  echo "vendor-ags: DESTDIR pkg-config files must keep prefix=${PREFIX}" >&2
  exit 1
fi

log "installed AGS ${AGS_COMMIT} and Astal ${ASTAL_COMMIT} into ${DESTDIR}${PREFIX}"
