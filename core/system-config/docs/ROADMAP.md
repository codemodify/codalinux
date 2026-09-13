# system-config remaining work

Shipped in-tree (not on the ISO): display, network, audio, bluetooth, input, datetime, locale apply, devices (summary/pci/usb), hardware.dmi, session lock, power (brightness/lid/suspend/hibernate). CLI + GUI TreeView pages for each.

Still open:

- Wire binaries onto the live/install image.
- Real udev netlink watch (report `--watch` is a poll stub).
- Persist Hyprland input/display beyond runtime `hyprctl eval` (hyprland.lua rewrite is out of scope).
- iwd scan quality / hidden SSIDs; networkd drop-in merge with existing `*.network` files.
- PipeWire node names vs numeric ids when wpctl status format changes.
- BlueZ D-Bus instead of `bluetoothctl` text; pairing agent for interactive PIN.
- Session idle inhibit; reboot/poweroff remain **not** allowlisted.
- Multi-monitor position (display later).
- Replace AGS + `coda-settings` (explicitly **not** a migrate in v1).
- uitoolkit gaps: [`uitoolkit-gaps.md`](uitoolkit-gaps.md) — do not block; compose existing widgets.
