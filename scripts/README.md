# Scripts

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
| `import-wallpaper.sh` | Copy a host still into `branding/wallpapers/default.png` (never generates) |
| `gen-wallpaper.py` | Fallback still only; refuses to overwrite a committed PNG |
| `coda-settings` | Open official Wi-Fi / BT / audio / webcam / appearance tools |
| `coda-live-setup.sh` | Creates `live` user, empty-password autologin, timezone/locale |
| `qemu-boot-test.sh` | Boot the live ISO under QEMU/KVM + OVMF (preferred automated test path; serial + QMP screenshots) |
| `hooks/nvidia.sh` | NVIDIA detect/install placeholder (no-op) |

Run compose + check after editing `packages/*.txt`.
