# system-config

Greenfield Coda **core** settings stack (Go). Architecture: [architecture.md — system-config](../../architecture.md#system-config).

Clients talk to **`system-configd` only**. Report never writes config. Apply only executes D’s closed allowlist (typed argv, **no** arbitrary shell). Coda stack: systemd-networkd + iwd, PipeWire, BlueZ, Hyprland, greetd.

## Build

Needs Go 1.22+ (Arch `go`). Tests and helper binaries stay `CGO_ENABLED=0`. **`system-config-gui` must be `CGO_ENABLED=1`** — uitoolkit Wayland/X11 backends are Linux+CGO; CGO off falls through to offscreen (process lives, no window). ISO/install builds use `scripts/install-system-config.sh`, which refuses a static/headless GUI. The `/usr/local/bin/system-config-gui` wrapper prefers Wayland and defaults `UITK_PAINT=cpu` (virtio-gpu EGL often fails; wl_shm is the v1 path).

```bash
cd core/system-config
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go build -o bin/ ./cmd/system-configd ./cmd/system-config-apply \
  ./cmd/system-config-report ./cmd/system-config ./cmd/system-config-tui
CGO_ENABLED=1 go build -o bin/ ./cmd/system-config-gui
ldd bin/system-config-gui | grep wayland
```

`system-config-gui` uses [`github.com/codemodify/uitoolkit@dev`](https://github.com/codemodify/uitoolkit) (v0.19.x is enough). Mail/Settings pattern: app-level Unix socket + JSON-lines client, two-pane `Splitter` + `TreeView`, **per-section Apply** (dirty vs that page’s baseline), `TableView` for devices (capped), `Slider`/`NumberField` for display scale. Gaps: [`docs/uitoolkit-gaps.md`](docs/uitoolkit-gaps.md). Headless (`-headless` / `-screenshot` only):

```bash
go run ./cmd/system-config-gui -headless
```

Live/install ISO ships the six binaries under `/usr/local/bin` plus systemd user units (`system-configd`, `system-config-report`) and the root `system-config-apply` unit. `coda-hyprland` also starts the user daemons. Guest smoke stays host-refused (`--guest` + `ID=codalinux`).

## Run order (user session)

Sockets default to `$XDG_RUNTIME_DIR/coda/` (or `$TMPDIR/coda-$UID/`).

```bash
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/tmp/coda-run}"
mkdir -p "$XDG_RUNTIME_DIR/coda"

./bin/system-configd &
./bin/system-config-report &
./bin/system-config-apply &

# report RPC server also watches udev by default. --once pushes and exits.
./bin/system-config-report --once
./bin/system-config-report --watch      # watch only
./bin/system-config-report --no-watch   # scan socket only

./bin/system-config refresh display
./bin/system-config get display
./bin/system-config set display '{"outputs":[{"name":"Virtual-1","scale":2}]}'
./bin/system-config apply display
./bin/system-config get network
./bin/system-config get audio
./bin/system-config get bluetooth
./bin/system-config get input
./bin/system-config get datetime
./bin/system-config get locale
./bin/system-config get devices.usb
./bin/system-config get hardware.dmi
./bin/system-config get session
./bin/system-config get power
./bin/system-config get printers
./bin/system-config get users
./bin/system-config get storage
./bin/system-config watch display
./bin/system-config watch display --follow
./bin/system-config-tui -dump
./bin/system-config-tui            # TTY: every KnownPath, per-section Apply
```

`apply display` needs Hyprland (`hyprctl`). Apply may run as root; it discovers the graphical session (`loginctl` / `/run/user/*/hypr/*`) and runs `hyprctl` as that uid with `XDG_RUNTIME_DIR` + `HYPRLAND_INSTANCE_SIGNATURE` (same env `coda-settings` expects). Report uses the same discovery so `refresh display` fills `observed.outputs`. Eval form is `hl.monitor({ output = "NAME", ... })` — not `name=`, not `keyword`.

## QEMU guest smoke (never the host)

[`scripts/guest-smoke.sh`](scripts/guest-smoke.sh) is for a **live CodaLinux QEMU guest only**. It refuses unless `--guest` and `/etc/os-release` is `ID=codalinux`. Do not run it on the build host.

```bash
# inside the guest, binaries on PATH
./scripts/guest-smoke.sh --guest
```

That starts the three daemons (D + report as the seat user, apply as root with `CODA_SYSTEM_CONFIG_UID`), then `refresh display` (expect `Virtual-1`), `set` scale 2, `apply`, and checks `hyprctl -j monitors` scale.

Full guest e2e (every KnownPath refresh, safe applies only, optional `coda-sandbox` create/exec/destroy):

```bash
./scripts/guest-e2e-all.sh --guest
```

Prints a PASS/FAIL table and exits non-zero on any fail. Never suspends/hibernates. ISO copies both scripts to `/usr/local/share/codalinux/system-config/`.

## Binaries

| Binary | Role |
| --- | --- |
| `system-configd` | Unprivileged control plane (desired + observed, plans) |
| `system-config-apply` | Typed executor (plans from D) |
| `system-config-report` | Inventory → observed (udev/sysfs/DMI + `hyprctl -j monitors`) |
| `system-config` | CLI → D |
| `system-config-tui` | Terminal Settings → D (every KnownPath; per-section Apply) |
| `system-config-gui` | Settings GUI (uitoolkit Mail pattern) → D |

## Protocol

JSON lines on a Unix socket. Peer-cred (SO_PEERCRED) restricts connections to the same uid (and root). Ops: `get`, `set`, `watch`, `refresh` (ask report), `apply` (ask apply).

**watch:** default is one snapshot (same fields as `get`) plus a note. The connection stays request/response so CLI `watch` then `set` still works. `data: {"follow":true}` holds the connection and writes further JSON-line responses when D’s store changes — typically `put-observed` from report (udev/netlink + the existing slow poll). Identical re-pushes are not emitted. `timeout_ms` ends the stream. Empty path / `submodels` watches every KnownPath. CLI: `system-config watch display --follow`. TUI uses follow to refresh a clean section live.

**TUI:** `system-config-tui` is a stdlib + `x/sys` terminal UI (no extra TUI module; CGO off). Every KnownPath is a page (refresh / get / edit; Apply only when that section is dirty vs its baseline). Observe-only paths have no Apply. Non-TTY or `-dump` prints the path list. Clients still talk to D only.

**Paths:** `display` `network` `audio` `bluetooth` `input` `datetime` `locale` `session` `power` `printers` `users` `storage` `devices.summary` `devices.pci` `devices.usb` `hardware.dmi`

**Apply allowlist:** `display.scale` `display.mode` `display.position` `network.iface.enable` `network.iface.method` `network.wifi.connect` `network.wifi.disconnect` `network.airplane` `audio.default.sink` `audio.default.source` `audio.volume` `audio.mute` `bluetooth.power` `bluetooth.scan` `bluetooth.pair` `bluetooth.connect` `bluetooth.disconnect` `bluetooth.trust` `input.keymap` `input.kb_layout` `input.pointer.speed` `input.pointer.natural_scroll` `input.touchpad.tap` `datetime.timezone` `datetime.ntp` `datetime.time` `locale.lang` `locale.keymap` `session.lock` `power.suspend` `power.hibernate` `power.brightness` `power.lid` `printers.default` `printers.enable` `users.shell` `storage.mount` `storage.unmount`

Reboot/poweroff are **not** allowlisted. Desktop Settings is `system-config-gui` only (`coda-settings` is hidden). D/report refuse uid 0.

Display eval stays `hl.monitor({ output = "NAME", ... })`. Hyprland/PipeWire tools use session discovery (`internal/hyprsession`). Remaining work: [`docs/ROADMAP.md`](docs/ROADMAP.md).
