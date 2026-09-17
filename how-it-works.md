# How CodaLinux works

CodaLinux is an Arch-based desktop with four layers that stay separate on purpose: a small bootable **core** on A/B slots, the **desktop** (Hyprland + AGS) on a data partition, **sandboxes** for day-to-day packages, and your **home** on that same data partition so it survives OS swaps.

Deeper partition / boot / update detail: [architecture.md](architecture.md). Sandbox command reference: [docs/sandbox.md](docs/sandbox.md).

## Big picture

After install the disk looks like this:

- `coda-esp` — systemd-boot + kernels for slot A and B
- `coda-a` / `coda-b` — twin copies of the **bootable Arch core only** (kernel, systemd, firmware, base tools — not Hyprland)
- `coda-data` — `/home`, `/var`, and `/coda/data/desktop` (Hyprland, AGS, Settings, apps)

At boot, UEFI starts systemd-boot, which boots the **active** core slot. That core mounts `coda-data`, then `coda-desktop-mount` merges the desktop onto `/` so you get a normal Hyprland session. Your files and sandboxes live on data, so flipping A↔B does not wipe them.

The **live ISO** is still one full desktop root (handy to try and to install from). The **split** only happens at first install (`coda-install`). Later OS updates use `coda-update`.

---

## How to install the OS

1. Boot the live ISO (UEFI). You land in the full Hyprland desktop as `user` / `1`.
2. Run the installer (it only asks for a disk; locale/timezone/keymap stay Bozeman defaults):

```bash
coda-install --disk /dev/nvme0n1
# or for QEMU / unattended:
CODA_INSTALL_DISK=auto coda-install
```

That **wipes the disk** and lays out ESP + A + B + data. Offline (no network): it splits the live airootfs so **core goes into OS-A** and **desktop goes onto `coda-data`**. OS-B is prepared as the inactive twin. Boot default is A.
3. Reboot from disk. greetd autologins as `user` / `1` into Hyprland with the desktop merged from data.

---

## How to update the OS (core + desktop session)

OS updates are **A/B slot swaps**, not `sudo pacman -Syu` on the running root.

```bash
coda-update status
coda-update core                 # Arch repos → inactive slot; oneshot next reboot
# reboot; if the new slot looks good:
coda-update core --promote       # only after you are already running on that slot

coda-update desktop              # Arch repos → /coda/data/desktop; /home stays
```

What that does:

- **`coda-update core`** refuses to write the **running** slot (and, from the live ISO, the current boot-default slot — that is the fallback if oneshot fails). It pulls **core** packages from official Arch repos into the inactive slot (`pacman --root` on that slot, not on `/`), refreshes that slot’s kernel/ESP entry, and sets a systemd-boot **oneshot**. The boot **default** stays put. A failed boot consumes the oneshot and keeps the previous default. A failed write umounts and leaves the old slot bootable.
- **`coda-update core --promote`** swaps the default only when this boot **is** the new slot (after a successful oneshot). Promote is **not** immediate on the first command.
- **`coda-update desktop`** updates Hyprland/AGS/session packages on `coda-data`. It does not write core slots (except mounting data if needed). `/home` is left alone. Vendored `/usr/local` AGS/hyprbars are kept (not deleted).

Offline / no-network hook (live ISO, current QEMU e2e — **no NIC**):

```bash
coda-update core --from-iso --disk /dev/vda    # old ISO-split path; not the product default
```

`coda-slot` is still there as a low-level helper (`install` / `boot-test` / `promote`) for e2e and recovery. Prefer `coda-update`. Host `pacman -Syu` on the running root is **not** the OS update path.

---

## How to install software and use it

Day-to-day packages do **not** go on the host core. They go in a **named sandbox**: a disposable Arch root under `~/.coda/sandbox/<env>`, run with upstream **bubblewrap**. No sudo, no Docker.

```bash
coda-sandbox create dev
coda-sandbox install dev postgresql redis git
coda-sandbox shell dev                 # login shell inside the sandbox
coda-sandbox exec dev postgres --version
```

One env is one Arch root and can hold many packages. Create another name only when you want a separate throwaway tree. Host Settings (`system-config-gui`) and the Hyprland session stay on the desktop layer; sandboxes are for apps and toolchains.

---

## How to update software and use it again

Same sandbox, install again (pacman into that env):

```bash
coda-sandbox install dev postgresql    # upgrades/refreshes packages in that env
coda-sandbox exec dev postgres --version
```

Or throw the env away and rebuild:

```bash
coda-sandbox destroy experiment
coda-sandbox create experiment
coda-sandbox install experiment …
```

Shared pacman cache lives under `~/.coda/cache/pacman`, so downloads are reused across envs. Because sandboxes sit under `$HOME` on `coda-data`, they survive core A/B updates.

---

## Short mental model

Install once with `coda-install`, update the OS with `coda-update`, put apps in `coda-sandbox`, keep your life on `coda-data`. Host `pacman` on the running core is still possible today because slots are writable, but that is **not** the product update path.
