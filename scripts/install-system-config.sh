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

export CGO_ENABLED=0
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"
build() {
  (cd "${mod}" && go build -o "${bindir}/" ./cmd/system-configd ./cmd/system-config-apply \
    ./cmd/system-config-report ./cmd/system-config ./cmd/system-config-tui \
    ./cmd/system-config-gui)
}
if ! build; then
  echo "install-system-config: retry with GOTOOLCHAIN=auto" >&2
  GOTOOLCHAIN=auto build
fi

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

echo "install-system-config: installed to ${bindir}"
