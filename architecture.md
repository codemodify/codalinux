# CodaLinux architecture

Canonical picture of **how the entire system looks**: partitions → layers → `/` folders → sandboxes → updates.

- **Target** is the locked product intent. Do not treat it as shipped until the installer and A/B slots exist.
- **Current tree** is what this repository actually builds today.
- Locked *choices* (Hyprland, no NetworkManager, official repos only, …) stay in [DESIGN.md](DESIGN.md). Commands: [docs/sandbox.md](docs/sandbox.md).

Do **not** claim A/B partitions, a read-only core, or a core-only ISO work until those are built.

## Target vs current (read this first)

| Piece | Target | Current tree |
| --- | --- | --- |
| Disk | ESP + OS-A + OS-B + data | Single mutable ext4 `/` (+ ESP on install). **A/B not implemented.** |
| Core OS | ~1.1 GiB-class bootable Arch only, RO while running | Live/install image is a **full desktop root** (core + Hyprland + AGS on one `/`) |
| Desktop | Hyprland + AGS + branding as a **session** on top of core | Same: Hyprland + vendored AGS on the live ISO. Do not rip it out. |
| Sandboxes | User-owned disposable Arch roots (`pacman --root` + upstream `bwrap`) | **Implemented:** `coda-sandbox` + `bubblewrap` on live and install lists |
| Host `pacman` | Core / OS only; gated; writes the **inactive** slot | Ordinary rolling pacman on the mutable root (not gated) |
| Installer | Pick **disk only**; locale/timezone/keymap fixed (Bozeman) | `coda-install` preseeds Bozeman defaults; custom profile still incomplete |
| Updates | OS slot swap; apps via sandbox `pacman` | ISO rebuild cadence; sandbox helper is in-tree |

Shipped and tryable now: **Hyprland + AGS desktop**, **`bubblewrap`**, **`coda-sandbox`** under `~/.coda/sandbox/<name>/`. Not shipped: A/B RO slots, core-only image, installer partition layout.

## Partitions (target)

UEFI-only. systemd-boot. Default filesystems: FAT32 ESP, ext4 OS slots and data.

| Partition | Size (intent) | FS | Mount / role |
| --- | --- | --- | --- |
| ESP | ~1–2 GiB | FAT32 | `/boot` — systemd-boot, A/B boot entries, UKI/kernel/initramfs |
| OS-A | ~4–8 GiB | ext4 | Bootable **Arch core only** (`base` + `linux` + firmware + mkinitcpio + microcode + systemd). **Read-only** when this slot is running |
| OS-B | ~4–8 GiB | ext4 | Inactive twin of OS-A. Gated OS updates write here, then flip the boot entry |
| data | remainder | ext4 | `/home`, `/var`, other mutable state. Survives slot swaps |

**Install:** operator picks the **disk** only. Locale `en_US.UTF-8`, timezone `America/Denver` (Bozeman), keymap `us` — **no region prompts**.

**Current tree:** recommended layout is still “ESP + one ext4 `/`”. The installer does not create OS-A/OS-B/data. Do not document those partitions as working.

## Layers

```
┌─────────────────────────────────────────────────────────┐
│  4. User data     /home  (includes ~/.coda/sandbox)     │
├─────────────────────────────────────────────────────────┤
│  3. Sandboxes     disposable Arch roots + bwrap         │
│                   many packages per named root          │
├─────────────────────────────────────────────────────────┤
│  2. Desktop       Hyprland + AGS + branding (session)   │
├─────────────────────────────────────────────────────────┤
│  1. Core OS       bootable Arch (~1.1 GiB class)        │
│                   NOT the full Hyprland/AGS image       │
└─────────────────────────────────────────────────────────┘
```

| # | Layer | What it is | How it is updated |
| --- | --- | --- | --- |
| 1 | **Core OS** | Bootable Arch: `base` + `linux` + firmware + mkinitcpio + microcode + systemd + boot. **Not** a full Hyprland/AGS root. | Host `pacman` into the **inactive** slot (gated). Reboot into the new slot. |
| 2 | **Desktop** | Hyprland + vendored AGS/Astal + greetd + branding + official settings apps. A **session**, not “the OS”. | Target: ship with or beside core (slot or later split). Current: on the same live/install image. |
| 3 | **Sandboxes** | Disposable Arch filesystem trees. Isolation is **upstream bubblewrap** only. | `coda-sandbox install` / `pacman` (repeatable into the same name). Destroy and recreate. |
| 4 | **User data** | `/home` (and later other data mounts). | Ordinary files. Sandbox trees live here so they survive OS slot swaps. |

Do not collapse this back into “`pacman -S postgres` on the host.”

## `/` mapping

### Target (after A/B + data split)

```
Core (RO slot while running)
  /usr              OS userland
  /boot             ESP: systemd-boot + A/B entries + kernel
  /etc              Base OS config (or a small writable overlay)
  /bin /sbin /lib   usr-merge symlinks into /usr (stock Arch)

Data (writable, survives slot swaps)
  /home             Users; ~/.coda/sandbox lives here
  /var              Host logs and host caches (not sandbox trees)

Runtime (not persisted as “the OS”)
  /dev /proc /sys   kernel interfaces
  /run /tmp         ephemeral

Apps
  Prefer coda-sandbox roots, not host /opt or host /usr pollution
```

### Current tree

One writable `/` holds core + desktop + `/home` + `/var`. usr-merge is whatever stock Arch ships. Sandbox trees still go under `~/.coda/sandbox/` on that same `/`.

## Sandboxes (implemented)

Path spelling is locked: **`~/.coda/sandbox/<name>`** (singular `sandbox`).

```
~/.coda/sandbox/<name>/         sandbox directory
~/.coda/sandbox/<name>/root     pacman --root tree
~/.coda/sandbox/<name>/meta
~/.coda/cache/pacman            shared cache for this user
```

- **User-owned. No sudo** for create / install / pacman / enter / run / destroy.
- `pacman --root` runs in `unshare --user --map-root-user` (optional `--keep-caps`) so extract sees uid 0 and files on disk stay owned by the real user.
- `enter` / `run` are unprivileged **upstream `bwrap`**. CodaLinux does not reimplement bubblewrap.
- **One name = one Arch root = many packages/apps.** Example: `dev` with `postgresql`, `redis`, `git` via repeated `coda-sandbox install dev …`. Not one sandbox per app.
- Host `pacman` = **core / OS only** (rare; gated later). Extra software goes in a sandbox.
- Shared cache is a deliberate choice: one download of `base`, many installs. Override with `--store` / `CODA_SANDBOX_STORE` or `--cache` / `CODA_SANDBOX_CACHE`.
- Not `/var/coda`, not `/var/lib/coda`, not XDG `~/.local/share/…` as the primary path.

`~/.coda` is created on first use (`0700`). No system tmpfiles under `/var` for the store.

```bash
coda-sandbox create dev
coda-sandbox install dev postgresql
coda-sandbox install dev redis git
coda-sandbox enter dev
coda-sandbox destroy dev --force
```

Needs Arch/CodaLinux, official `core`/`extra`, a working keyring, network for the first `create`, `bubblewrap`, and unprivileged user namespaces. Details: [docs/sandbox.md](docs/sandbox.md).

## Update model (target)

| What | How |
| --- | --- |
| **OS / core** | Write the **inactive** slot (OS-A or OS-B). Keep the running slot read-only. Flip systemd-boot to the new slot. Roll back by flipping back. |
| **Desktop session** | Travels with the core slot until a later split. Not a third A/B pair in v1. |
| **Apps / extras** | Sandbox `pacman` only. Throw the root away if it is broken. Persist *data* under `$HOME` or `--bind`. |
| **User files** | Stay on **data** (`/home`). Independent of slot swaps. |

**Current tree:** no slot writer, no RO remount, no boot-entry flip. Delivery is still periodic live ISO rebuilds. Sandbox create/install/destroy is the only implemented half of this table.

## How to try what exists

Do not rebuild the ISO just to read this document. The desktop image already includes `bubblewrap` (`packages/sandbox.txt`) and `/usr/local/bin/coda-sandbox`.

1. **Desktop UX** (Hyprland / AGS): `./scripts/qemu-desktop-dev.sh` — 9p share; `coda-sync-desktop-from-host.sh` copies wrappers including `coda-sandbox`.
2. **Sandbox helper** (on a live/Arch session with network): the commands in [Sandboxes](#sandboxes-implemented). First `create` bootstraps `base` and needs `pacman` on the host. This is **not** an A/B disk test.
3. **ISO smoke**: `./scripts/qemu-boot-test.sh` after `./scripts/build-iso.sh`. Confirms the mutable desktop live image, not OS-A/OS-B.

A/B partition layouts, RO core mounts, and gated host pacman are **future work** ([docs/TODO.md](docs/TODO.md) §5).
