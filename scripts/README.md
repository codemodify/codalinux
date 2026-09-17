# Scripts

## Iteration phases

- **Desktop UX trial/error** (hypr configs, AGS, wallpaper): `./scripts/qemu-desktop-dev.sh` — GTK QEMU + virtio-9p share of the host tree. No ISO rebuild. An installed qcow2 comes later. virtio-vga is 1920x1080; Hyprland pins `1920x1080@60` (`preferred` on virtio EDID is 640x480@120).
- **ISO smoke** (boot, greetd, first paint): `./scripts/qemu-boot-test.sh` (same 1920x1080 virtio-vga default)
- **Install + A/B e2e** (abox, local ISO): `./scripts/qemu-install-e2e.sh` — first disk, offline install, reboot, `coda-update core --from-iso`, promote
- **Package / vendor / squashfs changes**: `./scripts/build-iso.sh`

## Inventory

| Script | Purpose |
| --- | --- |
| `compose-package-lists.sh` | Build `archiso/packages.x86_64`, `install/packages.txt`, and the `packages` array in `user_configuration.json` (needs `python3`) |
| `install-system-config.sh` | Build/install system-config* into DEST; helpers CGO=0, **GUI CGO=1** (refuses static/headless) |
| `system-config-gui` | Settings launcher: Wayland + `UITK_PAINT=cpu` (virtio EGL skip); execs CGO binary |
| `check-package-lists.sh` | Reject forbidden / unofficial names |
| `check-wrapper-modes.sh` | Assert coda-* wrappers are +x and listed in archiso `file_permissions` |
| `build-iso.sh` | Native/rootless `mkarchiso`, or Docker/Podman **without sudo**; vendors AGS |
| `vendor-ags.sh` | Compile pinned AGS/Astal into airootfs `/usr/local` (works unprivileged via a staging sysroot) |
| `vendor-hyprbars.sh` | Compile pinned hyprbars against official `hyprland` headers into `/usr/local/lib/hyprland` |
| `coda-install` | Live helper: disk pick / `CODA_INSTALL_DISK=auto`, space check, offline A/B install into OS-A |
| `coda-install-layout.py` | Size planner + `check` / `minimums` (4 GiB core slots + 8 GiB data; ~17 GiB min; recommend 32G) |
| `coda-install-split.py` | Classify live airootfs into core vs desktop file lists (offline) |
| `coda-desktop-mount` | Boot-time merge of `/coda/data/desktop` onto the core slot |
| `coda-install-ab.sh` | GPT ESP+A+B+data, core → A, desktop → data, systemd-boot, post-install |
| `coda-install-post.sh` | After the split: desktop onto `coda-data`, greetd autologin `user`, QGA+sshd on the slot |
| `coda-update` | Product CLI: `core` (Arch → inactive slot + oneshot; `--promote` after reboot), `desktop` (Arch → coda-data), `status` |
| `coda-slot` | Low-level A/B helper (ISO `install` / `boot-test` / `promote`). Prefer `coda-update` |
| `coda-install-verify.sh` | Guest layout / Hyprland-from-data checks (`ID=codalinux` for `--boot`) |
| `qemu-install-e2e.sh` | Host QEMU+QGA loop for steps 1–8 (local ISO, no prompts, no NIC) |
| `coda-pacman-init.sh` | Live keyring oneshot (after graphical; no-ops if already populated) |
| `coda-hyprland` | greetd session wrapper (VM-safe env, execs `start-hyprland`) |
| `coda-ags` | Start or message the vendored AGS shell (`ags run` / `ags toggle`); cds to `$HOME`/`/tmp` first |
| `coda-hypr-ws` | Workspace add/remove, tile↔overlapping-float, and taskbar focus/minimize |
| `coda-hyprlock` | Start hyprlock with `/etc/xdg/hypr/hyprlock.conf` |
| `coda-hyprpaper` | Start hyprpaper with the xdg config (`monitor = *`); log `/tmp/hyprpaper.log` + `/var/log` |
| `coda-wallpaper` | Session wallpaper: hyprpaper first, `swaybg` fallback; logs which backend won |
| `coda-sync-desktop-from-host.sh` | Guest helper: copy hypr / wallpaper / AGS from the 9p share; restart AGS as the seat user (tty2/root must not inherit `/run/user/0`) |
| `import-wallpaper.sh` | Copy a host still into `branding/wallpapers/default.png` (never generates) |
| `gen-wallpaper.py` | Fallback still only; refuses to overwrite a committed PNG |
| `coda-settings` | Settings hub helpers: display, hypr gaps/animations, workspaces, wallpaper, Wi-Fi, BT, audio, power, about |
| `coda-sandbox` | User-owned Arch roots under `~/.coda/sandbox/<env>` (no sudo; one env, many packages; `shell`/`exec` via upstream `bwrap`) |
| `coda-live-setup.sh` | Creates `live` user, empty-password autologin, timezone/locale |
| `qemu-boot-test.sh` | Boot the live ISO under QEMU/KVM + OVMF (preferred automated ISO smoke path; serial + QMP screenshots; virtio-vga 1920x1080) |
| `qemu-desktop-dev.sh` | Interactive GTK QEMU + virtio-9p host share for desktop UX iteration (virtio-vga 1920x1080) |
| `hooks/nvidia.sh` | NVIDIA detect/install placeholder (no-op) |

Run compose + check after editing `packages/*.txt`.
