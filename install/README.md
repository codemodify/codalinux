# archinstall profile

CodaLinux installs with **archinstall**, not Calamares. This directory is a stub tailored to locked defaults.

## Files

| File | Role |
| --- | --- |
| `user_configuration.json` | Guided-installer answers: systemd-boot, PipeWire, hostname `coda`; `packages` is generated |
| `packages.txt` | Generated package list (no live-only tools) — source for the JSON array |
| `profiles/codalinux.py` | Custom profile hooks — **not wired yet** |

Locale, timezone, and keymap are **fixed** to Bozeman, Montana (`en_US.UTF-8`, `America/Denver`, `us`). `coda-install` must not ask for them. Disk is asked only when `CODA_INSTALL_DISK` is unset. With a disk, the helper imports `archinstall.lib.disk.device_handler` and `suggest_single_disk_layout` from `disk_menu` (not the legacy `devicehandler` module) and writes `disk_config` for `--silent`.

`coda-install` as user `live` writes `$XDG_RUNTIME_DIR/codalinux-archinstall.json` (or `/tmp/codalinux-archinstall-$UID.json`; override `CODA_ARCHINSTALL_RUNTIME`). It does **not** write `/run/…` (root-only). `disk_config` is generated with `sudo -E python3 … --emit-layout` — as `live`, `import archinstall.lib.disk.device_handler` hits “maximum recursion depth exceeded”. Already-root skips that extra sudo. Then `exec sudo -E -- archinstall --config <that path> [--creds …] [--silent]`.

Credentials must never be committed. Silent install still needs them:

```bash
# Existing archinstall creds JSON (gitignored if named user_credentials.json)
CODA_INSTALL_DISK=/dev/vda CODA_INSTALL_CREDS=./user_credentials.json coda-install

# Opt-in test user (not a production password)
CODA_INSTALL_DISK=/dev/vda \
  CODA_INSTALL_USER=coda CODA_INSTALL_PASSWORD='…' \
  CODA_INSTALL_ROOT_PASSWORD='…' \
  coda-install
```

## What archinstall does not do for us yet

Upstream guided/desktop profiles do **not** match CodaLinux:

- Greeter enums are SDDM / LightDM / GDM / Ly — **greetd is custom**.
- Desktop networking often pulls **NetworkManager** — we use systemd-networkd + iwd.
- Hyprland may exist as a community profile; it still will not enable our greetd + networkd + os-release hook.

`profiles/codalinux.py` lists the hooks a real profile (or `archinstall --script`) must implement. Until then, `user_configuration.json` only encodes the parts guided install already understands.

## Intended invocation (later)

```bash
archinstall --config /usr/share/codalinux/install/user_configuration.json
# plus a custom script/profile once codalinux.py is implemented
```

Keep `"additional-repositories": []`. Do not add a Coda repo.

The composed install set includes `bubblewrap`. Extra software after install belongs in `coda-sandbox` under `~/.coda/sandbox/<name>` (no sudo; one named root holds many packages — see [architecture.md](../architecture.md) and [docs/sandbox.md](../docs/sandbox.md)). Disk layout is still a single ext4 `/` until a core-vs-data split lands. Do not treat A/B OS slots as implemented.
