#!/usr/bin/env bash
# Re-apply CodaLinux /usr/lib/os-release after the filesystem package writes Arch's.
# archiso copies airootfs *before* pacman, so this hook is required.
set -euo pipefail

src="/usr/local/share/codalinux/os-release"
if [[ ! -f "${src}" ]]; then
  echo "codalinux: missing ${src}" >&2
  exit 0
fi

install -m 0644 "${src}" /usr/lib/os-release
