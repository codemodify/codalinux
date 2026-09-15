#!/usr/bin/env bash
# Enable en_US.UTF-8 and point localtime at America/Denver (Bozeman, MT).
# glibc/tzdata own the packaged files; run after those packages install.
set -euo pipefail

if [[ -f /etc/locale.gen ]]; then
  sed -i 's/^#en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen
  locale-gen >/dev/null 2>&1 || true
fi
if [[ -e /usr/share/zoneinfo/America/Denver ]]; then
  ln -sfn /usr/share/zoneinfo/America/Denver /etc/localtime
fi
printf 'LANG=en_US.UTF-8\n' >/etc/locale.conf
printf 'KEYMAP=us\nFONT=ter-132n\n' >/etc/vconsole.conf
