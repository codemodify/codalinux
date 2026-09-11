#!/usr/bin/env bash
# First-boot / every-boot live ISO setup: locale, timezone, live user.
# Do not maintain a static /etc/passwd overlay — that wipes package users
# (greeter, systemd-*, etc.). Create the session user here instead.
set -euo pipefail

log() { printf 'coda-live-setup: %s\n' "$*"; }

# Bozeman, Montana defaults (no installer questions for these).
ln -sfn /usr/share/zoneinfo/America/Denver /etc/localtime
if [[ -f /etc/locale.gen ]] && grep -q '^#en_US.UTF-8 UTF-8' /etc/locale.gen; then
  sed -i 's/^#en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen
fi
if ! locale -a 2>/dev/null | grep -qi 'en_US.utf8\|en_US.UTF-8'; then
  locale-gen en_US.UTF-8 >/dev/null 2>&1 || locale-gen || true
fi
printf 'LANG=en_US.UTF-8\n' >/etc/locale.conf
printf 'KEYMAP=us\nFONT=ter-132n\n' >/etc/vconsole.conf

if ! id -u live >/dev/null 2>&1; then
  useradd -m -u 1000 -U -s /usr/bin/bash -c 'CodaLinux live session' live
  log "created user live"
fi

# Empty-password autologin for the live session only.
# video/render help DRM clients; hyprpaper still GBM-crashes on this VM
# path — wallpaper uses swaybg fallback (coda-wallpaper), not groups alone.
passwd -d live >/dev/null
usermod -aG wheel,video,audio,input,render,storage,lp,optical,users live 2>/dev/null \
  || usermod -aG wheel,video,audio,input,users live || true

install -d -o live -g live -m 0755 /home/live
shopt -s nullglob
copy_xdg_config() {
  local name="$1"
  local dest="/home/live/.config/${name}"
  install -d -o live -g live -m 0700 "${dest}"
  local files=(/etc/xdg/"${name}"/*)
  if ((${#files[@]})) && [[ -e "${files[0]}" ]]; then
    install -o live -g live -m 0644 "${files[@]}" "${dest}/"
  fi
}
copy_xdg_config hypr
# Hyprland 0.56+ warns on legacy hyprland.conf; 0.57 removes it.
rm -f /home/live/.config/hypr/hyprland.conf
chown -R live:live /home/live

printf 'live ALL=(ALL:ALL) NOPASSWD: ALL\n' >/etc/sudoers.d/live
chmod 0440 /etc/sudoers.d/live

install -d -m 0755 /var/log
touch /var/log/coda-hyprland.log
chmod 0666 /var/log/coda-hyprland.log

log "timezone=America/Denver locale=en_US.UTF-8 keymap=us user=live"
