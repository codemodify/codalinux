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

DEFAULT_USER = "user"
DEFAULT_PASSWORD = "1"
DEFAULT_ROOT_PASSWORD = "1"
DEFAULT_GROUPS = ["wheel", "video", "audio", "input", "render"]
ENABLE_SERVICES = [
    "greetd",
    "systemd-networkd",
    "systemd-resolved",
    "iwd",
    "bluetooth",
]


def load_config(path: Path) -> dict:
    cfg = json.loads(path.read_text(encoding="utf-8"))
    cfg.update(BOZEMAN)
    cfg["locale_config"] = dict(BOZEMAN["locale_config"])
    services = list(cfg.get("services") or [])
    for svc in ENABLE_SERVICES:
        if svc not in services:
            services.append(svc)
    cfg["services"] = services
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


def print_login_banner(user: str, password: str) -> None:
    print()
    print("Login after reboot:")
    print(f"  user: {user}")
    print(f"  password: {password}")
    print()


def creds_from_file(path: Path) -> tuple[str, str]:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return DEFAULT_USER, DEFAULT_PASSWORD
    users = data.get("!users") or data.get("users") or []
    if users and isinstance(users[0], dict):
        name = str(users[0].get("username") or DEFAULT_USER)
        pw = str(users[0].get("!password") or users[0].get("password") or DEFAULT_PASSWORD)
        return name, pw
    return DEFAULT_USER, DEFAULT_PASSWORD


def resolved_account() -> tuple[str, str, str]:
    user = os.environ.get("CODA_INSTALL_USER", "").strip() or DEFAULT_USER
    password = os.environ.get("CODA_INSTALL_PASSWORD", "")
    if not password:
        password = DEFAULT_PASSWORD
    root_pw = os.environ.get("CODA_INSTALL_ROOT_PASSWORD", "") or DEFAULT_ROOT_PASSWORD
    return user, password, root_pw


def maybe_creds_path() -> tuple[Path, str, str]:
    """Always produce archinstall creds. Default account is user/1."""
    preset = os.environ.get("CODA_INSTALL_CREDS", "").strip()
    if preset:
        path = Path(preset)
        if not path.is_file():
            raise FileNotFoundError(f"CODA_INSTALL_CREDS={preset} is not a file.")
        user, password = creds_from_file(path)
        return path, user, password
    user, password, root_pw = resolved_account()
    creds: dict[str, Any] = {
        "!root-password": root_pw,
        "!users": [
            {
                "username": user,
                "!password": password,
                "sudo": True,
                "groups": list(DEFAULT_GROUPS),
            }
        ],
    }
    path = runtime_config_path().with_name(f"codalinux-archinstall-creds-{os.getuid()}.json")
    path.write_text(json.dumps(creds, indent=2) + "\n", encoding="utf-8")
    os.chmod(path, 0o600)
    return path, user, password


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


def _log(msg: str) -> None:
    print(msg, file=sys.stderr)


def _import_device_handler() -> Any | None:
    try:
        from archinstall.lib.disk.device_handler import device_handler

        return device_handler
    except Exception as current_exc:
        try:
            from archinstall.lib.disk.devicehandler import device_handler  # type: ignore

            return device_handler
        except Exception:
            _log(f"Could not import archinstall.lib.disk.device_handler ({current_exc}).")
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
        _log(f"Could not import suggest_single_disk_layout ({exc}).")
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
            _log(f"suggest_single_disk_layout failed: {exc}")
            return None
    if last_type_error is not None:
        _log(f"suggest_single_disk_layout failed: {last_type_error}")
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
        _log(f"archinstall did not recognize {device}.")
        return None
    raw = _call_suggest(suggest_fn, dev, _ext4_type())
    if raw is None:
        return None
    data = layout_to_disk_config(raw)
    if data is None:
        _log("suggest_single_disk_layout returned an unusable layout.")
    return data


def emit_layout(device: str, out_path: Path | None = None) -> int:
    """Root-only: print or write disk_config JSON. Diagnostics go to stderr."""
    layout = suggest_layout(device)
    if not layout:
        return 1
    text = json.dumps(layout, indent=2) + "\n"
    if out_path is not None:
        out_path.write_text(text, encoding="utf-8")
        os.chmod(out_path, 0o644)
        return 0
    sys.stdout.write(text)
    return 0


def _parse_json_object(text: str) -> dict | None:
    text = text.strip()
    if not text:
        return None
    try:
        data = json.loads(text)
        return data if isinstance(data, dict) else None
    except json.JSONDecodeError:
        pass
    start = text.find("{")
    end = text.rfind("}")
    if start < 0 or end <= start:
        return None
    try:
        data = json.loads(text[start : end + 1])
    except json.JSONDecodeError:
        return None
    return data if isinstance(data, dict) else None


def suggest_layout_via_sudo(device: str) -> dict | None:
    helper = Path(__file__).resolve()
    cmd = [
        "sudo",
        "-E",
        "--",
        sys.executable,
        str(helper),
        "--emit-layout",
        device,
    ]
    try:
        proc = subprocess.run(cmd, check=False, text=True, capture_output=True)
    except OSError as exc:
        _log(f"Privileged layout helper failed ({exc}).")
        return None
    if proc.stderr:
        sys.stderr.write(proc.stderr)
        if not proc.stderr.endswith("\n"):
            sys.stderr.write("\n")
    if proc.returncode != 0:
        _log("Privileged layout helper exited non-zero.")
        return None
    data = _parse_json_object(proc.stdout)
    if data is None:
        _log("Privileged layout helper returned no disk_config JSON.")
    return data


def suggest_layout_for_install(device: str) -> dict | None:
    """archinstall disk helpers recurse/fail as live; generate layout as root."""
    if os.geteuid() == 0:
        return suggest_layout(device)
    return suggest_layout_via_sudo(device)


def find_post_script() -> Path | None:
    here = Path(__file__).resolve()
    for path in (
        Path("/usr/local/lib/codalinux/coda-install-post.sh"),
        here.parent / "coda-install-post.sh",
        Path("/usr/share/codalinux/install/coda-install-post.sh"),
    ):
        if path.is_file():
            return path
    return None


def run_install(cmd: list[str], user: str) -> int:
    print(f"Running: {' '.join(cmd)}")
    rc = subprocess.call(cmd)
    if rc != 0:
        return rc
    post = find_post_script()
    if post is None:
        print("coda-install-post.sh missing; installed system will have no Coda desktop.", file=sys.stderr)
        return rc
    target = os.environ.get("CODA_INSTALL_TARGET", "/mnt")
    post_cmd = [str(post), "--user", user, "--target", target]
    if os.geteuid() != 0:
        post_cmd = ["sudo", "-E", "--", *post_cmd]
    print(f"Running: {' '.join(post_cmd)}")
    return subprocess.call(post_cmd)


def main(argv: list[str]) -> int:
    if len(argv) >= 3 and argv[1] == "--emit-layout":
        out_path = None
        if len(argv) >= 5 and argv[3] == "--layout-out":
            out_path = Path(argv[4])
        return emit_layout(argv[2], out_path)

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
        layout = suggest_layout_for_install(disk)
        if layout:
            cfg["disk_config"] = layout
            silent = True
            print(f"Using ext4 default layout on {disk}.")
        else:
            print("Disk menu will be shown; locale/timezone/keymap stay Bozeman defaults.")

    creds, login_user, login_password = maybe_creds_path()
    print_login_banner(login_user, login_password)
    os.environ.setdefault("CODA_INSTALL_USER", login_user)

    runtime.write_text(json.dumps(cfg, indent=2) + "\n", encoding="utf-8")
    os.chmod(runtime, 0o644)
    cmd = archinstall_cmd(runtime, extra, silent, creds)
    return run_install(cmd, login_user)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
