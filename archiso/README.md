# archiso profile

This directory is a **CodaLinux archiso profile**, not a vendor of the `archiso` tool.

```bash
./scripts/build-iso.sh
```

On Arch that runs `mkarchiso` directly. Elsewhere it uses a privileged `archlinux` container (Docker or Podman) if one is available.

`scripts/build-iso.sh` composes `packages.x86_64`, copies branding/desktop/session/installer overlays into `airootfs/`, then runs `mkarchiso`.

## Contents

| Path | Role |
| --- | --- |
| `profiledef.sh` | ISO metadata; UEFI + systemd-boot only |
| `packages.x86_64` | Generated from `packages/*.txt` — do not hand-edit |
| `pacman.conf` | Official `core` + `extra` only |
| `efiboot/` | systemd-boot loader stubs (`timeout 1`) |
| `airootfs/` | Root overlay: hostname, networkd, iwd, greetd, service wants |

## Assumptions

- Overlay files are copied **before** pacman installs packages. `os-release` is re-applied by the hook in `branding/hooks/`.
- No BIOS/syslinux tree on purpose (UEFI-only).
- No `syslinux/` or `grub/` directories.
- `mkinitcpio` and `mkinitcpio-archiso` come from `packages/live.txt`.
- A complete releng-equivalent image (initramfs hooks, mirror chooser, speech entry) is still TODO — see [docs/TODO.md](../docs/TODO.md).

Copy a current [releng](https://gitlab.archlinux.org/archlinux/archiso/-/tree/master/configs/releng) profile beside this stub when implementing the real build, then re-apply CodaLinux branding and package lists.
