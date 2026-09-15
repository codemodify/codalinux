#!/usr/bin/env bash
# Build and install system-config binaries (+ launch helper) into DEST.
# Usage: install-system-config.sh <dest-root>
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dest="${1:?dest root}"
mod="${root}/core/system-config"

if ! command -v go >/dev/null 2>&1; then
  echo "install-system-config: go not found; install Go 1.22+ (Arch: pacman -S --noconfirm --needed go git)" >&2
  exit 1
fi

bindir="${dest}/usr/local/bin"
libdir="${dest}/usr/local/lib/codalinux"
install -d "${bindir}" "${libdir}"

# Helpers stay CGO-off (static). The GUI must NOT: uitoolkit Wayland/X11
# backends are Linux+CGO; CGO_ENABLED=0 makes autoBackend offscreen so
# system-config-gui stays alive with no window. Never ship that silently.
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"

need_gui_pc() {
  command -v pkg-config >/dev/null 2>&1 || return 1
  pkg-config --exists wayland-client wayland-cursor wayland-egl xkbcommon egl glesv2
}

ensure_gui_cgo_deps() {
  if need_gui_pc; then
    return 0
  fi
  local depsfile="${root}/packages/system-config-gui-build-deps.txt"
  if command -v pacman >/dev/null 2>&1 && [[ "$(id -u)" -eq 0 && -f "${depsfile}" ]]; then
    local -a deps
    mapfile -t deps < <(grep -vE '^\s*(#|$)' "${depsfile}")
    echo "install-system-config: installing CGO GUI build deps" >&2
    pacman -S --noconfirm --needed "${deps[@]}"
  fi
  if ! need_gui_pc; then
    echo "install-system-config: refusing headless-only system-config-gui" >&2
    echo "uitoolkit Wayland/X11 need Linux+CGO (wayland, libxkbcommon, libX11/Xext/Xrandr, EGL/GLES)." >&2
    echo "On Arch: pacman -S --noconfirm --needed \$(grep -vE '^\\s*(#|$)' packages/system-config-gui-build-deps.txt)" >&2
    exit 1
  fi
}

build_helpers() {
  (cd "${mod}" && CGO_ENABLED=0 go build -o "${bindir}/" \
    ./cmd/system-configd ./cmd/system-config-apply \
    ./cmd/system-config-report ./cmd/system-config ./cmd/system-config-tui)
}

build_gui() {
  (cd "${mod}" && CGO_ENABLED=1 go build -o "${libdir}/system-config-gui" \
    ./cmd/system-config-gui)
}

if ! build_helpers; then
  echo "install-system-config: retry helpers with GOTOOLCHAIN=auto" >&2
  GOTOOLCHAIN=auto build_helpers
fi
ensure_gui_cgo_deps
if ! build_gui; then
  echo "install-system-config: retry GUI with GOTOOLCHAIN=auto" >&2
  GOTOOLCHAIN=auto build_gui
fi

gui="${libdir}/system-config-gui"
if [[ ! -x "${gui}" ]]; then
  echo "install-system-config: ${gui} missing after CGO build" >&2
  exit 1
fi
if command -v ldd >/dev/null 2>&1; then
  if ldd "${gui}" 2>/dev/null | grep -q 'not a dynamic'; then
    echo "install-system-config: ${gui} is statically linked (CGO off?) — refusing headless GUI" >&2
    exit 1
  fi
  if ! ldd "${gui}" | grep -qE 'libwayland-client|libwayland'; then
    echo "install-system-config: ${gui} is not linked to wayland — refusing headless GUI" >&2
    ldd "${gui}" >&2 || true
    exit 1
  fi
fi
chmod 0755 "${gui}"

install -m 0755 "${root}/scripts/system-config-gui" "${bindir}/system-config-gui"
install -m 0755 "${root}/scripts/system-config-apply-launch" \
  "${libdir}/system-config-apply-launch"
chmod 0755 "${bindir}/system-config"*
docdir="${dest}/usr/local/share/codalinux/system-config"
install -d "${docdir}"
install -m 0644 "${mod}/README.md" "${docdir}/README.md"
install -m 0755 "${mod}/scripts/guest-smoke.sh" "${docdir}/guest-smoke.sh"
install -m 0755 "${mod}/scripts/guest-e2e-all.sh" "${docdir}/guest-e2e-all.sh"

# build-iso.sh DEST is archiso/airootfs — units already live there.
# GNU install errors on same-inode src/dest. Only copy when dest differs.
install_if_different() {
  local mode="$1" src="$2" destf="$3"
  [[ -e "${src}" ]] || return 0
  mkdir -p "$(dirname "${destf}")"
  if [[ -e "${destf}" ]]; then
    if [[ "$(stat -c '%d:%i' "${src}" 2>/dev/null || true)" == "$(stat -c '%d:%i' "${destf}" 2>/dev/null || true)" ]]; then
      return 0
    fi
  fi
  install -m "${mode}" "${src}" "${destf}"
}

unit_user="${root}/archiso/airootfs/etc/systemd/user"
unit_sys="${root}/archiso/airootfs/etc/systemd/system"
install -d "${dest}/etc/systemd/user/graphical-session.target.wants" \
  "${dest}/etc/systemd/system/graphical.target.wants"
if [[ -f "${unit_user}/system-configd.service" ]]; then
  install_if_different 0644 "${unit_user}/system-configd.service" \
    "${dest}/etc/systemd/user/system-configd.service"
  ln -sfn /etc/systemd/user/system-configd.service \
    "${dest}/etc/systemd/user/graphical-session.target.wants/system-configd.service"
fi
if [[ -f "${unit_user}/system-config-report.service" ]]; then
  install_if_different 0644 "${unit_user}/system-config-report.service" \
    "${dest}/etc/systemd/user/system-config-report.service"
  ln -sfn /etc/systemd/user/system-config-report.service \
    "${dest}/etc/systemd/user/graphical-session.target.wants/system-config-report.service"
fi
if [[ -f "${unit_sys}/system-config-apply.service" ]]; then
  install_if_different 0644 "${unit_sys}/system-config-apply.service" \
    "${dest}/etc/systemd/system/system-config-apply.service"
  ln -sfn /etc/systemd/system/system-config-apply.service \
    "${dest}/etc/systemd/system/graphical.target.wants/system-config-apply.service"
fi

if [[ -f "${root}/desktop/applications/system-config-gui.desktop" ]]; then
  install -d "${dest}/usr/share/applications"
  install -m 0644 "${root}/desktop/applications/system-config-gui.desktop" \
    "${dest}/usr/share/applications/system-config-gui.desktop"
fi
if [[ -f "${root}/desktop/applications/coda-settings.desktop" ]]; then
  install -d "${dest}/usr/share/applications"
  install -m 0644 "${root}/desktop/applications/coda-settings.desktop" \
    "${dest}/usr/share/applications/coda-settings.desktop"
fi
rm -f "${dest}/etc/systemd/user/default.target.wants/system-configd.service" \
  "${dest}/etc/systemd/user/default.target.wants/system-config-report.service"

echo "install-system-config: installed to ${bindir} (GUI CGO/wayland)"
