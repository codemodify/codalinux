# Display-manager sessions

greetd (and any later greeter) should list sessions from:

| Path on the image | Source |
| --- | --- |
| `/usr/share/wayland-sessions/codalinux-hyprland.desktop` | `wayland/codalinux-hyprland.desktop` |
| `/usr/share/xsessions/codalinux-x11.desktop` | `xlibre/codalinux-x11.desktop` (not installed by default) |

Default: Wayland Hyprland + XWayland. The XLibre `.desktop` file is a stub and is **not** copied onto the ISO until packaging is decided — see `xlibre/README.md`.
