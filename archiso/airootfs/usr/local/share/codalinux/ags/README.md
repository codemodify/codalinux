# AGS / Astal shell (placeholder)

CodaLinux's unified desktop shell will be an [AGS](https://aylur.github.io/ags/) app on [Astal](https://aylur.github.io/astal/).

**This directory is not a working UI.** The live ISO ships an official-repo interim shell (waybar + fuzzel + mako + Coda Settings) so the desktop is usable now. This tree stays the future AGS/Astal layout.

## Packaging constraint

`aylurs-gtk-shell` and `libastal*` are **not** in official Arch repositories. v1 forbids a Coda pacman repo and a default AUR helper ([DESIGN.md](../../DESIGN.md#ags--astal)).

Until a later change decides otherwise:

1. Keep shell source in this tree.
2. Build with official-repo toolchains listed in [`packages/ags-build-deps.txt`](../../packages/ags-build-deps.txt).
3. Do not add `yay -S aylurs-gtk-shell` to the ISO or installer.

## Intended layout

```
src/app.ts             entry (stub)
src/bar/               status bar
src/notifications/     notification daemon UI
src/launcher/          app launcher / control center
```

When AGS actually starts, add `hl.exec_cmd` on `hyprland.start` in `../hypr/hyprland.lua` and stop autostarting waybar/mako/fuzzel as the primary shell.

## Next

See [docs/TODO.md](../../docs/TODO.md) §3.
