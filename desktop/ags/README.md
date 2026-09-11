# AGS / Astal shell (v1)

CodaLinux's unified desktop shell is an [AGS](https://aylur.github.io/ags/) 3 app on [Astal](https://aylur.github.io/astal/). It is the default live-session bar, launcher, notification daemon, and control center. Waybar / fuzzel / mako were rejected as an interim shell and are not on the default path.

## Pinned upstream (vendor at ISO image-build time)

AGS and libastal are **not** in official Arch repositories. v1 still forbids a Coda pacman repo and a default AUR helper. The live ISO therefore **compiles from source** during `scripts/build-iso.sh` and installs into `/usr/local` (non-pacman prefix).

| Project | Version | Git commit | Source archive |
| --- | --- | --- | --- |
| [AGS](https://github.com/Aylur/ags) | 3.1.2 | `bbee2f18939f1ec7ff720e717cf305e73635628f` (2026-04-08) | `https://github.com/Aylur/ags/archive/<commit>.tar.gz` |
| [Astal](https://github.com/Aylur/astal) | tree at pin | `ae8dc0acc66932171ec70d347a8cab9310ce74e4` (2026-09-07) | `https://github.com/Aylur/astal/archive/<commit>.tar.gz` |

Pins live in [`scripts/vendor-ags.sh`](../../scripts/vendor-ags.sh). Bump both the table and that script together.

Vendored Astal libraries (official-repo build deps only):

- `lib/astal/io`, `lib/astal/gtk4`
- `lib/apps`, `lib/hyprland`, `lib/notifd` (`-Dcli=false`; the CLI needs in-tree `quarrel`)
- `lib/bluetooth`, `lib/wireplumber`, `lib/battery`

**Not vendored:**

- `lib/network` — wraps NetworkManager / `nmcli`. CodaLinux is systemd-networkd + iwd; Wi-Fi UI is `impala`.
- `lib/tray` — needs AUR `appmenu-glib-translator`.

## Layout

```
app.tsx                 AGS entry (`ags run` looks for app.ts/tsx)
Bar.tsx                 top bar (workspaces +/−, tile/stack, running-app taskbar)
Launcher.tsx            application launcher
Notification*.tsx       notification popups (Astal notifd)
ControlCenter.tsx       settings surface
Clipboard.tsx           cliphist picker
style.css               shell theme
```

## Runtime on the live image

- Binary: `/usr/local/bin/ags` plus Astal shared libraries / typelibs under `/usr/local`.
- App sources: `/usr/local/share/codalinux/ags`.
- Wrapper: `/usr/local/bin/coda-ags` sets `GI_TYPELIB_PATH` / `LD_LIBRARY_PATH` and runs `ags run` (instance name `coda`).
- Hyprland starts `coda-ags` on `hyprland.start`. Super+Space / Super+D toggles the launcher; Super+, toggles the control center.
- The bar lists running Hyprland clients (grouped by app class). Click focuses and switches workspace; click again on the focused single window minimizes it to `special:minimized`. Multi-window groups cycle on click.
- Workspace `+` / `−` and the Tile/Stack control call `/usr/local/bin/coda-hypr-ws` (same helper as Super+N / Super+− / Super+T). **Stack** is floating/overlapping windows on the current workspace (hyprfloat-style float mode via `hyprctl` / Lua dispatchers). Tabbed Hyprland window groups are not used. New windows on a stacked workspace are floated by `coda-hypr-ws apply-new` from `hyprland.lua`.
- Floating windows show hyprbars titlebars (close / maximize / minimize). Minimize matches the taskbar (`coda-hypr-ws minimize`). Tiled windows hide the bar.

Control-center tiles launch official apps: `impala` (Wi-Fi / iwd), `blueman-manager` or `bluetui`, `pavucontrol`, `snapshot`, `nwg-look`, plus the input-help text.

## Build deps

Official Arch packages only, listed in [`packages/ags-build-deps.txt`](../../packages/ags-build-deps.txt) (includes `glib2-devel` for `glib-mkenums` used by Astal WirePlumber). That list is **not** composed into the live ISO (keeps meson/npm/go off the image). `vendor-ags.sh` installs them on the Arch ISO builder, then compiles.

Unprivileged ISO builds cannot write `/usr/local`. The vendor script installs each Astal library into a writable staging sysroot (`$CACHE/stage`) with `--prefix=/usr/local` (so typelibs keep live soname paths), rewrites only the staging `.pc` `prefix=` lines, and wraps `valac` with `--vapidir`/`--girdir` so the next library can resolve `astal-io-0.1`. DESTDIR `/usr/local` `.pc` files stay `prefix=/usr/local`.

Do not add `aylurs-gtk-shell`, `libastal*`, an AUR helper, or a `[codalinux]` repo.
