# CodaLinux installer

First install is a **Coda-owned** path (`coda-install` → `coda-install-ab.sh`), not Calamares and not archinstall/pacstrap.

Install time is **offline**: the live ISO already contains the system. Nothing is downloaded. ISO **build** may still fetch official Arch packages.

## Layout

Operator picks **one disk**. Locale/timezone/keymap stay Bozeman (`en_US.UTF-8`, `America/Denver`, `us`).

| Partition | PARTLABEL | FS | Size (v1) | Mount |
| --- | --- | --- | --- | --- |
| ESP | `coda-esp` | FAT32 | 1 GiB | `/boot` |
| OS-A | `coda-a` | ext4 | 8 GiB floor | `/` on first install |
| OS-B | `coda-b` | ext4 | 8 GiB floor | inactive (empty until `coda-slot install`) |
| data | `coda-data` | ext4 | remainder (≥ 4 GiB) | `/coda/data` + bind `/home` + `/var` |

v1 slots hold the **full live desktop** (not a 4 GiB core-only image). **Minimum disk ~22 GiB** (1+8+8+4 + GPT slack). Recommend **32 GiB** for QEMU. The planner refuses smaller disks with a clear error.

`~/.coda/sandbox` lives on data (`/home`) so it survives slot swaps.

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

`coda-install-post.sh` still runs on the installed root (greetd → `user`, networkd+iwd, QGA+sshd, live `/usr/local` desktop bits).

## Updates (inactive slot)

From the **live ISO** (offline payload) with the installed disk attached:

```bash
coda-slot --disk /dev/vda status
coda-slot --disk /dev/vda install          # write airootfs into inactive (B if A is filled)
coda-slot --disk /dev/vda boot-test        # systemd-boot oneshot; default unchanged
# reboot into the oneshot slot; if it fails, next boot is still the old default
coda-slot promote --slot b                 # only after a successful boot-test
```

From a running installed system, `--disk` is optional. `install` refuses to write the running slot. Kernels live at `/boot/coda/a/` and `/boot/coda/b/` so A and B do not share one `vmlinuz-linux`.

## Automated e2e (abox)

One host command, no guest TTY, local ISO only:

```bash
./scripts/qemu-install-e2e.sh
# or: ./scripts/qemu-install-e2e.sh --phase install
```

Uses `out/codalinux-*.iso` (or `--build`). QEMU has **no NIC**. Guest disk is the first virtio disk (`/dev/vda`). Steps 1–8: pick disk → layout → offline install A → reboot Hyprland `user`/`1` → write B → oneshot-boot B → promote → reboot B.

Success signal: `/etc/coda/slot` matches, `/home` and `/var` bind `coda-data`, greetd autologin `user`, Hyprland instance under `/run/user/<uid>/hypr`.

Physical laptop promote is **manual after QEMU is green**. Do not run this e2e as a system-config host test.

## Leftover archinstall files

`user_configuration.json`, `packages.txt`, and `profiles/codalinux.py` remain for reference. They are **not** the first-install driver. Keep `"additional-repositories": []`.
