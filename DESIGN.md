# CodaLinux v1 design decisions

This document is the source of truth for locked v1 architecture. Do not contradict it in ISO profiles, installer configs, package lists, or desktop stubs. If a later decision changes the stack, update this file in the same change.

CodaLinux is a rolling Arch Linux derivative: it does not fork the base system. Periodic live ISO rebuilds are the delivery cadence; day-to-day updates come from official Arch repositories via `pacman`.

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
| Multilib | **Off by default** — enable only if a later decision requires 32-bit NVIDIA userspace |
| Security | Stock Arch defaults (pacman signature policy, no extra MAC/firewall product) |

Package names that are **not** in official repositories must not appear in default lists. That includes AGS/Astal binaries and XLibre. See [Out-of-repo components](#out-of-repo-components).

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
| Shell | AGS/Astal unified shell (in-tree under `desktop/ags/`) |
| Companions | hyprlock, hypridle, hyprpaper |
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
2. Install official-repo **build/runtime GTK dependencies** from [`packages/ags-build-deps.txt`](packages/ags-build-deps.txt) on systems that will compile the shell.
3. Decide later (not in this scaffold) whether the ISO vendors a prebuilt tree under `/usr/local` or documents a post-install source build.
4. Do not add `yay -S aylurs-gtk-shell` to any default path.

### XLibre session path

XLibre is not in official Arch repos (AUR and a third-party `[xlibre-stable]` repo exist upstream). CodaLinux **must not** enable that third-party repo in `pacman.conf`.

v1 approach:

1. Default session is Hyprland + XWayland. X11 applications run there.
2. [`sessions/xlibre/`](sessions/xlibre/README.md) holds the future `.desktop` stub and packaging notes.
3. Shipping XLibre later requires an explicit packaging decision (source build, optional user-enabled upstream repo, or waiting for official packages). That decision is **not** made here.

## Repository layout assumptions

These are scaffolding choices, not product-stack changes. Prefer this conventional Arch-derivative layout unless a later change replaces it.

### Top-level map

| Path | Role |
| --- | --- |
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
- `packages/nvidia.txt` and `packages/optional-cups.txt` are **not** in the default compose.
- `packages/ags-build-deps.txt` is **not** in the live ISO default set (keeps the image smaller until the shell is built).
- `archiso/packages.x86_64` also includes archiso-mandatory packages (`mkinitcpio`, `mkinitcpio-archiso`) from `packages/live.txt`.

### archiso overlay timing

archiso copies `airootfs/` **before** installing packages. Files owned by packages (notably `/usr/lib/os-release` from `filesystem`) will be overwritten.

Assumption: branding is re-applied with a pacman hook installed from [`branding/hooks/`](branding/hooks/codalinux-os-release.hook) into the airootfs. The hook writes `branding/os-release` over `/usr/lib/os-release` after `filesystem` is installed. `/etc/os-release` remains the usual symlink.

### Service enablement

Enabled on live and (via the installer profile) on the installed system:

- `greetd.service` (as the display manager)
- `systemd-networkd.service`
- `systemd-resolved.service`
- `iwd.service`
- `bluetooth.service`

Not enabled: NetworkManager, CUPS, firewalld, Plymouth.

Live ISO service symlinks live under `archiso/airootfs/etc/systemd/system/`. Installed-system enablement is an archinstall profile responsibility.

### Hostname, locale, users

| Item | v1 scaffold default | Overridable |
| --- | --- | --- |
| Hostname | `coda` | Installer |
| Locale | `en_US.UTF-8` | Installer |
| Console keymap | `us` | Installer |
| Timezone | `UTC` until the installer sets one | Installer |
| Live session | stock archiso root console + greetd placeholder | Later live-UX work |
| Installed users | Created by archinstall credentials file | Yes |

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
| `desktop/hypr/*.conf` | `/etc/skel/.config/hypr/` (and `/etc/xdg/hypr/` later if we ship system defaults) |
| `branding/os-release` | `/usr/lib/os-release` via hook |
| `branding/wallpapers/` | `/usr/share/backgrounds/codalinux/` (when assets exist) |

`build-iso.sh` is responsible for copying those trees into `airootfs/` at build time so we do not maintain duplicates.

## Explicit non-goals (v1 scaffold)

- Implementing the AGS/Astal UI
- A production-quality ISO in CI
- NVIDIA GPU auto-detection beyond documented hooks
- A Coda binary package repository
- Shipping an AUR helper
- Enabling the XLibre third-party repo
- Plymouth
- Calamares
- A default firewall
- NetworkManager

## Next steps

See [docs/TODO.md](docs/TODO.md) for the ISO build, archinstall profile, and AGS shell backlog.
