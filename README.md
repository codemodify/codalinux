# CodaLinux

CodaLinux is an [Arch Linux](https://archlinux.org/)-based rolling desktop distribution. It ships a custom [Hyprland](https://hypr.land/) compositor session and an [AGS](https://aylur.github.io/ags/)/[Astal](https://aylur.github.io/astal/) unified shell. The installed system tracks official Arch repositories; this tree holds branding, session layout, the live ISO profile, and a Coda-owned offline installer (`coda-install`).

This repository is a **v1 scaffold**. It captures locked architecture decisions and a conventional directory layout so the ISO, installer profile, and desktop shell can be implemented without re-litigating the stack.

## Locked defaults

| Area | v1 choice |
| --- | --- |
| Base | Arch Linux, UEFI-only, rolling |
| Bootloader | systemd-boot |
| Kernel / initramfs | stock `linux` + mkinitcpio + CPU microcode |
| Init | systemd |
| Root filesystem | ext4 |
| Packages | Official Arch repos only (no Coda repo, no default AUR helper). Host `pacman` = core OS; extra software = `coda-sandbox` + `bubblewrap` |
| Network | systemd-networkd + iwd (not NetworkManager; no firewall by default) |
| Display | Wayland + XWayland default; XLibre (X11) session path supported later |
| Display manager | greetd (placeholder greeter) |
| Desktop | Hyprland + AGS/Astal (vendored into `/usr/local`; no Waybar interim) |
| Delivery | archiso live ISO + offline `coda-install` (not Calamares) |
| Support | GitHub issues |

The system picture (partitions → layers → `/` → sandboxes → **system-config** → updates) is **[architecture.md](architecture.md)** — **Target** vs **Current tree**. Locked choices live in **[DESIGN.md](DESIGN.md)** (decision log). Backlog: **[docs/TODO.md](docs/TODO.md)**. Sandbox commands: **[docs/sandbox.md](docs/sandbox.md)**. `system-config` implementation: [`core/system-config/`](core/system-config/README.md) (ISO-wired).

## Repository layout

```
architecture.md  System picture (partitions, layers, `/`, sandboxes, system-config, updates)
DESIGN.md        Locked decision log (do not contradict)
packages/        Source-of-truth package lists (official Arch names only)
archiso/         archiso profile stubs (UEFI + systemd-boot live ISO)
install/         archinstall config + custom profile stubs
desktop/         Hyprland, hypr* companions, AGS/Astal app layout
branding/        os-release, wallpaper/theme placeholders
sessions/        Wayland Hyprland session + XLibre path notes
scripts/         ISO compose/build helpers, coda-sandbox, NVIDIA hook placeholder
core/            Go core components (`system-config` daemons + clients)
docs/            Implementation TODOs + sandbox command reference
```

Layout assumptions (airootfs overlay timing, how lists are composed, why AGS is in-tree) are recorded in [DESIGN.md](DESIGN.md#repository-layout-assumptions).

## Quick start

### Read the decisions

1. Skim [architecture.md](architecture.md) for the system picture, then [DESIGN.md](DESIGN.md) before changing the stack.
2. Edit package sets under [`packages/`](packages/README.md). Do not add AUR packages, unofficial repos, or a Coda `pacman` repo.
3. Keep ISO and installer consumers in sync with `scripts/compose-package-lists.sh`.

### Validate lists

```bash
./scripts/check-package-lists.sh
```

The checker rejects known policy violations (AUR helpers, Calamares, NetworkManager, firewalls, AGS/XLibre binary names that are not in official repos).

### Build a live ISO (not production-ready)

```bash
./scripts/build-iso.sh
```

On Arch this runs `mkarchiso` as the current user when possible (archiso 89+ can unshare). Otherwise it uses a privileged `archlinux` container **only if Docker/Podman already works without sudo**. The script never calls `sudo` and never answers pacman provider prompts. Output: `out/codalinux-<date>-x86_64.iso`.

Iteration phases:

- **Desktop UX trial/error** (hypr / AGS / wallpaper): `./scripts/qemu-desktop-dev.sh` — GTK QEMU + 9p share of the host tree; no ISO rebuild.
- **ISO smoke**: `./scripts/qemu-boot-test.sh` (QEMU/KVM + OVMF, serial + QMP screenshots).
- **Package, vendor, or squashfs changes**: `./scripts/build-iso.sh`.

VirtualBox remains valid for manual VMSVGA checks. It is not the automated path.

### Unattended local builds (abox)

`./scripts/build-iso.sh` must stay noninteractive: no sudo password prompts, no pacman provider menus.

- Prefer native `mkarchiso` on Arch.
- If you use Docker, add the builder account to the `docker` group so `docker info` works without sudo. Re-login after `usermod -aG docker "$USER"`.
- Optional host sudoers may allow **only** `/usr/bin/mkarchiso` and `/usr/bin/docker`. Do **not** grant passwordless root (`ALL=(ALL) NOPASSWD: ALL`).
- Provider packages are pinned in `packages/` (`iptables`, `pipewire-jack`, `tesseract-data-eng`) so pacstrap does not ask.

The live ISO autologins user `live` into Hyprland on tty1 (`coda-hyprland` → `start-hyprland`, VirtualBox software-render path) with the vendored **AGS** shell (top bar with workspaces and a running-app taskbar, launcher via `Super+Space`, notifications, control center via `Super+,`) and official settings apps for Wi-Fi (iwd / impala), Bluetooth, audio, webcam, and appearance. Locale/timezone/keymap are Bozeman, Montana defaults (`en_US.UTF-8`, `America/Denver`, `us`) and are not asked at install time.

A GitHub Actions workflow (`Build live ISO`) uploads `codalinux-live-iso` as an artifact when it succeeds.

### Install (offline, one disk)

From the CodaLinux live ISO (no network required at install time):

```bash
coda-install
```

That helper keeps Bozeman locale/timezone/keymap and only asks for the disk. `CODA_INSTALL_DISK=/dev/vda coda-install` (or `auto` for the first disk) partitions ESP + OS-A + OS-B + data, splits the live airootfs into **core on OS-A** and **desktop on `coda-data`**, and enables greetd autologin for **`user` / `1`**. Automated QEMU loop (local ISO only): `./scripts/qemu-install-e2e.sh`. See [`install/`](install/README.md).

### Extra software (sandboxes)

Do not `pacman -S` experimental apps onto the host. Create a throwaway Arch root and run it with upstream `bwrap`:

```bash
coda-sandbox create <env>
coda-sandbox install <env> postgresql
coda-sandbox install <env> redis git
coda-sandbox shell <env>
coda-sandbox exec <env> postgres --version
coda-sandbox destroy <env>
```

Trees live under `~/.coda/sandbox/<env>/` (user-owned; no sudo). One name is one Arch root that can hold many packages. Shared cache: `~/.coda/cache/pacman`. Live ISO overlay is **`cow_spacesize=4G`** so `create` is not capped at the stock 256M COW (see [docs/sandbox.md](docs/sandbox.md#live-iso-space)).

See [architecture.md](architecture.md) and [docs/sandbox.md](docs/sandbox.md). ESP+A+B+data install and `coda-slot` updates are implemented (core on A/B, desktop on data). The running slot is still writable (RO remount is later). The live ISO includes the Hyprland + AGS desktop (`bubblewrap` + `coda-sandbox` are on that image).

## What this repo does not contain

- A second package repository or binary package pipeline
- A default AUR helper
- Working NVIDIA auto-detection (hook + package list only)
- Plymouth (explicitly deferred)
- Calamares
- A shipping read-only running slot or a core-only live ISO (slots are core-only after install; the ISO is still the full desktop)
- Docker / Distrobox as a required app runtime

## Support

Use [GitHub issues](https://github.com/codemodify/codalinux/issues) for bugs, hardware reports, and design questions. There is no separate forum or tracker in v1.
