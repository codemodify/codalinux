# Desktop session configs

Hyprland + companion configs live here. The unified shell is AGS/Astal in `ags/` (vendored into `/usr/local` at ISO image-build time).

**Live desktop:** AGS top bar (workspaces, per-window taskbar on the active workspace, tile/stack toggle), launcher, notifications, and control center. Official settings apps (`impala`, `blueman` / `bluetui`, `pavucontrol`, `snapshot`, `nwg-look`) are opened from that shell. Wallpaper is Plasma Horos via `coda-wallpaper` (hyprpaper, then swaybg) → `/usr/share/backgrounds/codalinux/default.png` (same path for hyprlock).

Workspace add/remove and per-workspace tiling ↔ **overlapping float** go through `scripts/coda-hypr-ws`. That is hyprfloat-style float mode (all windows on the workspace float and cascade), not Hyprland tabbed groups. Floating windows get hyprbars titlebars (close / maximize / minimize) and are resizable from borders/corners; tiled windows stay border-only. hyprbars is vendored from hyprland-plugins at ISO build time (`scripts/vendor-hyprbars.sh`). Window borders and light shadows are set in `hypr/hyprland.lua`.

The compositor config is Lua (`hypr/hyprland.lua`, Hyprland 0.55+). Companion tools (`hypridle`, `hyprlock`, `hyprpaper`) still use hyprlang `.conf` files.

`scripts/build-iso.sh` copies these trees onto the live image and compiles AGS via `scripts/vendor-ags.sh`. For hypr / AGS / wallpaper trial/error, boot the existing ISO with `scripts/qemu-desktop-dev.sh` and copy from the 9p share instead of rebuilding.

Do not add AUR helpers or unofficial compositor packages. Hyprland and the hypr* companions are official `extra` packages (see `packages/desktop.txt`). Do not ship Waybar, fuzzel, or mako as the default shell.
