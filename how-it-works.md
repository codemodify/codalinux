# How CodaLinux works

CodaLinux is an Arch-based desktop with four layers that stay separate on purpose: a small bootable **core** on A/B slots, the **desktop** (Hyprland + AGS) on a data partition, **sandboxes** for day-to-day packages, and your **home** on that same data partition so it survives OS swaps.

Deeper partition / boot / update detail: [architecture.md](architecture.md). Sandbox command reference: [docs/sandbox.md](docs/sandbox.md).

## Big picture

After install the disk looks like this:

- `coda-esp` — systemd-boot + kernels for slot A and B
- `coda-a` / `coda-b` — twin copies of the **bootable Arch core only** (kernel, systemd, firmware, base tools — not Hyprland)
- `coda-data` — `/home`, `/var`, and `/coda/data/desktop` (Hyprland, AGS, Settings, apps)

At boot, UEFI starts systemd-boot, which boots the **active** core slot. That core mounts `coda-data`, then `coda-desktop-mount` merges the desktop onto `/` so you get a normal Hyprland session. Your files and sandboxes live on data, so flipping A↔B does not wipe them.

The **live ISO** is still one full desktop root (handy to try and to install from). The **split** only happens when you install or run `coda-slot`.

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

OS updates are **A/B slot swaps**, not “pacman -Syu on the running root and hope.” Today the practical path is from a **newer live ISO** (or the same machine with that ISO’s airootfs available):

```bash
# From live ISO, pointing at the installed disk:
coda-slot status --disk /dev/vda
coda-slot install --disk /dev/vda          # write core into inactive slot; refresh desktop on data
coda-slot boot-test --disk /dev/vda        # oneshot boot into the new slot (old default kept)
# If that boot looks good:
coda-slot promote --disk /dev/vda          # make the new slot the default
```

What that does:

- **`install`** fills the **inactive** slot with a fresh core and refreshes `/coda/data/desktop`. It does **not** mutate the slot you are running.
- **`boot-test`** uses `bootctl set-oneshot`, so a bad boot falls back to the previous default.
- **`promote`** sets the new slot as the lasting default.

Desktop bits on data are refreshed with the slot write; `/home` and sandbox trees stay. The longer-term target is gated host `pacman` that only writes the inactive slot; that gating is **not** shipped yet, so prefer `coda-slot` for real OS updates rather than casually upgrading the live core in place.

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

Install once with `coda-install`, flip OS versions with `coda-slot`, put apps in `coda-sandbox`, keep your life on `coda-data`. Host `pacman` on the running core is still possible today because slots are writable, but the design intent is core via slots and everything else via sandboxes.
