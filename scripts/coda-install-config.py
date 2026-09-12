#!/usr/bin/env python3
"""Build an archinstall config with Bozeman defaults; ask only for the disk."""

from __future__ import annotations

import asyncio
import inspect
import json
import os
import subprocess
import sys
from pathlib import Path
from typing import Any


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


def runtime_config_path() -> Path:
    """User-writable archinstall JSON. /run is root-only on the live ISO."""
    override = os.environ.get("CODA_ARCHINSTALL_RUNTIME", "").strip()
    if override:
        return Path(override)
    xdg = os.environ.get("XDG_RUNTIME_DIR", "").strip()
    if xdg:
        runtime_dir = Path(xdg)
        try:
            if runtime_dir.is_dir() and os.access(runtime_dir, os.W_OK):
                return runtime_dir / "codalinux-archinstall.json"
        except OSError:
            pass
    return Path(f"/tmp/codalinux-archinstall-{os.getuid()}.json")


def creds_hint() -> None:
    print("Silent install still needs archinstall credentials (not stored in git):")
    print("  CODA_INSTALL_CREDS=/path/to/user_credentials.json")
    print("  or CODA_INSTALL_USER=name CODA_INSTALL_PASSWORD=…")
    print("    optional: CODA_INSTALL_ROOT_PASSWORD=…")


def maybe_creds_path() -> Path | None:
    preset = os.environ.get("CODA_INSTALL_CREDS", "").strip()
    if preset:
        path = Path(preset)
        if not path.is_file():
            print(f"CODA_INSTALL_CREDS={preset} is not a file.")
            return None
        return path
    user = os.environ.get("CODA_INSTALL_USER", "").strip()
    password = os.environ.get("CODA_INSTALL_PASSWORD", "")
    root_pw = os.environ.get("CODA_INSTALL_ROOT_PASSWORD", "")
    if not user and not root_pw:
        return None
    if user and not password:
        print("CODA_INSTALL_USER is set but CODA_INSTALL_PASSWORD is empty.")
        return None
    creds: dict[str, Any] = {}
    if root_pw:
        creds["!root-password"] = root_pw
    if user:
        creds["!users"] = [
            {
                "username": user,
                "!password": password,
                "sudo": True,
                "groups": [],
            }
        ]
    path = runtime_config_path().with_name(f"codalinux-archinstall-creds-{os.getuid()}.json")
    path.write_text(json.dumps(creds, indent=2) + "\n", encoding="utf-8")
    os.chmod(path, 0o600)
    return path


def archinstall_cmd(
    config_path: Path, extra: list[str], silent: bool, creds: Path | None
) -> list[str]:
    cmd = ["archinstall", "--config", str(config_path)]
    if creds is not None:
        cmd.extend(["--creds", str(creds)])
    if silent:
        cmd.append("--silent")
    cmd.extend(extra)
    if os.geteuid() == 0:
        return cmd
    return ["sudo", "-E", "--", *cmd]


def _import_device_handler() -> Any | None:
    try:
        from archinstall.lib.disk.device_handler import device_handler

        return device_handler
    except Exception as current_exc:
        try:
            from archinstall.lib.disk.devicehandler import device_handler  # type: ignore

            return device_handler
        except Exception:
            print(f"Could not import archinstall.lib.disk.device_handler ({current_exc}).")
            return None


def _import_suggest_layout() -> Any | None:
    # Current archinstall: async helper in disk_menu. Older: filesystem.
    try:
        from archinstall.lib.disk.disk_menu import suggest_single_disk_layout

        return suggest_single_disk_layout
    except Exception:
        pass
    try:
        from archinstall.lib.disk.filesystem import suggest_single_disk_layout

        return suggest_single_disk_layout
    except Exception as exc:
        print(f"Could not import suggest_single_disk_layout ({exc}).")
        return None


def _ext4_type() -> Any:
    try:
        from archinstall.lib.models.device import FilesystemType

        return FilesystemType.EXT4
    except Exception:
        return "ext4"


def _resolve_device(device_handler: Any, device: str) -> Any | None:
    path = Path(device)
    for getter in ("get_device", "get_device_by_path"):
        fn = getattr(device_handler, getter, None)
        if fn is None:
            continue
        for arg in (path, device):
            try:
                dev = fn(arg)
            except Exception:
                dev = None
            if dev:
                return dev
    for item in getattr(device_handler, "devices", None) or []:
        info = getattr(item, "device_info", None)
        raw = None
        if info is not None:
            raw = getattr(info, "path", None)
        raw = raw or getattr(item, "path", None) or getattr(item, "device_path", None)
        if raw is None:
            continue
        if Path(str(raw)) == path or str(raw) == device:
            return item
    return None


def _call_suggest(fn: Any, dev: Any, fs: Any) -> Any | None:
    attempts: list[dict[str, Any]] = [
        {"filesystem_type": fs, "separate_home": False},
        {"filesystem_type": fs},
        {},
    ]
    last_type_error: Exception | None = None
    for kwargs in attempts:
        try:
            if inspect.iscoroutinefunction(fn):
                return asyncio.run(fn(dev, **kwargs))
            return fn(dev, **kwargs)
        except TypeError as exc:
            last_type_error = exc
            continue
        except Exception as exc:
            print(f"suggest_single_disk_layout failed: {exc}")
            return None
    if last_type_error is not None:
        print(f"suggest_single_disk_layout failed: {last_type_error}")
    return None


def layout_to_disk_config(layout: Any) -> dict | None:
    """Normalize DeviceModification or DiskLayoutConfiguration to archinstall JSON."""
    if isinstance(layout, dict):
        if layout.get("config_type"):
            return layout
        if "device_modifications" in layout:
            return layout
        if "device" in layout and "partitions" in layout:
            return {
                "config_type": "default_layout",
                "device_modifications": [layout],
            }
        return None
    for method in ("json", "model_dump"):
        fn = getattr(layout, method, None)
        if fn is None:
            continue
        try:
            data = fn()
        except Exception:
            continue
        if isinstance(data, dict):
            converted = layout_to_disk_config(data)
            if converted:
                return converted
    try:
        from archinstall.lib.models.device import DiskLayoutConfiguration, DiskLayoutType

        if hasattr(layout, "partitions") and hasattr(layout, "device"):
            cfg = DiskLayoutConfiguration(
                config_type=DiskLayoutType.Default,
                device_modifications=[layout],
            )
            return cfg.json()
    except Exception:
        pass
    return None


def suggest_layout(device: str) -> dict | None:
    """Build disk_config via current archinstall helpers (device_handler + disk_menu)."""
    device_handler = _import_device_handler()
    suggest_fn = _import_suggest_layout()
    if device_handler is None or suggest_fn is None:
        return None

    try:
        device_handler.load_devices()
    except Exception:
        pass
    dev = _resolve_device(device_handler, device)
    if dev is None:
        print(f"archinstall did not recognize {device}.")
        return None
    raw = _call_suggest(suggest_fn, dev, _ext4_type())
    if raw is None:
        return None
    data = layout_to_disk_config(raw)
    if data is None:
        print("suggest_single_disk_layout returned an unusable layout.")
    return data


def main(argv: list[str]) -> int:
    src = Path(argv[1]) if len(argv) > 1 else Path(
        "/usr/share/codalinux/install/user_configuration.json"
    )
    extra = argv[2:]
    cfg = load_config(src)
    runtime = runtime_config_path()
    runtime.parent.mkdir(parents=True, exist_ok=True)

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

    creds = maybe_creds_path()
    if silent and creds is None:
        creds_hint()

    runtime.write_text(json.dumps(cfg, indent=2) + "\n", encoding="utf-8")
    os.chmod(runtime, 0o644)
    cmd = archinstall_cmd(runtime, extra, silent, creds)
    print(f"Running: {' '.join(cmd)}")
    os.execvp(cmd[0], cmd)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
