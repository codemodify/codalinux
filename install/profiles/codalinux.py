"""Leftover finish hook. First install is coda-install-ab.sh, not archinstall.

`coda-install` / `coda-slot` run `coda-install-post.sh` on the live ISO
against the target mount (default `/mnt`). This module is the same hook:
`python3 codalinux.py --target /mnt --user user`.
"""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path


LOCKED = [
    "Bootloader: systemd-boot (UEFI only)",
    "Filesystem default: ext4 on /",
    "Packages: official Arch core/extra from install/packages.txt",
    "additional-repositories: empty",
    "Display manager: greetd autologin → /usr/local/bin/coda-hyprland",
    "Network: systemd-networkd + systemd-resolved + iwd (not NetworkManager)",
    "Default account: user / 1 (sudo); root password 1",
]


def find_post_script() -> Path:
    here = Path(__file__).resolve()
    candidates = [
        Path("/usr/local/lib/codalinux/coda-install-post.sh"),
        here.parent.parent.parent / "scripts" / "coda-install-post.sh",
        Path("/usr/share/codalinux/install/coda-install-post.sh"),
        here.parent / "coda-install-post.sh",
    ]
    for path in candidates:
        if path.is_file():
            return path
    raise FileNotFoundError("coda-install-post.sh not found")


def apply(target: str = "/mnt", user: str = "user") -> int:
    script = find_post_script()
    cmd = [str(script), "--user", user, "--target", target]
    print(f"codalinux profile: {' '.join(cmd)}")
    return subprocess.call(cmd)


def main(argv: list[str] | None = None) -> int:
    argv = list(sys.argv[1:] if argv is None else argv)
    user = os.environ.get("CODA_INSTALL_USER", "user")
    target = os.environ.get("CODA_INSTALL_TARGET", "/mnt")
    i = 0
    while i < len(argv):
        if argv[i] in ("--user",) and i + 1 < len(argv):
            user = argv[i + 1]
            i += 2
            continue
        if argv[i] in ("--target",) and i + 1 < len(argv):
            target = argv[i + 1]
            i += 2
            continue
        if not argv[i].startswith("-"):
            target = argv[i]
            i += 1
            continue
        print(f"unknown argument: {argv[i]}", file=sys.stderr)
        return 2
    return apply(target, user)


if __name__ == "__main__":
    sys.exit(main())
