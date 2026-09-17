# Implementation TODOs

Scaffolding only. Each item is work for a later change. Do not treat stubs as finished.

## 1. Live ISO (`archiso/` + `scripts/build-iso.sh`)

- [x] Profile includes mkinitcpio-archiso hooks, pacman-init, UEFI systemd-boot entries, greetd/iwd/networkd enables, os-release overlay, `coda-install`.
- [x] `scripts/build-iso.sh` prefers native/rootless `mkarchiso`; Docker/Podman only if already usable without sudo (no sudo fallbacks).
- [x] First `mkarchiso` completed (ISO 9660, label `CODA_202609`). Boot on OVMF/hardware still unverified.
- [ ] Run `mkarchiso` and boot the image on UEFI firmware. Preferred automated path is `./scripts/qemu-boot-test.sh` (QEMU/KVM + OVMF). VirtualBox EFI/VMSVGA remains a manual check only.
- [ ] Confirm `bootmodes=('uefi.systemd-boot')` matches the build host's archiso.
- [ ] Verify the os-release pacman hook wins over the `filesystem` package.
- [x] Live autologin: user `live` via `coda-hyprland` (VM software-render path + `/var/log/coda-hyprland.log`). Verify on VirtualBox EFI/VMSVGA.
- [x] Vendored AGS/Astal shell on the live ISO (bar, launcher, notifications, control center) plus official settings apps (impala / blueman / pavucontrol / snapshot / nwg-look). Waybar interim rejected.
- [ ] Confirm greetd + Hyprland + PipeWire + AGS actually start on the live image (tty2 is the rescue console).
- [ ] Accessibility / speech boot entry (optional; not in v1).
- [ ] Keep the ISO official-repos-only; no `[codalinux]` repo, no AUR helper.
- [ ] Periodic rebuild pipeline (manual first; CI only if an Arch builder exists).

## 2. Installer (`install/` + `scripts/coda-install*`)

- [x] `coda-install` + `coda-install-post.sh`: greetd autologin `user` on the **desktop** tree, networkd+sshd+QGA on the slot, default creds `user`/`1`.
- [x] Operator picks **disk only** (or `CODA_INSTALL_DISK` / `auto` for e2e). Bozeman locale/tz/keymap not asked.
- [x] Automatic GPT layout ESP + OS-A + OS-B + data with documented size floors; fail if the disk is too small.
- [x] Offline first install into OS-A (live airootfs **split**: core → slot, desktop → `coda-data`; no pacstrap/mirrors). ISO **build** may still fetch packages.
- [x] systemd-boot entries A (default) and B (placeholder); data bind-mounted at `/home` and `/var`.
- [x] Host e2e: `scripts/qemu-install-e2e.sh` (local ISO, QEMU+QGA, no prompts).
- [x] Keep `additional-repositories` empty.
- [ ] `user_configuration.json` / archinstall profile are leftover (not the install path).
- [ ] NVIDIA: call `scripts/hooks/nvidia.sh` when detection is implemented — not before.

## 3. AGS / Astal shell (`desktop/ags/`)

- [x] Pin AGS 3.1.2 + Astal commits in `desktop/ags/README.md` and `scripts/vendor-ags.sh`.
- [x] Implement the unified shell (bar, notifications, launcher, control center).
- [x] AGS taskbar of running apps, workspace add/remove, and per-workspace tile ↔ overlapping-float stack via `coda-hypr-ws` (hyprfloat semantics; not tabbed groups).
- [x] Floating titlebars via vendored hyprbars (close / max / min); tiled windows stay border-only.
- [x] Vendor from source at ISO build time with official-repo deps in `packages/ags-build-deps.txt`. No AUR helper, no Coda repo.
- [x] Start AGS from `desktop/hypr/hyprland.lua` via `coda-ags` (Waybar is not a fallback).
- [ ] Confirm the vendored shell on a real live boot (`scripts/qemu-boot-test.sh`; VirtualBox VMSVGA still optional).

## 4. Nearby follow-ups (not blockers for the three tracks above)

- [ ] Confirm virtio-9p modules on the live image (`find /usr/lib/modules/$(uname -r) -name '*9p*'`). Add them only if that find is empty. Desktop UX iteration is `scripts/qemu-desktop-dev.sh`; ISO smoke is `scripts/qemu-boot-test.sh`; package/vendor/squashfs still needs `scripts/build-iso.sh`.

- [ ] Branded greetd greeter (theme hooks are placeholders).
- [x] Default hyprpaper / hyprlock still is Plasma Horos at `branding/wallpapers/default.png`. Full theme assets still later.
- [ ] XLibre session packaging decision (see [DESIGN.md](../DESIGN.md#xlibre-session-path)); still no Coda repo.
- [ ] NVIDIA detect/install hook implementation.
- [ ] Optional CUPS profile using `packages/optional-cups.txt`.
- [ ] Plymouth — still deferred.

## 5. Core OS vs sandboxes (bubblewrap)

Picture: [architecture.md](../architecture.md) (target A/B vs current). Shipped now: `packages/sandbox.txt` (`bubblewrap`), `scripts/coda-sandbox`, docs. Desktop (Hyprland + AGS) stays on the **live ISO**. Installed desktop lives on `coda-data`.

- [x] Installer disk layout: ESP + OS-A + OS-B + data (`/home` includes `~/.coda/sandbox`). Slots are **core-only** (4 GiB floor). Desktop is on `coda-data` (8 GiB data floor).
- [x] Inactive-slot writer + oneshot boot-test + promote. Product CLI: `coda-update core` / `desktop` / `status` (Arch repos → inactive slot / coda-data). `coda-slot` remains low-level. Host e2e: `scripts/qemu-install-e2e.sh` (local ISO only; `--from-iso`).
- [ ] QEMU **network** e2e of `coda-update core` (Arch pacman into the inactive slot). Hook: `CODA_E2E_CORE_FROM=repos`. Not claimed green (current e2e has no NIC).
- [x] Offline split: core packages → slot; Hyprland/AGS/session → `/coda/data/desktop`; boot merge via `coda-desktop-mount`.
- [ ] Read-only remount of the **running** slot and gated host `pacman`. Documented in DESIGN.md; **not implemented**.
- [ ] Core-only (~1.1 GiB) *live ISO* (desktop as a session on top of a minimal squashfs). The live image is still the full desktop.
- [ ] A/B the desktop payload (today `coda-update desktop` refreshes `/coda/data/desktop` in place).
- [ ] Optional Distrobox / Podman **alongside** `coda-sandbox`, not as a replacement.
- [ ] Flatpak alongside bwrap (still official-repo / documented policy only).
- [ ] GUI entry from the AGS Settings hub (list / shell / destroy).
- [x] User-owned create/install/destroy via `unshare --map-root-user` (no sudo). Store: `~/.coda/sandbox/<env>` (one env = many packages).
- [ ] Confirm `coda-sandbox create` on a live ISO with network (`pacman --root base` in a user namespace).

## 6. system-config (Go core)

Locked in [architecture.md](../architecture.md#system-config). Code: [`core/system-config/`](../core/system-config/README.md).

- [x] Greenfield Go module: `system-configd`, apply, report, CLI, TUI (every KnownPath), uitoolkit GUI.
- [x] `watch` follow stream: default stays one snapshot; `{"follow":true}` emits when observed/desired actually changes.
- [x] Settings GUI Mail/Settings pattern on uitoolkit `dev` (v0.19.x enough to ship). Remaining gaps in `core/system-config/docs/uitoolkit-gaps.md` — do not block.
- [x] Wire binaries + systemd units onto the live/install image.
- [x] Report `--watch` udev netlink (`NETLINK_KOBJECT_UEVENT`); 30s L2 poll remains for Hyprland/PipeWire/timedatectl.
- [x] Guest e2e: `core/system-config/scripts/guest-e2e-all.sh --guest` (host-refused).
- [x] Network / audio / bluetooth / input / datetime / locale apply allowlist (plus session lock, power).
- [x] Persist Hyprland display/input to `~/.config/hypr/coda-system-config.lua`; iwd PSK file + `--passphrase`.
- [x] BlueZ D-Bus pairing agent + GUI PIN field / `$XDG_RUNTIME_DIR/coda/bluetooth-pin`. See `core/system-config/docs/ROADMAP.md`.
- [x] Desktop Settings is `system-config-gui` only (not a migrate from `coda-settings`).
