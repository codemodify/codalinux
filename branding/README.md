# Branding

Light v1 branding: identify the OS as CodaLinux without a full theme product.

| File | Installed path |
| --- | --- |
| `os-release` | `/usr/lib/os-release` (via pacman hook; `/etc/os-release` stays a symlink) |
| `issue`, `issue.net` | `/etc/issue`, `/etc/issue.net` |
| `hooks/codalinux-os-release.hook` | `/etc/pacman.d/hooks/` |
| `hooks/apply-os-release.sh` | `/usr/local/lib/codalinux/` |
| `wallpapers/default.png` | `/usr/share/backgrounds/codalinux/default.png` (hyprpaper + hyprlock) |
| `themes/` | greetd / GTK / Hyprland token placeholders |

Plymouth is deferred — do not add splash themes here.

`scripts/build-iso.sh` copies these files into the archiso overlay. The installer profile must install the same hook on the target system so `pacman -Syu filesystem` does not revert the name to Arch Linux.
