#!/usr/bin/env bash
# Last chance inside mkarchiso's airootfs (if this profile hook still runs).
# Bake linker cache + locale so live sysinit/greetd do not.
set -euo pipefail

# mkarchiso runs this as root in the airootfs; ldconfig -X is OK here.
if command -v ldconfig >/dev/null 2>&1; then
  ldconfig -X
fi

if [[ -f /etc/locale.gen ]] && grep -q '^#en_US.UTF-8 UTF-8' /etc/locale.gen; then
  sed -i 's/^#en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen
fi
if command -v locale-gen >/dev/null 2>&1; then
  if ! locale -a 2>/dev/null | grep -qi 'en_US.utf8\|en_US.UTF-8'; then
    locale-gen en_US.UTF-8 >/dev/null 2>&1 || locale-gen || true
  fi
fi

# systemd ConditionNeedsUpdate=/etc compares /usr mtime to this stamp.
touch /etc/.updated /var/.updated
if [[ /usr -nt /etc/.updated ]]; then
  touch /etc/.updated /var/.updated
fi
