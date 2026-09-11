# Desktop session configs

Hyprland + companion configs live here. The unified shell is AGS/Astal in `ags/` (vendored into `/usr/local` at ISO image-build time).

**Live desktop:** AGS top bar (workspaces, running-app taskbar, tile/stack toggle), launcher, notifications, and control center. Official settings apps (`impala`, `blueman` / `bluetui`, `pavucontrol`, `snapshot`, `nwg-look`) are opened from that shell. Wallpaper is `hypr/hyprpaper.conf`.

Workspace add/remove and per-workspace tiling ↔ stacking (Hyprland window groups) go through `scripts/coda-hypr-ws`. Window borders are set in `hypr/hyprland.lua`.

The compositor config is Lua (`hypr/hyprland.lua`, Hyprland 0.55+). Companion tools (`hypridle`, `hyprlock`, `hyprpaper`) still use hyprlang `.conf` files.

`scripts/build-iso.sh` copies these trees onto the live image and compiles AGS via `scripts/vendor-ags.sh`.

Do not add AUR helpers or unofficial compositor packages. Hyprland and the hypr* companions are official `extra` packages (see `packages/desktop.txt`). Do not ship Waybar, fuzzel, or mako as the default shell.
