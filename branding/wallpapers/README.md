# Wallpapers

Install path: `/usr/share/backgrounds/codalinux/`.

`default.png` is **Plasma Horos** (Nuno Pinheiro), the classic KDE 4 / Oxygen still. It is **not** a CodaLinux original. The live image uses a 2560×1440 PNG resized from the Plasma Oxygen 5K file `/usr/share/wallpapers/Horos/contents/images/5120x2880.png` (same asset the user attached from their abox KDE host).

`coda-wallpaper` (hyprpaper, then swaybg) and hyprlock both load `/usr/share/backgrounds/codalinux/default.png`.

`scripts/build-iso.sh` copies this file if it exists. `scripts/gen-wallpaper.py` is a last-resort fallback and **will not overwrite** a committed PNG (pass `--force` only if you intend to replace Horos). Refresh from a host path with `scripts/import-wallpaper.sh`.
