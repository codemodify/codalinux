# CodaLinux v1 design decisions

This document is the **decision log** for locked v1 choices. Do not contradict it in ISO profiles, installer configs, package lists, or desktop stubs. If a later decision changes the stack, update this file in the same change.

The canonical **system picture** (partitions → layers → `/` folders → sandboxes → update model, with **Target** vs **Current tree**) is [architecture.md](architecture.md). Do not claim A/B slots or a core-only ISO work until they are built.

CodaLinux is a rolling Arch Linux derivative: it does not fork the base system. Periodic live ISO rebuilds are the delivery cadence.

**App model (locked):** day-to-day packages must not pollute the host OS. Host `pacman` is for the **core OS** (rare; gated later). Extra software is installed with `pacman --root` into disposable trees and run with **upstream [bubblewrap](https://github.com/containers/bubblewrap)** (`bwrap`, LGPL-2.1-or-later). `coda-sandbox` is the user-facing create / install / enter / destroy tool. **One named sandbox is one Arch root that holds many packages** (e.g. `dev` with `postgresql`, `redis`, `git`) — not one sandbox per app. Trees live under `~/.coda/sandbox/<name>/` (user-owned; no sudo). This supersedes “one mutable ext4 root + rolling pacman for everything” as the long-term app story. The current live/install image still ships the Hyprland + AGS desktop for v1; it is not yet a minimal core-only image. Read-only A/B core slots are the target, not something this tree implements yet. Picture: [architecture.md](architecture.md). Decisions below: [Core, desktop, and sandboxes](#core-desktop-and-sandboxes).

## Locked stack

### Base system

| Decision | Choice | Notes |
| --- | --- | --- |
| Upstream | Arch Linux | Rolling; no Coda release freeze |
| Firmware interface | UEFI-only | No BIOS/legacy boot support in ISO or installer |
| Bootloader | systemd-boot | Installed system and live ISO |
| Kernel | stock `linux` | Same as Arch; no custom kernel package |
| Initramfs | mkinitcpio | Same hooks/presets as Arch unless a later change says otherwise |
| Microcode | `intel-ucode` + `amd-ucode` | Both installed; the CPU uses the matching image |
| Init | systemd | |
| Default root filesystem | ext4 | Installer may offer other filesystems later; v1 default is ext4 |
| Architecture | x86_64 | First and only v1 target |

### Packaging

| Decision | Choice |
| --- | --- |
| Package manager | pacman |
| Repositories | Official Arch `core` and `extra` only |
| Coda package repo | **None** — do not add a `[codalinux]` repo to `pacman.conf` |
| AUR helper | **None by default** — do not ship yay/paru/pamac |
| Host pacman | **Core / OS only** — kernel, firmware, boot, session stack. Rare after install. |
| App / extra pacman | **`pacman --root` inside `coda-sandbox`** (one named root, many packages) |
| Isolation | Upstream **bubblewrap** (`bwrap`). Do not reimplement it. |
| Docker / Distrobox | **Not required** for v1. Optional later, alongside bwrap. |
| Firejail | **Not** the primary sandbox. |
| Multilib | **Off by default** — enable only if a later decision requires 32-bit NVIDIA userspace |
| Security | Stock Arch defaults (pacman signature policy, no extra MAC/firewall product) |

Package names that are **not** in official repositories must not appear in default lists. That includes AGS/Astal binaries and XLibre. See [Out-of-repo components](#out-of-repo-components). `bubblewrap` is official `extra` and lives in [`packages/sandbox.txt`](packages/sandbox.txt).

### Hardware and graphics

| Decision | Choice |
| --- | --- |
| Firmware | `linux-firmware` |
| Open GPU stack | Mesa (Vulkan ICD packages for Intel/AMD as needed) |
| Audio | PipeWire + WirePlumber (`pipewire`, `pipewire-audio`, `pipewire-pulse`, `wireplumber`) |
| Bluetooth | BlueZ (`bluez`, `bluez-utils`) |
| NVIDIA proprietary | Optional install path when an NVIDIA GPU is detected — **not implemented yet** |
| Printing | CUPS optional; **not** in the base set |

NVIDIA work is limited to [`packages/nvidia.txt`](packages/nvidia.txt) and [`scripts/hooks/nvidia.sh`](scripts/hooks/nvidia.sh). Do not land auto-detect logic until that hook is implemented on purpose.

### Networking

| Decision | Choice |
| --- | --- |
| Wired/wireless IP | systemd-networkd |
| Wireless daemon | iwd |
| NetworkManager | **Not used** |
| Firewall | **None by default** (no firewalld, ufw, or custom nftables policy) |
| DNS | systemd-resolved (stock companion to networkd) |

`iwd` must not be configured to do its own IP setup (`EnableNetworkConfiguration=false`) so networkd remains the DHCP client.

archinstall's guided installer historically defaults toward NetworkManager for desktop profiles. CodaLinux must override that via a custom profile and/or post-install steps. See [`install/README.md`](install/README.md).

### Display and desktop

| Decision | Choice |
| --- | --- |
| Default session | Wayland compositor: Hyprland |
| X11 apps on the default session | XWayland (`xorg-xwayland`) |
| Alternate session | XLibre (X11) path — packaging unresolved; not default |
| Display manager | greetd |
| Greeter UI | Stub / custom placeholder (`agreety` until a branded greeter exists) |
| Shell | **AGS/Astal** (vendored from source into `/usr/local` at ISO image-build time). Waybar / fuzzel / mako are **not** the default shell. |
| Companions | hyprlock, hypridle, hyprpaper (swaybg live/VM fallback) |
| Portals | `xdg-desktop-portal-hyprland` + `xdg-desktop-portal-gtk` as needed |
| Branding | Light: `/etc/os-release` as CodaLinux, theme/wallpaper placeholders, greetd theming hooks |
| Plymouth | **Deferred** — do not add splash work in v1 scaffolding |

### Default applications

Official-repo packages only:

- Terminal: `foot`
- File manager: `thunar` (plus conventional GVFS/thumbnail helpers)
- Web: `firefox`
- Video: `mpv`
- Images: `imv`
- Documents: `zathura` + `zathura-pdf-mupdf`
- Settings tools (opened from the AGS control center): `impala`, `blueman` / `bluetui`, `pavucontrol`, `snapshot`, `nwg-look`

### Delivery

| Decision | Choice |
| --- | --- |
| Live image | archiso profile in `archiso/` |
| Installer | archinstall (guided JSON + custom CodaLinux profile) |
| Calamares | **Not used** |
| ISO rebuilds | Periodic, later — not every upstream Arch ISO date |

### Support and docs

| Decision | Choice |
| --- | --- |
| Human docs in-tree | This file + README |
| Support surface | GitHub issues on this repository |
| Extra support products | None in v1 |

## Out-of-repo components

Two locked product pieces are **not** in official Arch repositories today. v1 still forbids a Coda package repo and a default AUR helper, so they are handled as follows.

### AGS / Astal

[`aylurs-gtk-shell`](https://aur.archlinux.org/packages/aylurs-gtk-shell) and Astal libraries are AUR/source-only. They must **not** be listed in `packages/*.txt` default sets.

v1 approach:

1. Keep the shell **source tree** in [`desktop/ags/`](desktop/ags/README.md).
2. At ISO image-build time, [`scripts/vendor-ags.sh`](scripts/vendor-ags.sh) fetches the pinned AGS/Astal commits and compiles them with official-repo toolchains from [`packages/ags-build-deps.txt`](packages/ags-build-deps.txt), installing into airootfs `/usr/local`.
3. Runtime packages on the live image are official only (`gjs`, `gtk4`, `gtk4-layer-shell`, …). Do not ship meson/npm/go on the ISO by default.
4. Do not add `yay -S aylurs-gtk-shell`, a Coda pacman repo, or a default AUR helper.

Pinned commits and the vendored Astal library set are documented in [`desktop/ags/README.md`](desktop/ags/README.md). **Astal Network is not used** (it wraps NetworkManager / `nmcli`). Wi-Fi is iwd via `impala`. A Waybar + fuzzel + mako interim was explicitly rejected.

Official settings apps remain available and are opened from the AGS control center: `impala` (Wi-Fi / iwd), `blueman` or `bluetui`, `pavucontrol`, `snapshot`, `nwg-look`. `blueman` depends on `libnm`; it does **not** install or enable NetworkManager.

### hyprbars

[hyprbars](https://github.com/hyprwm/hyprland-plugins/tree/main/hyprbars) is not an official Arch package. v1 vendors it from the pinned hyprland-plugins commit at ISO build time (`scripts/vendor-hyprbars.sh`) into `/usr/local/lib/hyprland/libhyprbars.so`, compiled against official `hyprland` headers. Do not add `hyprpm` as a live-session workflow and do not list AUR plugin packages.

### XLibre session path

XLibre is not in official Arch repos (AUR and a third-party `[xlibre-stable]` repo exist upstream). CodaLinux **must not** enable that third-party repo in `pacman.conf`.

v1 approach:

1. Default session is Hyprland + XWayland. X11 applications run there.
2. [`sessions/xlibre/`](sessions/xlibre/README.md) holds the future `.desktop` stub and packaging notes.
3. Shipping XLibre later requires an explicit packaging decision (source build, optional user-enabled upstream repo, or waiting for official packages). That decision is **not** made here.

## Core, desktop, and sandboxes

Three layers. Do not collapse them back into “install postgres on the host.”

| Layer | What it is | How it is updated | v1 status |
| --- | --- | --- | --- |
| **Core OS** | Bootable Arch: `base` + `linux` + firmware + mkinitcpio + microcode + systemd + boot | Host pacman, later **gated**; target is read-only **A/B** slots | Documented target. Not implemented. Today’s ISO is still a full desktop image. |
| **Desktop** | Hyprland + vendored AGS/Astal, greetd, portals, official settings apps | Same image as core for now (do not rip out this PR) | Shipped on live/install. May become a slot or a sandbox later. |
| **Apps / extras** | Disposable Arch roots (`pacman --root`) run with `bwrap` | `coda-sandbox pacman` / `install` (repeatable into the same name) | **Default place for extra software.** One name = many packages. |

### Why bubblewrap (not Docker or Firejail)

- **bubblewrap** is a small upstream setuid-optional helper (`extra/bubblewrap`) that sets up user/mount namespaces and bind mounts. Flatpak uses it. License: **LGPL-2.1-or-later**. CodaLinux ships the Arch package; it does not fork or reimplement `bwrap`.
- **Docker / Podman** need a daemon or a heavier image workflow. They are optional later notes, not v1 requirements. Distrobox is the same class.
- **Firejail** is a different policy language and is not the Arch-root + `pacman --root` model.

A named sandbox is **one Arch root**, not one app. Create `dev` once, then `coda-sandbox install dev postgresql`, `coda-sandbox install dev redis git`, and so on. Do not assume a 1:1 app↔sandbox mapping. The tree is not a container image format. Throw it away with `coda-sandbox destroy`. Persist user data (database files, project dirs) with binds under `$HOME` or a data directory so destroy does not take the only copy.

### Folder mapping (target)

```
Core (future RO A/B slots — not implemented)
  /usr          OS userland (read-only when A/B lands)
  /boot         UKI / systemd-boot + kernel
  /etc          Base OS config (or a small writable overlay)

Data (writable, survives OS slot swaps)
  /home         Users (includes ~/.coda/sandbox)
  /var          Logs and host caches
  /etc overlay  Host-specific bits if /etc is split later

Sandboxes (user-owned under $HOME, no sudo)
  ~/.coda/sandbox/<name>/           sandbox directory (documented path)
  ~/.coda/sandbox/<name>/root       pacman --root tree
  ~/.coda/sandbox/<name>/meta
  ~/.coda/cache/pacman              shared cache for this user
```

Default store is **`~/.coda/sandbox`** (singular), not `/var/coda/…` and not XDG `~/.local/share/…`. The cache is shared across this user’s named roots (one download of `base`, many installs). `coda-sandbox` never needs sudo. `pacman --root` runs inside `unshare --map-root-user` so extract sees uid 0 while files on disk stay owned by the real user. `enter` / `run` are unprivileged `bwrap` and bind the sandbox as `/` so **host `/usr` is not the sandbox’s `/usr`**. Override with `--store` / `CODA_SANDBOX_STORE`.

### Phases

1. **Now (this tree):** `bubblewrap` on the desktop live/install image; `coda-sandbox` wired into `/usr/local/bin`; docs. Desktop stays. Host is still a single mutable ext4 root.
2. **Installer:** partition or subvolumes for **core vs data**; put `~/.coda/sandbox` and `/home` on data.
3. **Later:** read-only A/B core images, gated OS updates, optional Distrobox or Flatpak **alongside** bwrap — not instead of it.

Do not claim A/B or a core-only ISO exists until those land. Picture: [architecture.md](architecture.md). Commands: [docs/sandbox.md](docs/sandbox.md).

## Repository layout assumptions

These are scaffolding choices, not product-stack changes. Prefer this conventional Arch-derivative layout unless a later change replaces it.

### Top-level map

| Path | Role |
| --- | --- |
| `architecture.md` | System picture (target vs current) |
| `DESIGN.md` | This decision log |
| `packages/` | Editable source of truth for package names |
| `archiso/` | One archiso **profile** (not a copy of the `archiso` tool) |
| `install/` | archinstall JSON + custom profile stubs |
| `desktop/` | User-session configs and the AGS app tree |
| `branding/` | Files that identify the OS (os-release, issue, themes) |
| `sessions/` | Display-manager session desktop files |
| `scripts/` | Host-side helpers; not installed unless an ISO overlay copies them |

`iso/` is **not** used. The profile lives at `archiso/` because that matches `mkarchiso <profile-dir>` usage and Arch documentation.

### Package list composition

- Lists are plain text, one official package per line. `#` comments and blank lines are ignored.
- [`scripts/compose-package-lists.sh`](scripts/compose-package-lists.sh) concatenates the default sets into `archiso/packages.x86_64`, `install/packages.txt`, and the `packages` array in `install/user_configuration.json`.
- `packages/sandbox.txt` (`bubblewrap`) is in the default compose (live + install).
- `packages/nvidia.txt` and `packages/optional-cups.txt` are **not** in the default compose.
- `packages/ags-build-deps.txt` is **not** in the live ISO default set (build-only; used by `scripts/vendor-ags.sh` on the Arch ISO builder).
- `packages/hyprbars-build-deps.txt` is **not** in the live ISO default set (build-only; used by `scripts/vendor-hyprbars.sh`).
- `archiso/packages.x86_64` also includes archiso-mandatory packages (`mkinitcpio`, `mkinitcpio-archiso`) from `packages/live.txt`.
- Unattended builds pin pacman providers in the default lists (`iptables`, `pipewire-jack`, `tesseract-data-eng`). `iptables` is a provider pin only — it does not enable a firewall.

### Unattended host builds (abox)

`scripts/build-iso.sh` must not prompt for a sudo password or a pacman provider. The script never calls `sudo`.

- Prefer native / rootless `mkarchiso` when it is on `PATH` (archiso 89+ can unshare as a regular user).
- Otherwise use Docker or Podman only if they already work as the current user (`docker info` without sudo). Add the builder to the `docker` group; do not grant passwordless root.
- Optional host sudoers (outside this repo) may allow only `/usr/bin/mkarchiso` and `/usr/bin/docker`.
- `archiso/pacman.conf` sets `NoConfirm`; container `pacman` invocations use `--noconfirm`.

### archiso overlay timing

archiso copies `airootfs/` **before** installing packages. Files owned by packages (notably `/usr/lib/os-release` from `filesystem`) will be overwritten.

Assumption: branding is re-applied with a pacman hook installed from [`branding/hooks/`](branding/hooks/codalinux-os-release.hook) into the airootfs. Templates live under `/usr/local/share/codalinux/` (not packaged paths). The hook writes `os-release` and `issue*` after `filesystem` is installed. `/etc/os-release` remains the usual symlink. Do not pre-copy packaged paths into `airootfs/` — pacstrap will fail with “exists in filesystem”.

### Service enablement

Enabled on live and (via the installer profile) on the installed system:

- `greetd.service` (as the display manager)
- `systemd-networkd.service`
- `systemd-resolved.service`
- `iwd.service`
- `bluetooth.service`

Not enabled: NetworkManager, CUPS, firewalld, Plymouth.

Live ISO service symlinks live under `archiso/airootfs/etc/systemd/system/`. Installed-system enablement is an archinstall profile responsibility.

`pacman-init.service` must **not** be `WantedBy=multi-user.target`. Graphical.target waits on multi-user, so that oneshot (`pacman-key --init` + `--populate`) delayed greetd/Hyprland by tens of seconds. A `pacman-init.timer` (`WantedBy=timers.target`, `OnBootSec=3s`) starts the same job **after** `graphical.target`. The stock `etc-pacman.d-gnupg.mount` tmpfs is masked: it wiped `/etc/pacman.d/gnupg` every boot. `scripts/build-iso.sh` best-effort bakes that keyring (`unshare --map-root-user` when not root; never sudo; failures are logged and the ISO build continues). The live oneshot still populates after greetd if bake skipped. Do not delete pacman-init — `coda-install` / live `pacman` still need a keyring.

`ldconfig.service` is on `sysinit.target` (~30s on QEMU when it runs). Stock glibc conditions are triggering **OR** (`|`): `ConditionNeedsUpdate=|/etc` **or** `ConditionFileNotEmpty=|!/etc/ld.so.cache`. Live overlay boots usually have a cache already (glibc hook / `customize_airootfs.sh` `ldconfig -X`) but **no** `/etc/.updated`, so NeedsUpdate is always true. An empty `ConditionNeedsUpdate=` drop-in resets every prior start condition (systemd.unit(5)), which left the unit with no conditions and systemd always started it (`ConditionResult=yes`, +31s with a 107KB cache present). The drop-in now resets, then sets a required (no `|`) `ConditionFileNotEmpty=!/etc/ld.so.cache`: skip when the cache exists and is non-empty; run only if missing/empty. `root/customize_airootfs.sh` (when mkarchiso still invokes it) runs `ldconfig -X`, `locale-gen en_US.UTF-8`, and `touch /etc/.updated`.

Live systemd-boot `archiso/efiboot/loader/loader.conf` uses `timeout 1` (editor still yes). `coda-live-setup` only runs `locale-gen` when `en_US.UTF-8` is missing.

### Hostname, locale, users

| Item | v1 scaffold default | Overridable |
| --- | --- | --- |
| Hostname | `coda` | Installer |
| Locale | `en_US.UTF-8` (Bozeman / US) | **Not asked** — fixed |
| Console keymap | `us` | **Not asked** — fixed |
| Timezone | `America/Denver` (Bozeman, Montana) | **Not asked** — fixed |
| Live session | greetd autologins user `live` into `/usr/local/bin/coda-hyprland` on tty1; tty2 is a root rescue console | Live ISO |
| Installed users | Created by archinstall credentials file | Yes |

Locale, keymap, and timezone are **Bozeman, Montana defaults**. The live image writes `/etc/localtime` → `America/Denver`, `/etc/locale.conf`, and `/etc/vconsole.conf` via a pacman hook plus `coda-live-setup.service`. `coda-install` / `user_configuration.json` preseed the same values and must not prompt for region, timezone, locale, or keymap. Disk is asked only when `CODA_INSTALL_DISK` is unset; a successful `device_handler` + `suggest_single_disk_layout` path launches `archinstall --silent`. Credentials stay opt-in (`CODA_INSTALL_CREDS` or `CODA_INSTALL_USER` / `CODA_INSTALL_PASSWORD`).

Live overlay size is **`cow_spacesize=4G`** on the systemd-boot entry (tmpfs limit for `/run/archiso/cowspace`). Stock 256M is too small for `coda-sandbox create` (~500M+ `base`).

Live GUI notes:

- Do **not** overlay a minimal `/etc/passwd` or `/etc/shadow` — that wipes package accounts (`greeter`, systemd users). The `live` user is created with `systemd-sysusers` and `coda-live-setup.sh`.
- Root on tty1 was fragile: Hyprland exited in ~1s and greetd fell back to agreety (PAM `SERVICE_ERR` / `pam_securetty`). Autologin is the `live` user via a wrapper.
- `coda-hyprland` sets `XDG_RUNTIME_DIR`, enables software rendering on VMs (`WLR_RENDERER=pixman`, `WLR_NO_HARDWARE_CURSORS=1`, `LIBGL_ALWAYS_SOFTWARE=1`) for VirtualBox VMSVGA, logs to `/var/log/coda-hyprland.log`, and execs `start-hyprland` (not the bare `Hyprland` binary).
- greetd `initial_session` and `default_session` both run the wrapper so a crash retries Hyprland instead of agreety.
- Compositor config is `desktop/hypr/hyprland.lua` (Hyprland 0.55+ Lua). Companion tools still use hyprlang `.conf` (`hypridle` / `hyprlock` / `hyprpaper`).
- The vendored AGS shell autostarts as `coda-ags` (bar with workspaces, running-app taskbar, and tile/stack toggle; launcher; notifications; control center). hyprpaper, the polkit agent, and blueman-applet still start. Super+Space / Super+D toggles the AGS launcher; Super+, toggles the control center; Super+N / Super+= add a workspace; Super+- removes an empty one; Super+T toggles tiling ↔ overlapping float (hyprfloat-style workspace float mode implemented in `coda-hypr-ws`, not tabbed groups and not a hyprpm plugin).
- Floating windows use vendored **hyprbars** (`/usr/local/lib/hyprland/libhyprbars.so`, hyprland-plugins `7644cecdb947060682891a0db2a0cdc5c0b9e704`, the official hyprpm pin for Hyprland 0.56.2) for close / maximize / minimize. Tiled windows keep `hyprbars:no_bar`. Minimize uses `coda-hypr-ws minimize` (`special:minimized`). The plugin is compiled at ISO build time against official `hyprland` headers; do not run `hyprpm` on the live image. Do not track hyprland-plugins `main` — later chases need headers newer than Arch `hyprland` 0.56.2.
- Live wallpaper is Plasma **Horos** (Nuno Pinheiro / Oxygen, not a Coda original) at `branding/wallpapers/default.png` via `coda-wallpaper` (`hyprpaper`, then `swaybg` on the live/VM path) at `/usr/share/backgrounds/codalinux/default.png`. `scripts/gen-wallpaper.py` must not overwrite that file. hyprlock uses the same path. Live hypridle does not lock or DPMS-off on idle (hyprlock dies under VirtualBox/pixman). Super+L runs `coda-hyprlock`. If hyprlock still crashes: `hyprctl --instance 0 eval 'hl.clear_crashed_lockscreen()'` and `killall -9 hyprlock`.

Swap (partition vs zram vs none) is **not** locked. The archinstall JSON currently leaves `swap` at `true` as an installer default only.

### Boot modes

The live ISO is UEFI + systemd-boot only. `profiledef.sh` uses the combined `uefi.systemd-boot` bootmode from current archiso. If a build host has an older `mkarchiso` that still wants the split names, replace with:

```text
uefi-x64.systemd-boot.esp
uefi-x64.systemd-boot.eltorito
```

Do not add `bios.syslinux.*`.

### Desktop file install paths

| File | Destination on the image |
| --- | --- |
| `sessions/wayland/codalinux-hyprland.desktop` | `/usr/share/wayland-sessions/` |
| `desktop/hypr/*` (`hyprland.lua` + companion `.conf`) | `/etc/skel/.config/hypr/` and `/etc/xdg/hypr/` |
| `desktop/ags/` | `/usr/local/share/codalinux/ags/` (UI sources; AGS binary is vendored to `/usr/local`) |
| `desktop/applications/*.desktop` | `/usr/share/applications/` |
| `branding/os-release` | `/usr/lib/os-release` via hook |
| `branding/wallpapers/default.png` | `/usr/share/backgrounds/codalinux/` |

`build-iso.sh` is responsible for copying those trees into `airootfs/` at build time so we do not maintain duplicates.

## Explicit non-goals (v1 scaffold)

- A production-quality ISO in CI
- NVIDIA GPU auto-detection beyond documented hooks
- A Coda binary package repository
- Shipping an AUR helper
- Enabling the XLibre third-party repo
- Plymouth
- Calamares
- A default firewall
- NetworkManager
- Docker / Distrobox / Firejail as the required app runtime
- Claiming a read-only A/B core is already shipping

## Next steps

See [architecture.md](architecture.md) for the system picture and [docs/TODO.md](docs/TODO.md) for the ISO build, archinstall profile, AGS shell, and sandbox / A/B backlog.
