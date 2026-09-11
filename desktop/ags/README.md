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
Bar.tsx                 top bar / toolbar
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

Control-center tiles launch official apps: `impala` (Wi-Fi / iwd), `blueman-manager` or `bluetui`, `pavucontrol`, `snapshot`, `nwg-look`, plus the input-help text.

## Build deps

Official Arch packages only, listed in [`packages/ags-build-deps.txt`](../../packages/ags-build-deps.txt). That list is **not** composed into the live ISO (keeps meson/npm/go off the image). `vendor-ags.sh` installs them on the Arch ISO builder, then compiles.

Do not add `aylurs-gtk-shell`, `libastal*`, an AUR helper, or a `[codalinux]` repo.
