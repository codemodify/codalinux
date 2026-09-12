# archinstall profile

CodaLinux installs with **archinstall**, not Calamares.

## Files

| File | Role |
| --- | --- |
| `user_configuration.json` | Guided-installer answers: systemd-boot, PipeWire, hostname `coda`; `packages` is generated |
| `packages.txt` | Generated package list (no live-only tools) — source for the JSON array |
| `profiles/codalinux.py` | Finish hook: runs `coda-install-post.sh` on a target root |
| `../scripts/coda-install-post.sh` | Real post-install: copy live `/usr/local` desktop, greetd for `user`, networkd+iwd |

Locale, timezone, and keymap are **fixed** to Bozeman, Montana (`en_US.UTF-8`, `America/Denver`, `us`). `coda-install` must not ask for them. Disk is asked only when `CODA_INSTALL_DISK` is unset.

## Default login (always printed)

```
Login after reboot:
  user: user
  password: 1
```

Root password is also `1`. Automation may override with `CODA_INSTALL_CREDS` or `CODA_INSTALL_USER` / `CODA_INSTALL_PASSWORD` / `CODA_INSTALL_ROOT_PASSWORD`.

`coda-install` as user `live` writes `$XDG_RUNTIME_DIR/codalinux-archinstall.json` (or `/tmp/codalinux-archinstall-$UID.json`). `disk_config` is generated with `sudo -E python3 … --emit-layout`. After archinstall exits 0, **`coda-install-post.sh /mnt`** runs on the live ISO (not inside arch-chroot — that cannot see live `/usr/local`).

Post-install on the target:

1. Copy live `/usr/local` Coda bits (`coda-hyprland`, `coda-ags`, `ags`, hyprbars, Astal).
2. Hyprland configs into `/etc/xdg/hypr`, `/etc/skel`, and `/home/user`.
3. `codalinux-hyprland.desktop` Wayland session.
4. greetd enabled, autologin **`user`** → `/usr/local/bin/coda-hyprland` (never `live`).
5. systemd-networkd, systemd-resolved, iwd enabled; NetworkManager not required.
6. Branding/os-release hook if present.

```bash
CODA_INSTALL_DISK=/dev/vda coda-install
```

Keep `"additional-repositories": []`. Do not add a Coda repo.

The composed install set includes `bubblewrap`. Extra software after install belongs in `coda-sandbox` under `~/.coda/sandbox/<name>`.
