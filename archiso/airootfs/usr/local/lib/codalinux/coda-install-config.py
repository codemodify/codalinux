#!/usr/bin/env python3
"""Build an archinstall config with Bozeman defaults; ask only for the disk."""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path


BOZEMAN = {
    "timezone": "America/Denver",
    "archinstall-language": "English",
    "locale_config": {
        "kb_layout": "us",
        "sys_enc": "UTF-8",
        "sys_lang": "en_US",
    },
}


def load_config(path: Path) -> dict:
    cfg = json.loads(path.read_text(encoding="utf-8"))
    cfg.update(BOZEMAN)
    cfg["locale_config"] = dict(BOZEMAN["locale_config"])
    return cfg


def list_disks() -> list[str]:
    skip = {"loop", "sr", "ram", "zram", "dm-", "md"}
    disks: list[str] = []
    try:
        out = subprocess.check_output(
            ["lsblk", "-dnpo", "NAME,SIZE,TYPE,MODEL"],
            text=True,
        )
    except (OSError, subprocess.CalledProcessError):
        return disks
    for line in out.splitlines():
        parts = line.split(None, 3)
        if len(parts) < 3:
            continue
        name, size, typ = parts[0], parts[1], parts[2]
        model = parts[3] if len(parts) > 3 else ""
        if typ != "disk":
            continue
        base = name.rsplit("/", 1)[-1]
        if any(base.startswith(p) for p in skip):
            continue
        disks.append(f"{name}  {size}  {model}")
    return disks


def pick_disk() -> str | None:
    preset = os.environ.get("CODA_INSTALL_DISK", "").strip()
    if preset:
        return preset
    disks = list_disks()
    if not disks:
        print("No disks found. Launching archinstall TUI for disk selection only.")
        return None
    print("Select the install disk (this will be partitioned).")
    for i, row in enumerate(disks, 1):
        print(f"  {i}) {row}")
    raw = input("Disk number or path (empty = archinstall disk menu): ").strip()
    if not raw:
        return None
    if raw.isdigit():
        idx = int(raw)
        if 1 <= idx <= len(disks):
            return disks[idx - 1].split()[0]
        return None
    return raw


def suggest_layout(device: str) -> dict | None:
    """Use archinstall's helper when the installed version exposes it."""
    try:
        from archinstall.lib.disk.device_handler import device_handler
        from archinstall.lib.disk.filesystem import suggest_single_disk_layout
    except Exception:
        try:
            from archinstall.lib.disk.devicehandler import device_handler  # type: ignore
            from archinstall.lib.disk.filesystem import suggest_single_disk_layout
        except Exception as exc:
            print(f"Could not import archinstall disk helpers ({exc}).")
            return None

    try:
        device_handler.load_devices()
    except Exception:
        pass
    dev = None
    for getter in ("get_device", "get_device_by_path"):
        fn = getattr(device_handler, getter, None)
        if fn:
            try:
                dev = fn(device)
            except Exception:
                dev = None
            if dev:
                break
    if dev is None:
        devices = getattr(device_handler, "devices", None)
        if devices:
            for item in devices:
                path = getattr(item, "path", None) or getattr(item, "device_path", None)
                if path == device:
                    dev = item
                    break
    if dev is None:
        print(f"archinstall did not recognize {device}.")
        return None
    try:
        layout = suggest_single_disk_layout(dev)
    except TypeError:
        try:
            layout = suggest_single_disk_layout(dev, filesystem_type="ext4")
        except Exception as exc:
            print(f"suggest_single_disk_layout failed: {exc}")
            return None
    except Exception as exc:
        print(f"suggest_single_disk_layout failed: {exc}")
        return None
    for method in ("json", "model_dump"):
        fn = getattr(layout, method, None)
        if fn:
            data = fn()
            if isinstance(data, dict):
                return data
    if isinstance(layout, dict):
        return layout
    return None


def main(argv: list[str]) -> int:
    src = Path(argv[1]) if len(argv) > 1 else Path(
        "/usr/share/codalinux/install/user_configuration.json"
    )
    extra = argv[2:]
    cfg = load_config(src)
    runtime = Path("/run/codalinux-archinstall.json")

    disk = pick_disk()
    silent = False
    if disk:
        layout = suggest_layout(disk)
        if layout:
            cfg["disk_config"] = layout
            silent = True
            print(f"Using ext4 default layout on {disk}.")
        else:
            print("Disk menu will be shown; locale/timezone/keymap stay Bozeman defaults.")

    runtime.write_text(json.dumps(cfg, indent=2) + "\n", encoding="utf-8")
    cmd = ["archinstall", "--config", str(runtime)]
    if silent:
        cmd.append("--silent")
    cmd.extend(extra)
    print(f"Running: {' '.join(cmd)}")
    os.execvp(cmd[0], cmd)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
