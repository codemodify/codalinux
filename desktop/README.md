# Desktop session configs

Hyprland + companion configs live here. AGS/Astal source lives in `ags/`.

The compositor stub is Lua (`hypr/hyprland.lua`, Hyprland 0.55+). Hyprland 0.56 warns on the old hyprlang `hyprland.conf`; 0.57 removes that format. Companion tools (`hypridle`, `hyprlock`, `hyprpaper`) still use their own hyprlang `.conf` files.

`scripts/build-iso.sh` copies those files to `/etc/skel/.config/hypr` and `/etc/xdg/hypr` on the live image. The archinstall profile should copy the same files into the created user's `~/.config/hypr`.

Do not add AUR helpers or unofficial compositor packages. Hyprland and the hypr* companions are official `extra` packages (see `packages/desktop.txt`).
