# Hyprland session config

Lua compositor config is `hyprland.lua` (Hyprland 0.55+). Companions still use hyprlang: `hyprpaper.conf`, `hyprlock.conf`, `hypridle.conf`.

## Stack = overlapping float

Super+T and the AGS Tile/Stack control run `coda-hypr-ws toggle-stack`. That floats every window on the current workspace and cascades them. It is **not** Hyprland tabbed groups.

## Floating titlebars (hyprbars)

Official [hyprbars](https://github.com/hyprwm/hyprland-plugins/tree/main/hyprbars) from hyprland-plugins. Not in Arch `extra`; vendored at ISO build time by [`scripts/vendor-hyprbars.sh`](../../scripts/vendor-hyprbars.sh).

| Pin | Value |
| --- | --- |
| hyprland-plugins commit | `722f15a77768eab13f01f5e5dce024bd2f61f270` (2026-09-04, hyprbars chase) |
| Plugin path | `/usr/local/lib/hyprland/libhyprbars.so` |
| Load | `hl.plugin.load(...)` at the top of `hyprland.lua` |
| Buttons | close → `hl.dsp.window.close()`; max → maximized fullscreen toggle; min → `coda-hypr-ws minimize` (`special:minimized`) |
| Tiled windows | `hyprbars:no_bar` when `float = false` |

Buttons are registered with `hl.plugin.hyprbars.add_button` (Lua). Do not use the hyprlang `hyprbars-button` keyword — Hyprland 0.55’s legacy parser does not call that handler.

Build deps are official only (`packages/hyprbars-build-deps.txt`), including `hyprland` so headers match the compositor on the image. **Do not** use `hyprpm` on the live ISO (that needs a compiler). **Do not** add AUR plugin packages.

hyprbars is drawn inside Hyprland (cairo/pango decoration), not a separate Wayland client. It should follow the VirtualBox pixman path in `coda-hyprland`. If the plugin fails to load, floating still works — you just lose titlebar buttons (usually an ABI mismatch: rebuild the ISO so vendor-hyprbars and pacstrap see the same `hyprland`).

Blur on the bar is left off (`bar_blur = false`) because live VMs use software rendering.
