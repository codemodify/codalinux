# Scripts

| Script | Purpose |
| --- | --- |
| `compose-package-lists.sh` | Build `archiso/packages.x86_64`, `install/packages.txt`, and the `packages` array in `user_configuration.json` (needs `python3`) |
| `check-package-lists.sh` | Reject forbidden / unofficial names |
| `build-iso.sh` | Native/rootless `mkarchiso`, or Docker/Podman **without sudo**; vendors AGS |
| `vendor-ags.sh` | Compile pinned AGS/Astal into airootfs `/usr/local` (official-repo deps) |
| `coda-install` | Live helper: Bozeman locale defaults; disk is the only prompt |
| `coda-hyprland` | greetd session wrapper (VM-safe env, execs `start-hyprland`) |
| `coda-ags` | Start or message the vendored AGS shell (`ags run` / `ags toggle`) |
| `coda-settings` | Open official Wi-Fi / BT / audio / webcam / appearance tools |
| `coda-live-setup.sh` | Creates `live` user, empty-password autologin, timezone/locale |
| `hooks/nvidia.sh` | NVIDIA detect/install placeholder (no-op) |

Run compose + check after editing `packages/*.txt`.
