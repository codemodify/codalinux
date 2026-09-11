# Implementation TODOs

Scaffolding only. Each item is work for a later change. Do not treat stubs as finished.

## 1. Live ISO (`archiso/` + `scripts/build-iso.sh`)

- [x] Profile includes mkinitcpio-archiso hooks, pacman-init, UEFI systemd-boot entries, greetd/iwd/networkd enables, os-release overlay, `coda-install`.
- [x] `scripts/build-iso.sh` runs native `mkarchiso` or a privileged `archlinux` container.
- [ ] Run `mkarchiso` and boot the image on UEFI hardware or firmware (OVMF).
- [ ] Confirm `bootmodes=('uefi.systemd-boot')` matches the build host's archiso.
- [ ] Verify the os-release pacman hook wins over the `filesystem` package.
- [ ] Confirm greetd + Hyprland + PipeWire actually start on the live image (tty2 is the rescue console).
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

- [ ] Choose AGS v2/Astal toolkit versions and lock them in `desktop/ags/README.md`.
- [ ] Implement the unified shell (bar, notifications, launcher/control center) against the placeholder tree.
- [ ] Build from this source using official-repo deps in `packages/ags-build-deps.txt` (meson/npm/go/GTK). Do not add an AUR helper to the default path.
- [ ] Decide ISO integration: vendor a build into `/usr/local` at image-build time, or post-install compile.
- [ ] `exec-once` the shell from `desktop/hypr/hyprland.conf` once it starts.

## 4. Nearby follow-ups (not blockers for the three tracks above)

- [ ] Branded greetd greeter (theme hooks are placeholders).
- [ ] Wallpaper and icon/cursor theme assets under `branding/`.
- [ ] XLibre session packaging decision (see [DESIGN.md](../DESIGN.md#xlibre-session-path)); still no Coda repo.
- [ ] NVIDIA detect/install hook implementation.
- [ ] Optional CUPS profile using `packages/optional-cups.txt`.
- [ ] Plymouth — still deferred.
