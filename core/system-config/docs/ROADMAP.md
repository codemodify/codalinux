# system-config remaining work

Shipped on the live ISO: display, network, audio, bluetooth, input, datetime, locale apply, devices (summary/pci/usb), hardware.dmi, session lock, power (backlight + logind CanSuspend/CanHibernate). CLI + GUI TreeView pages for each. External probes (`bluetoothctl`/`iwctl`/`wpctl`/…) have hard timeouts so D’s accept loop cannot hang. Hyprland display/input apply persists to `~/.config/hypr/coda-system-config.lua` (dofile from `hyprland.lua`). iwd connect writes `/var/lib/iwd/<ssid>.psk` then `iwctl --passphrase`. Guest e2e: `scripts/guest-e2e-all.sh --guest` (host-refused).

`system-config-report --watch` listens on `NETLINK_KOBJECT_UEVENT` and re-probes the mapped path (net→network, usb/pci, bluetooth, input, backlight, drm→display). A **slow poll** (default 30s) remains for L2 stacks that do not emit kobject uevents: Hyprland scale/mode, PipeWire volume, timedatectl/locale, logind session.

Still open:
- Hidden SSIDs / iwd scan quality; networkd drop-in merge with existing `*.network` files.
- PipeWire node names vs numeric ids when wpctl status format changes.
- BlueZ D-Bus instead of `bluetoothctl` text; pairing agent for interactive PIN.
- Session idle inhibit; reboot/poweroff remain **not** allowlisted.
- Multi-monitor position (display later).
- Replace AGS + `coda-settings` (explicitly **not** a migrate in v1).
- uitoolkit gaps: [`uitoolkit-gaps.md`](uitoolkit-gaps.md) — do not block; compose existing widgets.
