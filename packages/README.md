# Package lists

These text files are the **source of truth** for CodaLinux package names. One official Arch package per line. Blank lines and `#` comments are ignored.

Compose default sets into ISO and installer consumers:

```bash
./scripts/compose-package-lists.sh
./scripts/check-package-lists.sh
```

## Files

| File | In default compose? | Purpose |
| --- | --- | --- |
| `base.txt` | yes | Kernel, firmware, microcode, filesystem tools, sudo |
| `hardware.txt` | yes | Mesa, PipeWire, BlueZ |
| `network.txt` | yes | iwd + resolved-related tools (not NetworkManager) |
| `desktop.txt` | yes | Hyprland, greetd, hypr*, portals, XWayland |
| `apps.txt` | yes | foot, Thunar, Firefox, mpv, imv, Zathura |
| `live.txt` | ISO only | archiso-mandatory + archinstall + live recovery tools |
| `ags-build-deps.txt` | no | Official-repo deps to *build* AGS/Astal from `desktop/ags/` |
| `nvidia.txt` | no | Proprietary NVIDIA path (hook later) |
| `optional-cups.txt` | no | Printing stack, not base |

## Policy

- Official Arch `core` / `extra` names only.
- No AUR packages (`aylurs-gtk-shell`, `libastal*`, `xlibre-*`, …).
- No AUR helpers, Calamares, NetworkManager, firewalld/ufw, or Plymouth.
- No Coda-specific package names — this repo does not publish a pacman repo.
- Pin pacman providers so unattended builds never prompt: `iptables`, `pipewire-jack`, `tesseract-data-eng`.

`scripts/check-package-lists.sh` greps for the known-forbidden names above and requires the provider pins.
