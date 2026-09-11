# Scripts

## Iteration phases

- **Desktop UX trial/error** (hypr configs, AGS, wallpaper): `./scripts/qemu-desktop-dev.sh` — GTK QEMU + virtio-9p share of the host tree. No ISO rebuild. An installed qcow2 comes later.
- **ISO smoke** (boot, greetd, first paint): `./scripts/qemu-boot-test.sh`
- **Package / vendor / squashfs changes**: `./scripts/build-iso.sh`

## Inventory

| Script | Purpose |
| --- | --- |
| `compose-package-lists.sh` | Build `archiso/packages.x86_64`, `install/packages.txt`, and the `packages` array in `user_configuration.json` (needs `python3`) |
| `check-package-lists.sh` | Reject forbidden / unofficial names |
| `build-iso.sh` | Native/rootless `mkarchiso`, or Docker/Podman **without sudo**; vendors AGS |
| `vendor-ags.sh` | Compile pinned AGS/Astal into airootfs `/usr/local` (works unprivileged via a staging sysroot) |
| `vendor-hyprbars.sh` | Compile pinned hyprbars against official `hyprland` headers into `/usr/local/lib/hyprland` |
| `coda-install` | Live helper: Bozeman locale defaults; disk is the only prompt |
| `coda-hyprland` | greetd session wrapper (VM-safe env, execs `start-hyprland`) |
| `coda-ags` | Start or message the vendored AGS shell (`ags run` / `ags toggle`) |
| `coda-hypr-ws` | Workspace add/remove, tile↔overlapping-float, and taskbar focus/minimize |
| `coda-hyprlock` | Start hyprlock with `/etc/xdg/hypr/hyprlock.conf` |
| `coda-hyprpaper` | Start hyprpaper with the xdg config (`monitor = *`); log `/tmp/hyprpaper.log` + `/var/log` |
| `coda-wallpaper` | Session wallpaper: hyprpaper first, `swaybg` fallback; logs which backend won |
| `coda-sync-desktop-from-host.sh` | Guest helper: copy hypr / wallpaper / AGS from the 9p share and restart |
| `import-wallpaper.sh` | Copy a host still into `branding/wallpapers/default.png` (never generates) |
| `gen-wallpaper.py` | Fallback still only; refuses to overwrite a committed PNG |
| `coda-settings` | Open official Wi-Fi / BT / audio / webcam / appearance tools |
| `coda-live-setup.sh` | Creates `live` user, empty-password autologin, timezone/locale |
| `qemu-boot-test.sh` | Boot the live ISO under QEMU/KVM + OVMF (preferred automated ISO smoke path; serial + QMP screenshots) |
| `qemu-desktop-dev.sh` | Interactive GTK QEMU + virtio-9p host share for desktop UX iteration |
| `hooks/nvidia.sh` | NVIDIA detect/install placeholder (no-op) |

Run compose + check after editing `packages/*.txt`.
