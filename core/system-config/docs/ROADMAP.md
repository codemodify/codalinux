# system-config remaining work

Shipped on the live ISO and this pass: display (scale/mode/position, multi-monitor persist), network (iwd scan quality + security, networkd drop-in merge, static IP/DNS/search that survives reboot, airplane/rfkill), audio (full sink/source names via `pw-dump`, per-node volume, default routing persisted to `~/.config/wireplumber/wireplumber.conf.d/51-coda-defaults.conf`), bluetooth (BlueZ D-Bus + pairing agent with GUI PIN field + `$XDG_RUNTIME_DIR/coda/bluetooth-pin`; `bluetoothctl --timeout` fallback), input (hypr persist + localectl/vconsole), datetime/locale, session (lock, seats, idle inhibit observe), power (backlight + logind lid; suspend/hibernate gated; **no** reboot/poweroff), printers/users/storage **apply** (`lpadmin`/`cupsenable`, `usermod -s`, `udisksctl` mount/unmount; system mounts refused). CLI + GUI TreeView pages for each.

**Desktop Settings:** AGS Control Center, bar audio/Wi-Fi/Bluetooth, and `system-config-gui.desktop` launch **`system-config-gui` only**. Session chrome (wallpaper, workspaces, GTK, hypr gaps) calls `coda-wallpaper` / `coda-hypr-ws` / `nwg-look` / `thunar` / `hyprctl` directly. `coda-settings` stays on the image as a hidden helper binary (`NoDisplay=true`); it is not a Settings app. Documented in [`architecture.md`](../../../architecture.md#system-config).

**Install:** `coda-install-post` copies the same `system-config*` binaries, apply/D/report units, graphical-session wants, and `system-config-gui.desktop` as live. D/report refuse uid 0 (`ConditionUser=!root`, `coda-hyprland` skip, `rootguard`) so they do not bind `/run/user/0`.

External probes have hard timeouts so D’s accept loop cannot hang. Hyprland display/input apply persists to `~/.config/hypr/coda-system-config.lua` (dofile from `hyprland.lua`). iwd connect writes `/var/lib/iwd/<ssid>.psk` then `iwctl --passphrase`. Guest e2e: `scripts/guest-e2e-all.sh --guest` (host-refused; waits for Hyprland outputs; printers/users/storage refresh; static IP dry-apply + restore; rfkill observe; never suspends/hibernates/locks).

`system-config-report` serves scan RPCs **and** watches `NETLINK_KOBJECT_UEVENT` by default (`--no-watch` / `--watch`). Mapped paths include block→storage. A **slow poll** remains for L2 stacks that do not emit kobject uevents.

## Later work (intentional; not incomplete product code)

- uitoolkit gaps: [`uitoolkit-gaps.md`](uitoolkit-gaps.md) — compose existing widgets; do not block on PrefsPage / modal dialogs. Bluetooth PIN is the GUI TextField + pin file (no toolkit dialog widget).
- Parent ISO rebuild + guest e2e after this coding pass (live smoke, not a missing feature).
