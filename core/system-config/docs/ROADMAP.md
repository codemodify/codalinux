# system-config remaining work

Shipped on the live ISO and this pass: display (scale/mode/position, multi-monitor persist), network (iwd scan quality + security, networkd drop-in merge, static IP/DNS/search that survives reboot, airplane/rfkill), audio (full sink/source names via `pw-dump`, per-node volume, session default routing), bluetooth (BlueZ D-Bus + pairing agent; `bluetoothctl --timeout` fallback), input (hypr persist + localectl/vconsole), datetime/locale, session (lock, seats, idle inhibit observe), power (backlight + logind lid; suspend/hibernate gated; **no** reboot/poweroff), printers/users/storage observe (`present=false` when empty/missing). CLI + GUI TreeView pages for each.

**Desktop cutover:** AGS Control Center, bar audio/Wi-Fi/Bluetooth, and `system-config-gui.desktop` launch **`system-config-gui`**. `coda-settings` is optional session chrome only (wallpaper, workspaces, GTK appearance). Documented in [`architecture.md`](../../../architecture.md#system-config).

External probes have hard timeouts so D’s accept loop cannot hang. Hyprland display/input apply persists to `~/.config/hypr/coda-system-config.lua` (dofile from `hyprland.lua`). iwd connect writes `/var/lib/iwd/<ssid>.psk` then `iwctl --passphrase`. Guest e2e: `scripts/guest-e2e-all.sh --guest` (host-refused; waits for Hyprland outputs before display apply; never suspends/hibernates/locks).

`system-config-report` serves scan RPCs **and** watches `NETLINK_KOBJECT_UEVENT` by default (`--no-watch` / `--watch`). Mapped paths include block→storage. A **slow poll** remains for L2 stacks that do not emit kobject uevents.

Still open (not this pass / not product-complete blockers for coding):
- Interactive PIN UI in the GUI (agent auto-accepts / uses staged PIN; no dialog widget yet).
- Persistent default audio routing across reboot (PipeWire session default only).
- Printer/user/storage **apply** (observe-only by design until a gated allowlist exists).
- uitoolkit gaps: [`uitoolkit-gaps.md`](uitoolkit-gaps.md) — do not block; compose existing widgets.
- Parent ISO rebuild + guest e2e after this coding pass.
