# CodaLinux installer

First install is a **Coda-owned** path (`coda-install` → `coda-install-ab.sh`), not Calamares and not archinstall/pacstrap.

Install time is **offline**: the live ISO already contains the system. Nothing is downloaded. ISO **build** may still fetch official Arch packages.

## Target vs Current

| | Target (required) | Current |
| --- | --- | --- |
| OS-A / OS-B | **Arch core only** (base + linux + firmware + mkinitcpio + microcode + systemd + boot) | Install **split**s the live airootfs: core → slot, desktop → `coda-data`. Later updates: `coda-update core` (Arch repos → inactive slot). |
| Desktop | Hyprland + AGS + greetd chrome + the rest of the session on **`coda-data`** | `/coda/data/desktop`, merged at boot by `coda-desktop-mount.service`. Hyprland success still means a running session; the binary must come from **data**, not from the slot root payload. |
| Live ISO | (unchanged) full desktop image for the live session | Still a full desktop airootfs. Do not rip Hyprland/AGS out of the ISO. |
| ESP | ~1 GiB | 1 GiB (`ESP_MIB = 1024`) |
| Slot floor | ~4 GiB once core-only | **4 GiB** (`SLOT_FLOOR_MIB`). The old 8 GiB floor was only because v1 copied the full live desktop into each slot. |
| Data | remainder; holds `/home`, `/var`, and desktop | **8 GiB floor** so the desktop tree fits with home/var. Remainder of the disk. |

Picture, boot merge, and failure modes: [architecture.md — Desktop on coda-data](../architecture.md#desktop-on-coda-data-required).

## Layout

Operator picks **one disk**. Locale/timezone/keymap stay Bozeman (`en_US.UTF-8`, `America/Denver`, `us`).

| Partition | PARTLABEL | FS | Size (required) | Mount |
| --- | --- | --- | --- | --- |
| ESP | `coda-esp` | FAT32 | 1 GiB | `/boot` |
| OS-A | `coda-a` | ext4 | 4 GiB floor | `/` on first install (**core only**) |
| OS-B | `coda-b` | ext4 | 4 GiB floor | inactive (empty until `coda-update core`) |
| data | `coda-data` | ext4 | remainder (≥ 8 GiB) | `/coda/data` + bind `/home` + `/var` + **`/coda/data/desktop`** |

**Minimum disk ~17 GiB** (1+4+4+8 + GPT slack). Recommend **32 GiB** for QEMU. The planner refuses smaller disks with a clear error.

`~/.coda/sandbox` lives on data (`/home`) so it survives slot swaps. The desktop payload also lives on data so it survives slot swaps and is **not** duplicated into A and B.

## Commands

```bash
# Interactive: list disks, confirm wipe
coda-install

# Silent (QEMU / e2e): first disk or an explicit path
CODA_INSTALL_DISK=auto coda-install
CODA_INSTALL_DISK=/dev/vda coda-install
```

Default login after reboot:

```
user: user
password: 1
```

Root password is also `1`. Override with `CODA_INSTALL_USER` / `CODA_INSTALL_PASSWORD` / `CODA_INSTALL_ROOT_PASSWORD`.

`coda-install-post.sh` still runs after the split (greetd → `user` on the **desktop** tree, networkd+sshd+QGA on the **slot**, live `/usr/local` session bits onto `/coda/data/desktop`).

## Updates (inactive slot)

Product CLI:

```bash
coda-update status
coda-update core                 # Arch repos → inactive slot; oneshot next reboot
# reboot; if it looks good:
coda-update core --promote       # only while running on the new slot
coda-update desktop              # Arch repos → /coda/data/desktop; /home stays
```

Do not `sudo pacman -Syu` on the running root.

Offline / current QEMU e2e (no NIC):

```bash
coda-update core --from-iso --disk /dev/vda
```

`coda-slot` remains the low-level helper (`install` / `boot-test` / `promote`). From a running installed system, `--disk` is optional. `coda-update core` refuses to write the running slot. Kernels live at `/boot/coda/a/` and `/boot/coda/b/` so A and B do not share one `vmlinuz-linux`.

Desktop is **not** A/B’d: a failed B oneshot leaves a new desktop (if you also ran `coda-update desktop`) with the previous default core (see architecture.md failure modes).

## Automated e2e (abox)

One host command, no guest TTY, **local ISO only** (never download GitHub Actions ISO artifacts):

```bash
./scripts/qemu-install-e2e.sh
# or: ./scripts/qemu-install-e2e.sh --phase install
```

Uses `out/codalinux-*.iso` (or `--build`). QEMU has **no NIC**. Guest disk is the first virtio disk (`/dev/vda`). Steps 1–8: pick disk → layout → offline **core** install A + desktop on data → reboot Hyprland `user`/`1` → `coda-update core --from-iso` into B (oneshot) → reboot B → `coda-update core --promote` → reboot B. Arch-repo pull is later (`CODA_E2E_CORE_FROM=repos` + a NIC); not claimed green here.

Success signal: `/etc/coda/slot` matches, `/home` and `/var` bind `coda-data`, **`/coda/data/desktop` holds Hyprland**, `/usr` is merged from that tree, greetd autologin `user`, Hyprland instance under `/run/user/<uid>/hypr`.

Physical laptop promote is **manual after QEMU is green**. Do not run this e2e as a system-config host test.

## Leftover archinstall files

`user_configuration.json`, `packages.txt`, and `profiles/codalinux.py` remain for reference. They are **not** the first-install driver. Keep `"additional-repositories": []`.
