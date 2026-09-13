# system-config remaining work

Shipped on the live ISO: display, network, audio, bluetooth, input, datetime, locale apply, devices (summary/pci/usb), hardware.dmi, session lock, power (backlight + logind CanSuspend/CanHibernate). CLI + GUI TreeView pages for each. External probes (`bluetoothctl`/`iwctl`/`wpctl`/…) have hard timeouts so D’s accept loop cannot hang. Hyprland display/input apply persists to `~/.config/hypr/coda-system-config.lua` (dofile from `hyprland.lua`). iwd connect writes `/var/lib/iwd/<ssid>.psk` then `iwctl --passphrase`.

Still open:

- Real udev netlink watch (report `--watch` is a poll stub).
- Hidden SSIDs / iwd scan quality; networkd drop-in merge with existing `*.network` files.
- PipeWire node names vs numeric ids when wpctl status format changes.
- BlueZ D-Bus instead of `bluetoothctl` text; pairing agent for interactive PIN.
- Session idle inhibit; reboot/poweroff remain **not** allowlisted.
- Multi-monitor position (display later).
- Replace AGS + `coda-settings` (explicitly **not** a migrate in v1).
- uitoolkit gaps: [`uitoolkit-gaps.md`](uitoolkit-gaps.md) — do not block; compose existing widgets.
