# archinstall profile

CodaLinux installs with **archinstall**, not Calamares. This directory is a stub tailored to locked defaults.

## Files

| File | Role |
| --- | --- |
| `user_configuration.json` | Guided-installer answers: systemd-boot, PipeWire, hostname `coda`; `packages` is generated |
| `packages.txt` | Generated package list (no live-only tools) — source for the JSON array |
| `profiles/codalinux.py` | Custom profile hooks — **not wired yet** |

Disk layout is machine-specific and left empty on purpose. Generate a layout on target hardware with `archinstall --dry-run` and merge the CodaLinux fields.

Credentials (`user_credentials.json`) must never be committed.

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
