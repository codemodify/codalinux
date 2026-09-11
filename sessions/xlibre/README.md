# XLibre (X11) session path

Locked product decision: CodaLinux supports an XLibre session path **in addition to** the default Wayland + XWayland session.

## Packaging (unresolved, do not paper over)

XLibre is **not** in official Arch `core`/`extra`. Upstream options today:

- AUR packages (`xlibre-xserver`, `xlibre-meta`, …)
- A third-party pacman repo (`[xlibre-stable]`) documented by the XLibre Arch packagers

CodaLinux v1 **must not**:

- enable that third-party repo in `archiso/pacman.conf` or installer configs
- add `xlibre-*` to default package lists
- ship an AUR helper to pull XLibre
- invent a Coda package repo to wrap it

Default X11 application support is **XWayland** on Hyprland. The `.desktop` stub in this directory is for a future session only.

A later change must pick one of: source build, optional operator-enabled upstream repo, or wait for official packages. Track that in [docs/TODO.md](../../docs/TODO.md).
