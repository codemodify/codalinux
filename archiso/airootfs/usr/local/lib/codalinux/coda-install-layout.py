#!/usr/bin/env python3
"""CodaLinux ESP + OS-A + OS-B + data size planner.

Disk sizes are in bytes. Output sizes are in whole mebibytes so sgdisk
can consume them as `+NMiB`.
"""
from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys

# GPT + alignment slack reserved at the end of the disk.
GPT_SLACK_MIB = 4

# Target picture from architecture.md / DESIGN.md.
# v1 copies the full live desktop into each slot (not a ~1.1G core-only
# image). 8 GiB is the implemented floor; 4 GiB remains the later
# core-only target and is too small for this payload.
ESP_MIB = 1024
SLOT_FLOOR_MIB = 8192
SLOT_PREFERRED_MIB = 8192
DATA_FLOOR_MIB = 4096
CORE_ONLY_SLOT_FLOOR_MIB = 4096

# 1 + 8 + 8 + 4 + slack. Documented v1 installer minimum (~21 GiB).
MINIMUM_DISK_MIB = ESP_MIB + (2 * SLOT_FLOOR_MIB) + DATA_FLOOR_MIB + GPT_SLACK_MIB
# Later core-only picture (not this v1 payload).
CORE_ONLY_MINIMUM_DISK_MIB = (
    ESP_MIB + (2 * CORE_ONLY_SLOT_FLOOR_MIB) + DATA_FLOOR_MIB + GPT_SLACK_MIB
)


def mib(n: int) -> int:
    return n * 1024 * 1024


def format_mib(n: int) -> str:
    if n >= 1024 and n % 1024 == 0:
        return f"{n // 1024} GiB"
    if n >= 1024:
        return f"{n / 1024:.1f} GiB"
    return f"{n} MiB"


def plan_layout(disk_bytes: int) -> dict | None:
    """Return a layout plan or None if the disk is too small."""
    if disk_bytes < 0:
        return None
    disk_mib = disk_bytes // (1024 * 1024)
    if disk_mib < MINIMUM_DISK_MIB:
        return None

    usable = disk_mib - ESP_MIB - GPT_SLACK_MIB
    # Prefer 8G slots when leftover data still meets the floor.
    if usable >= (2 * SLOT_PREFERRED_MIB) + DATA_FLOOR_MIB:
        slot_mib = SLOT_PREFERRED_MIB
    else:
        slot_mib = SLOT_FLOOR_MIB
    data_mib = usable - (2 * slot_mib)
    if data_mib < DATA_FLOOR_MIB:
        return None

    return {
        "disk_mib": disk_mib,
        "esp_mib": ESP_MIB,
        "slot_a_mib": slot_mib,
        "slot_b_mib": slot_mib,
        "data_mib": data_mib,
        "slack_mib": GPT_SLACK_MIB,
        "minimum_disk_mib": MINIMUM_DISK_MIB,
        "slot_floor_mib": SLOT_FLOOR_MIB,
        "slot_preferred_mib": SLOT_PREFERRED_MIB,
        "data_floor_mib": DATA_FLOOR_MIB,
    }


def explain_too_small(disk_bytes: int) -> str:
    have = disk_bytes // (1024 * 1024)
    return (
        f"disk is too small for ESP + OS-A + OS-B + data: "
        f"have {format_mib(have)} ({have} MiB), "
        f"need at least {format_mib(MINIMUM_DISK_MIB)} "
        f"({MINIMUM_DISK_MIB} MiB = {ESP_MIB} MiB ESP + "
        f"{SLOT_FLOOR_MIB} MiB OS-A + {SLOT_FLOOR_MIB} MiB OS-B + "
        f"{DATA_FLOOR_MIB} MiB data + {GPT_SLACK_MIB} MiB GPT slack). "
        f"v1 slots hold the full live desktop (not a 4 GiB core-only image). "
        f"Recommend 32 GiB for QEMU."
    )


def disk_size_bytes(path: str) -> int:
    return int(
        subprocess.check_output(["blockdev", "--getsize64", path], text=True).strip()
    )


def list_disks() -> list[dict]:
    raw = subprocess.check_output(
        ["lsblk", "-J", "-b", "-o", "NAME,PATH,SIZE,TYPE,MODEL,TRAN,RM"],
        text=True,
    )
    data = json.loads(raw)
    out = []
    for dev in data.get("blockdevices", []):
        if dev.get("type") != "disk":
            continue
        out.append(
            {
                "name": dev.get("name"),
                "path": dev.get("path") or f"/dev/{dev.get('name')}",
                "size": int(dev.get("size") or 0),
                "model": (dev.get("model") or "").strip(),
                "tran": dev.get("tran") or "",
                "rm": bool(dev.get("rm")),
            }
        )
    return out


def print_plan(plan: dict, disk: str | None = None) -> None:
    prefix = f"{disk}: " if disk else ""
    print(f"{prefix}ESP     {format_mib(plan['esp_mib'])}  FAT32  PARTLABEL=coda-esp  /boot")
    print(f"{prefix}OS-A    {format_mib(plan['slot_a_mib'])}  ext4   PARTLABEL=coda-a    /  (first install)")
    print(f"{prefix}OS-B    {format_mib(plan['slot_b_mib'])}  ext4   PARTLABEL=coda-b    inactive")
    print(f"{prefix}data    {format_mib(plan['data_mib'])}  ext4   PARTLABEL=coda-data /coda/data + bind /home /var")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="cmd", required=True)

    p_plan = sub.add_parser("plan", help="print a layout for a disk or size")
    p_plan.add_argument("target", help="block device or integer bytes/MiB/GiB")
    p_plan.add_argument("--json", action="store_true")

    p_check = sub.add_parser("check", help="exit 0 if target is large enough")
    p_check.add_argument("target")

    sub.add_parser("list", help="list disks (lsblk)")
    sub.add_parser("minimums", help="print documented size floors")

    args = parser.parse_args(argv)

    if args.cmd == "minimums":
        print(f"minimum_disk_mib={MINIMUM_DISK_MIB}")
        print(f"core_only_minimum_disk_mib={CORE_ONLY_MINIMUM_DISK_MIB}")
        print(f"esp_mib={ESP_MIB}")
        print(f"slot_floor_mib={SLOT_FLOOR_MIB}")
        print(f"slot_preferred_mib={SLOT_PREFERRED_MIB}")
        print(f"data_floor_mib={DATA_FLOOR_MIB}")
        print(f"recommend_qemu=32G")
        return 0

    if args.cmd == "list":
        if shutil.which("lsblk") is None:
            print("lsblk not found", file=sys.stderr)
            return 1
        for d in list_disks():
            print(f"{d['path']}\t{d['size']}\t{d['model']}")
        return 0

    target = args.target
    if target.startswith("/dev/"):
        size = disk_size_bytes(target)
        disk = target
    elif target.endswith("G") or target.endswith("g"):
        size = int(float(target[:-1]) * 1024 * 1024 * 1024)
        disk = None
    elif target.endswith("M") or target.endswith("m"):
        size = int(float(target[:-1]) * 1024 * 1024)
        disk = None
    else:
        size = int(target)
        disk = None

    plan = plan_layout(size)
    if args.cmd == "check":
        if plan is None:
            print(explain_too_small(size), file=sys.stderr)
            return 1
        print_plan(plan, disk)
        return 0

    if plan is None:
        print(explain_too_small(size), file=sys.stderr)
        return 1
    if args.json:
        print(json.dumps(plan, indent=2))
    else:
        print_plan(plan, disk)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
