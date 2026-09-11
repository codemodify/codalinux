#!/usr/bin/env bash
# Re-apply CodaLinux identity after the filesystem package writes Arch's files.
# archiso copies airootfs *before* pacman, so these packaged paths cannot live
# in the overlay or pacstrap fails with "exists in filesystem".
set -euo pipefail

share="/usr/local/share/codalinux"

if [[ -f "${share}/os-release" ]]; then
  install -m 0644 "${share}/os-release" /usr/lib/os-release
fi
if [[ -f "${share}/issue" ]]; then
  install -m 0644 "${share}/issue" /etc/issue
fi
if [[ -f "${share}/issue.net" ]]; then
  install -m 0644 "${share}/issue.net" /etc/issue.net
fi
