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

echo "install-system-config: installed to ${bindir}"
