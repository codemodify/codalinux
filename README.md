# CodaLinux

CodaLinux is an [Arch Linux](https://archlinux.org/)-based rolling desktop distribution. It ships a custom [Hyprland](https://hypr.land/) compositor session and an [AGS](https://aylur.github.io/ags/)/[Astal](https://aylur.github.io/astal/) unified shell. The installed system tracks official Arch repositories; this tree only holds branding, session layout, live ISO profile stubs, and an [archinstall](https://archlinux.org/packages/extra/any/archinstall/) profile.

This repository is a **v1 scaffold**. It captures locked architecture decisions and a conventional directory layout so the ISO, installer profile, and desktop shell can be implemented without re-litigating the stack.

## Locked defaults

| Area | v1 choice |
| --- | --- |
| Base | Arch Linux, UEFI-only, rolling |
| Bootloader | systemd-boot |
| Kernel / initramfs | stock `linux` + mkinitcpio + CPU microcode |
| Init | systemd |
| Root filesystem | ext4 |
| Packages | `pacman` + official Arch repos only (no Coda repo, no default AUR helper) |
| Network | systemd-networkd + iwd (not NetworkManager; no firewall by default) |
| Display | Wayland + XWayland default; XLibre (X11) session path supported later |
| Display manager | greetd (placeholder greeter) |
| Desktop | Hyprland + AGS/Astal, plus hyprlock / hypridle / hyprpaper / portals |
| Delivery | archiso live ISO + archinstall (not Calamares) |
| Support | GitHub issues |

The authoritative write-up is **[DESIGN.md](DESIGN.md)**. Implementation backlog is **[docs/TODO.md](docs/TODO.md)**.

## Repository layout

```
packages/     Source-of-truth package lists (official Arch names only)
archiso/      archiso profile stubs (UEFI + systemd-boot live ISO)
install/      archinstall config + custom profile stubs
desktop/      Hyprland, hypr* companions, AGS/Astal app layout
branding/     os-release, wallpaper/theme placeholders
sessions/     Wayland Hyprland session + XLibre path notes
scripts/      ISO compose/build helpers and NVIDIA hook placeholder
docs/         Implementation TODOs
```

Layout assumptions (airootfs overlay timing, how lists are composed, why AGS is in-tree) are recorded in [DESIGN.md](DESIGN.md#repository-layout-assumptions).

## Quick start

### Read the decisions

1. Skim [DESIGN.md](DESIGN.md) before changing the stack.
2. Edit package sets under [`packages/`](packages/README.md). Do not add AUR packages, unofficial repos, or a Coda `pacman` repo.
3. Keep ISO and installer consumers in sync with `scripts/compose-package-lists.sh`.

### Validate lists

```bash
./scripts/check-package-lists.sh
```

The checker rejects known policy violations (AUR helpers, Calamares, NetworkManager, firewalls, AGS/XLibre binary names that are not in official repos).

### Build a live ISO (not production-ready)

```bash
./scripts/build-iso.sh
```

On Arch this runs `mkarchiso`. On other hosts it uses a privileged `archlinux` Docker/Podman container when available. Output: `out/codalinux-<date>-x86_64.iso`.

The profile is intended to be buildable (mkinitcpio-archiso, pacman-init, UEFI systemd-boot, greetd → Hyprland, archinstall via `coda-install`). A first successful image and a hardware boot are still follow-ups — see [docs/TODO.md](docs/TODO.md).

### Install with archinstall

From a CodaLinux (or Arch) live environment, once the profile is wired up:

```bash
archinstall --config /path/to/codalinux/install/user_configuration.json
```

The JSON encodes CodaLinux defaults (systemd-boot, ext4, PipeWire, official-repo package set). Disk layout is machine-specific and must be filled in. greetd + systemd-networkd + iwd are **not** first-class archinstall options today; [`install/`](install/README.md) describes the custom profile and post-install hooks still to be written.

## What this repo does not contain

- A second package repository or binary package pipeline
- A default AUR helper
- A production AGS/Astal UI
- Working NVIDIA auto-detection (hook + package list only)
- Plymouth (explicitly deferred)
- Calamares

## Support

Use [GitHub issues](https://github.com/codemodify/codalinux/issues) for bugs, hardware reports, and design questions. There is no separate forum or tracker in v1.
