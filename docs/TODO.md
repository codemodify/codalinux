# Implementation TODOs

Scaffolding only. Each item is work for a later change. Do not treat stubs as finished.

## 1. Live ISO (`archiso/` + `scripts/build-iso.sh`)

- [x] Profile includes mkinitcpio-archiso hooks, pacman-init, UEFI systemd-boot entries, greetd/iwd/networkd enables, os-release overlay, `coda-install`.
- [x] `scripts/build-iso.sh` prefers native/rootless `mkarchiso`; Docker/Podman only if already usable without sudo (no sudo fallbacks).
- [x] First `mkarchiso` completed (ISO 9660, label `CODA_202609`). Boot on OVMF/hardware still unverified.
- [ ] Run `mkarchiso` and boot the image on UEFI hardware or firmware (OVMF).
- [ ] Confirm `bootmodes=('uefi.systemd-boot')` matches the build host's archiso.
- [ ] Verify the os-release pacman hook wins over the `filesystem` package.
- [x] Live autologin: user `live` via `coda-hyprland` (VM software-render path + `/var/log/coda-hyprland.log`). Verify on VirtualBox EFI/VMSVGA.
- [x] Vendored AGS/Astal shell on the live ISO (bar, launcher, notifications, control center) plus official settings apps (impala / blueman / pavucontrol / snapshot / nwg-look). Waybar interim rejected.
- [ ] Confirm greetd + Hyprland + PipeWire + AGS actually start on the live image (tty2 is the rescue console).
- [ ] Accessibility / speech boot entry (optional; not in v1).
- [ ] Keep the ISO official-repos-only; no `[codalinux]` repo, no AUR helper.
- [ ] Periodic rebuild pipeline (manual first; CI only if an Arch builder exists).

## 2. archinstall profile (`install/`)

- [ ] Turn `install/profiles/codalinux.py` into a real archinstall profile (or `--script`) that:
  - installs systemd-boot (UEFI);
  - formats `/` as ext4 by default;
  - installs the composed official package set;
  - enables greetd (not SDDM/GDM/LightDM);
  - enables systemd-networkd + iwd + systemd-resolved;
  - does **not** pull NetworkManager;
  - writes CodaLinux `os-release` and session files;
  - installs Hyprland configs into the new user's `~/.config`.
- [ ] Keep `additional-repositories` empty.
- [ ] Generate or validate `user_configuration.json` against the installed archinstall version (`archinstall --dry-run`).
- [ ] Disk layout remains operator-supplied; document a recommended ESP + ext4 `/` layout only.
- [ ] NVIDIA: call `scripts/hooks/nvidia.sh` from the profile when detection is implemented — not before.
- [ ] Credentials stay out of git (`user_credentials.json` is local-only).

## 3. AGS / Astal shell (`desktop/ags/`)

- [x] Pin AGS 3.1.2 + Astal commits in `desktop/ags/README.md` and `scripts/vendor-ags.sh`.
- [x] Implement the unified shell (bar, notifications, launcher, control center).
- [x] AGS taskbar of running apps, workspace add/remove, and per-workspace tile ↔ overlapping-float stack via `coda-hypr-ws` (hyprfloat semantics; not tabbed groups).
- [x] Floating titlebars via vendored hyprbars (close / max / min); tiled windows stay border-only.
- [x] Vendor from source at ISO build time with official-repo deps in `packages/ags-build-deps.txt`. No AUR helper, no Coda repo.
- [x] Start AGS from `desktop/hypr/hyprland.lua` via `coda-ags` (Waybar is not a fallback).
- [ ] Confirm the vendored shell on a real live boot (VirtualBox / OVMF).

## 4. Nearby follow-ups (not blockers for the three tracks above)

- [ ] Branded greetd greeter (theme hooks are placeholders).
- [x] Default hyprpaper / hyprlock still at `branding/wallpapers/default.png` (branded gradient). Full theme assets still later.
- [ ] XLibre session packaging decision (see [DESIGN.md](../DESIGN.md#xlibre-session-path)); still no Coda repo.
- [ ] NVIDIA detect/install hook implementation.
- [ ] Optional CUPS profile using `packages/optional-cups.txt`.
- [ ] Plymouth — still deferred.
