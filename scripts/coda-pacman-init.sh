#!/usr/bin/env bash
# Live keyring init. Runs after graphical.target (pacman-init.timer) so
# greetd/Hyprland do not wait. Skip if the keyring is already populated
# (ISO build may bake /etc/pacman.d/gnupg into the squashfs).
set -euo pipefail

gpgdir=/etc/pacman.d/gnupg
if [[ -s "${gpgdir}/pubring.kbx" || -s "${gpgdir}/pubring.gpg" ]]; then
  if [[ -e "${gpgdir}/trustdb.gpg" ]]; then
    echo "coda-pacman-init: keyring already populated, skipping"
    exit 0
  fi
fi

/usr/bin/pacman-key --init
/usr/bin/pacman-key --populate
