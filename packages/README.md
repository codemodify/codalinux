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
| `base.txt` | yes | Kernel, firmware, microcode, filesystem tools, sudo, openssh, qemu-guest-agent |
| `hardware.txt` | yes | Mesa, PipeWire, BlueZ |
| `network.txt` | yes | iwd + resolved-related tools (not NetworkManager) |
| `desktop.txt` | yes | Hyprland, greetd, hypr*, portals, AGS runtime, system-config-gui Wayland/X11 libs |
| `apps.txt` | yes | foot, Thunar, Firefox, mpv, imv, Zathura, settings apps |
| `sandbox.txt` | yes | `bubblewrap` — app isolation backbone (`coda-sandbox`) |
| `live.txt` | ISO only | archiso-mandatory + live recovery tools (`openssh` / `qemu-guest-agent` also in `base.txt` so install keeps them). `archinstall` remains on the ISO but is not the first-install path. |
| `ags-build-deps.txt` | no | Official-repo deps to *compile* AGS/Astal at ISO build time |
| `system-config-gui-build-deps.txt` | no | Official-repo deps to *compile* `system-config-gui` with CGO (Wayland/X11/EGL) |
| `hyprbars-build-deps.txt` | no | Official-repo deps to *compile* hyprbars (includes `hyprland` headers) |
| `nvidia.txt` | no | Proprietary NVIDIA path (hook later) |
| `optional-cups.txt` | no | Printing stack, not base |

## Policy

- Official Arch `core` / `extra` names only.
- No AUR packages (`aylurs-gtk-shell`, `libastal*`, `xlibre-*`, …).
- No AUR helpers, Calamares, NetworkManager, firewalld/ufw, or Plymouth.
- No Coda-specific package names — this repo does not publish a pacman repo.
- Pin pacman providers so unattended builds never prompt: `iptables`, `pipewire-jack`, `tesseract-data-eng`.

AGS/Astal themselves are vendored from source into `/usr/local` by `scripts/vendor-ags.sh` (not listed here). hyprbars is vendored the same way by `scripts/vendor-hyprbars.sh`.

`scripts/check-package-lists.sh` greps for the known-forbidden names above and requires the provider pins.
