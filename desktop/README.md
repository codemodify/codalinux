# Desktop session configs

Hyprland + companion configs live here. AGS/Astal source lives in `ags/`.

`scripts/build-iso.sh` copies `hypr/*.conf` to `/etc/skel/.config/hypr` and `/etc/xdg/hypr` on the live image. The archinstall profile should copy the same files into the created user's `~/.config/hypr`.

Do not add AUR helpers or unofficial compositor packages. Hyprland and the hypr* companions are official `extra` packages (see `packages/desktop.txt`).
