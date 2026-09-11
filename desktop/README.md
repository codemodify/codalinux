# Desktop session configs

Hyprland + companion configs live here. AGS/Astal source lives in `ags/` (future shell; not on the ISO).

**Interim live desktop** (official `extra` only): `waybar/` panel, `fuzzel/` launcher, `mako/` notifications, `hypr/hyprpaper.conf` wallpaper, and `applications/coda-settings.desktop` → `scripts/coda-settings`.

The compositor config is Lua (`hypr/hyprland.lua`, Hyprland 0.55+). Companion tools (`hypridle`, `hyprlock`, `hyprpaper`) still use hyprlang `.conf` files.

`scripts/build-iso.sh` copies these trees to `/etc/skel/.config/` and `/etc/xdg/` on the live image.

Do not add AUR helpers or unofficial compositor packages. Hyprland and the hypr* companions are official `extra` packages (see `packages/desktop.txt`).
