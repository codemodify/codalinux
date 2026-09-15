#!/usr/bin/env python3
"""Classify a live airootfs into core (OS slot) vs desktop (coda-data).

Offline only: reads the live pacman local db and explicit path lists.
Never pacstrap. Used by coda-install / coda-slot.

Core seeds = packages/base.txt + packages/core-slot.txt, then recursive
depends from the live db. Everything else installed is desktop.
Unpackaged files (/usr/local, branding, session chrome) use explicit
prefix lists; leftover unpackaged files under /usr /etc /opt go to
desktop so Hyprland/AGS cannot silently land on the slot.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

CORE_EXTRA_PREFIXES = (
    "/usr/local/bin/coda-install",
    "/usr/local/bin/coda-slot",
    "/usr/local/lib/codalinux/coda-install",
    "/usr/local/lib/codalinux/coda-slot",
    "/usr/local/lib/codalinux/coda-desktop-mount",
    "/usr/local/lib/codalinux/apply-os-release.sh",
    "/usr/local/lib/codalinux/apply-locale.sh",
    "/usr/local/lib/codalinux/coda-pacman-init.sh",
    "/usr/local/lib/codalinux/coda-live-setup.sh",
    "/usr/local/share/codalinux/packages/",
    "/usr/local/share/codalinux/os-release",
    "/usr/local/share/codalinux/issue",
    "/usr/local/share/codalinux/issue.net",
    "/etc/pacman.d/hooks/codalinux-os-release.hook",
    "/etc/pacman.d/hooks/codalinux-locale.hook",
    "/etc/systemd/system/coda-desktop-mount.service",
    "/etc/systemd/system/coda-desktop-mount.service.d",
    "/etc/systemd/network/",
    "/etc/ssh/",
    "/etc/systemd/system/sshd.service",
    "/etc/systemd/system/qemu-guest-agent.service",
    "/etc/systemd/system/multi-user.target.wants/sshd.service",
    "/etc/systemd/system/multi-user.target.wants/qemu-guest-agent.service",
    "/etc/systemd/system/qemu-guest-agent.service.d",
    "/etc/systemd/system-preset/80-codalinux.preset",
    "/usr/lib/os-release",
    "/etc/os-release",
    "/etc/hostname",
    "/etc/locale.conf",
    "/etc/locale.gen",
    "/etc/vconsole.conf",
    "/etc/localtime",
    "/etc/machine-id",
    "/etc/coda/",
)

DESKTOP_EXTRA_PREFIXES = (
    "/usr/local/bin/coda-hyprland",
    "/usr/local/bin/coda-ags",
    "/usr/local/bin/coda-hypr",
    "/usr/local/bin/coda-wallpaper",
    "/usr/local/bin/coda-settings",
    "/usr/local/bin/coda-sandbox",
    "/usr/local/bin/coda-sync-desktop-from-host",
    "/usr/local/bin/ags",
    "/usr/local/bin/astal",
    "/usr/local/bin/system-config",
    "/usr/local/lib/hyprland/",
    "/usr/local/lib/codalinux/system-config",
    "/usr/local/share/codalinux/ags",
    "/usr/local/share/codalinux/input-help.txt",
    "/usr/local/share/ags",
    "/usr/local/share/gir-1.0",
    "/usr/local/share/glib-2.0",
    "/usr/local/share/codalinux/system-config",
    "/etc/greetd/",
    "/etc/xdg/hypr/",
    "/etc/skel/",
    "/etc/iwd/",
    "/usr/share/wayland-sessions/",
    "/usr/share/backgrounds/codalinux/",
    "/usr/share/applications/",
    "/etc/systemd/system/greetd.service",
    "/etc/systemd/system/display-manager.service",
    "/etc/systemd/system/bluetooth.service",
    "/etc/systemd/system/iwd.service",
    "/etc/systemd/user/system-configd.service",
    "/etc/systemd/user/system-config-report.service",
    "/etc/systemd/system/system-config-apply.service",
    "/etc/ld.so.conf.d/codalinux-usr-local.conf",
)

SKIP_PREFIXES = (
    "/proc/",
    "/sys/",
    "/dev/",
    "/run/",
    "/tmp/",
    "/mnt/",
    "/media/",
    "/lost+found/",
    "/boot/syslinux/",
    "/var/tmp/",
)

SKIP_EXACT = {
    "/proc",
    "/sys",
    "/dev",
    "/run",
    "/tmp",
    "/mnt",
    "/media",
    "/lost+found",
}

# /home and /var are seeded onto coda-data as whole trees, not via the
# core/desktop package split (and /var already bind-mounts from data).
DATA_TREE_PREFIXES = ("/home/", "/var/")
DATA_TREE_EXACT = {"/home", "/var"}

# filesystem ships these as symlinks (usr-merge). Pacman may list them
# as lib/ with a trailing slash; they must still be copied so kmod can
# resolve /lib/modules (mkinitcpio autodetect + add_module).
USR_MERGE_LINKS = ("/bin", "/lib", "/lib64", "/sbin")


def parse_pkg_list(text: str) -> list[str]:
    out: list[str] = []
    for raw in text.splitlines():
        line = raw.split("#", 1)[0].strip()
        if line:
            out.append(line)
    return out


def load_pkg_list(path: Path) -> list[str]:
    return parse_pkg_list(path.read_text(encoding="utf-8"))


def _dep_name(token: str) -> str:
    token = token.strip()
    if not token or token == "None":
        return ""
    for sep in (">=", "<=", "=", ">", "<"):
        if sep in token:
            token = token.split(sep, 1)[0]
            break
    return token.strip()


def parse_desc(desc_text: str) -> dict:
    name = ""
    depends: list[str] = []
    provides: list[str] = []
    section = None
    for raw in desc_text.splitlines():
        line = raw.strip()
        if line.startswith("%") and line.endswith("%"):
            section = line.strip("%").upper()
            continue
        if not line:
            continue
        if section == "NAME":
            name = line
        elif section == "DEPENDS":
            dep = _dep_name(line)
            if dep:
                depends.append(dep)
        elif section == "PROVIDES":
            prov = _dep_name(line)
            if prov:
                provides.append(prov)
    return {"name": name, "depends": depends, "provides": provides}


def parse_files(files_text: str) -> list[str]:
    out: list[str] = []
    in_files = False
    for raw in files_text.splitlines():
        line = raw.strip()
        if line == "%FILES%":
            in_files = True
            continue
        if line.startswith("%") and line.endswith("%"):
            in_files = False
            continue
        if in_files and line:
            if not line.startswith("/"):
                line = "/" + line
            out.append(line)
    return out


def read_pacman_local(local_dir: Path) -> tuple[dict[str, dict], dict[str, str]]:
    """Return (packages_by_name, provide_to_name)."""
    pkgs: dict[str, dict] = {}
    provides: dict[str, str] = {}
    if not local_dir.is_dir():
        return pkgs, provides
    for entry in local_dir.iterdir():
        if not entry.is_dir():
            continue
        desc = entry / "desc"
        files = entry / "files"
        if not desc.is_file():
            continue
        meta = parse_desc(desc.read_text(encoding="utf-8", errors="replace"))
        name = meta.get("name") or ""
        if not name:
            continue
        file_list: list[str] = []
        if files.is_file():
            file_list = parse_files(files.read_text(encoding="utf-8", errors="replace"))
        pkgs[name] = {
            "name": name,
            "depends": meta.get("depends") or [],
            "provides": meta.get("provides") or [],
            "files": file_list,
        }
        provides[name] = name
        for p in pkgs[name]["provides"]:
            provides.setdefault(p, name)
    return pkgs, provides


def expand_seeds(
    seeds: list[str], pkgs: dict[str, dict], provides: dict[str, str]
) -> set[str]:
    core: set[str] = set()
    queue = list(seeds)
    seen: set[str] = set()
    while queue:
        raw = queue.pop()
        name = provides.get(raw, raw)
        if name in seen:
            continue
        seen.add(name)
        if name not in pkgs:
            continue
        core.add(name)
        for dep in pkgs[name]["depends"]:
            resolved = provides.get(dep, dep)
            if resolved not in seen:
                queue.append(resolved)
    return core


def matches_prefix(path: str, prefixes: tuple[str, ...]) -> bool:
    for p in prefixes:
        if p.endswith("/"):
            if path == p[:-1] or path.startswith(p):
                return True
        elif path == p or path.startswith(p + "/"):
            return True
    return False


def should_skip(path: str) -> bool:
    if path in SKIP_EXACT or path in DATA_TREE_EXACT:
        return True
    if matches_prefix(path, SKIP_PREFIXES) or matches_prefix(path, DATA_TREE_PREFIXES):
        return True
    if path.startswith("/boot/"):
        # Kernels are installed onto the ESP by coda_install_boot_files.
        return True
    return False


def relpath(path: str) -> str:
    return path[1:] if path.startswith("/") else path


def is_usr_merge_link(path: str) -> bool:
    rel = relpath(path.rstrip("/") if path != "/" else path)
    return f"/{rel}" in USR_MERGE_LINKS or rel in {p[1:] for p in USR_MERGE_LINKS}


def is_directory_entry(path: str, root: Path | None = None) -> bool:
    """True for real directories only. File lists must be files/symlinks.

    rsync -a --files-from recurses a listed directory, so a core
    ``filesystem`` entry like ``usr/local/bin/`` would copy coda-hyprland.

    Trailing slash is not enough: usr-merge ``lib/`` is a symlink and
    must stay on the core list so ``/lib/modules`` resolves in chroot.
    """
    if is_usr_merge_link(path):
        return False
    rel = relpath(path.rstrip("/") if path != "/" else path)
    if root is not None:
        try:
            p = root / rel
            if p.is_symlink():
                return False
            if p.is_dir():
                return True
        except OSError:
            pass
    return path.endswith("/")


def keep_rsync_leaf(path: str, root: Path | None = None) -> bool:
    return not is_directory_entry(path, root)


def apply_desktop_priority(core_files: list[str], desktop_files: list[str]) -> tuple[list[str], list[str]]:
    """Session chrome listed in DESKTOP_EXTRA_PREFIXES never stays on core."""
    core: list[str] = []
    moved: list[str] = []
    for f in core_files:
        if matches_prefix(f, DESKTOP_EXTRA_PREFIXES):
            moved.append(f)
        else:
            core.append(f)
    return core, sorted(set(desktop_files) | set(moved))


def collect_unpackaged(
    root: Path, owned: set[str]
) -> tuple[list[str], list[str]]:
    """Walk /usr /etc /opt for files not owned by any package."""
    core_extra: list[str] = []
    desktop_extra: list[str] = []
    if not root.is_dir():
        return core_extra, desktop_extra
    walk_roots = [root / "usr", root / "etc", root / "opt"]
    for base in walk_roots:
        if not base.exists():
            continue
        for p in base.rglob("*"):
            try:
                rel = "/" + str(p.relative_to(root))
            except ValueError:
                continue
            if should_skip(rel):
                continue
            if rel in owned:
                continue
            if p.is_dir() and not p.is_symlink():
                # Directories are created by rsync when files are copied.
                continue
            if matches_prefix(rel, CORE_EXTRA_PREFIXES):
                core_extra.append(rel)
            else:
                desktop_extra.append(rel)
    return core_extra, desktop_extra


def classify(
    root: Path,
    seeds: list[str],
    pkgs: dict[str, dict],
    provides: dict[str, str],
    include_unpackaged: bool = True,
) -> dict:
    core_pkgs = sorted(expand_seeds(seeds, pkgs, provides))
    desktop_pkgs = sorted(name for name in pkgs if name not in set(core_pkgs))

    core_files: list[str] = []
    desktop_files: list[str] = []
    owned: set[str] = set()
    for name in core_pkgs:
        for f in pkgs[name]["files"]:
            if should_skip(f):
                continue
            core_files.append(f)
            owned.add(f)
    for name in desktop_pkgs:
        for f in pkgs[name]["files"]:
            if should_skip(f):
                continue
            desktop_files.append(f)
            owned.add(f)

    extra_core: list[str] = []
    extra_desktop: list[str] = []
    if include_unpackaged:
        extra_core, extra_desktop = collect_unpackaged(root, owned)
        # Explicit desktop prefixes that exist even if the walk missed them
        # (empty dirs / optional).
        for prefix in DESKTOP_EXTRA_PREFIXES:
            p = root / relpath(prefix.rstrip("/"))
            if p.exists() or p.is_symlink():
                if p.is_dir() and not p.is_symlink():
                    for child in p.rglob("*"):
                        if child.is_dir() and not child.is_symlink():
                            continue
                        rel = "/" + str(child.relative_to(root))
                        if rel not in owned and not should_skip(rel):
                            extra_desktop.append(rel)
                else:
                    rel = "/" + str(p.relative_to(root))
                    if rel not in owned and not should_skip(rel):
                        extra_desktop.append(rel)

    core_files = sorted(set(core_files + extra_core + collect_usr_merge(root)))
    desktop_files = sorted(set(desktop_files + extra_desktop) - set(core_files))
    core_files, desktop_files = apply_desktop_priority(core_files, desktop_files)
    core_files = sorted({
        (f.rstrip("/") if f != "/" and f.endswith("/") else f)
        for f in core_files
        if keep_rsync_leaf(f, root)
    })
    desktop_files = sorted({
        (f.rstrip("/") if f != "/" and f.endswith("/") else f)
        for f in desktop_files
        if keep_rsync_leaf(f, root)
    })

    return {
        "core_packages": core_pkgs,
        "desktop_packages": desktop_pkgs,
        "core_files": core_files,
        "desktop_files": desktop_files,
        "core_file_count": len(core_files),
        "desktop_file_count": len(desktop_files),
    }


def find_pacman_local(root: Path) -> Path | None:
    candidates = [
        root / "var/lib/pacman/local",
        Path("/var/lib/pacman/local"),
    ]
    for c in candidates:
        if c.is_dir() and any(c.iterdir()):
            return c
    return None


def find_seed_file(name: str) -> Path | None:
    here = Path(__file__).resolve().parent
    repo = here.parent
    candidates = [
        Path(f"/usr/local/share/codalinux/packages/{name}"),
        repo / "packages" / name,
        here.parent / "packages" / name,
    ]
    for c in candidates:
        if c.is_file():
            return c
    return None


def collect_usr_merge(root: Path) -> list[str]:
    """Always keep usr-merge compat symlinks on the core slot."""
    out: list[str] = []
    if not root.is_dir():
        return out
    for abs_path in USR_MERGE_LINKS:
        p = root / relpath(abs_path)
        try:
            if p.exists() or p.is_symlink():
                out.append(abs_path)
        except OSError:
            continue
    return out


def write_list(path: Path, files: list[str], root: Path | None = None) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as fh:
        for f in files:
            if not keep_rsync_leaf(f, root):
                continue
            fh.write(relpath(f.rstrip("/") if f != "/" else f) + "\n")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", required=True, help="live airootfs or /")
    parser.add_argument("--core-list", help="write relative core paths here")
    parser.add_argument("--desktop-list", help="write relative desktop paths here")
    parser.add_argument("--report", help="write JSON classification report")
    parser.add_argument("--seeds", action="append", default=[], help="seed list file")
    parser.add_argument(
        "--pacman-local",
        help="override path to var/lib/pacman/local",
    )
    parser.add_argument(
        "--no-unpackaged",
        action="store_true",
        help="skip walking unpackaged files (tests / faster classify)",
    )
    parser.add_argument("--json", action="store_true", help="print report JSON")
    args = parser.parse_args(argv)

    root = Path(args.root)
    if not root.is_dir():
        print(f"coda-install-split: root is not a directory: {root}", file=sys.stderr)
        return 2

    seed_files = [Path(p) for p in args.seeds]
    if not seed_files:
        for name in ("base.txt", "core-slot.txt"):
            found = find_seed_file(name)
            if found is None:
                print(f"coda-install-split: missing seed list {name}", file=sys.stderr)
                return 2
            seed_files.append(found)

    seeds: list[str] = []
    for sf in seed_files:
        seeds.extend(load_pkg_list(sf))
    if not seeds:
        print("coda-install-split: no seed packages", file=sys.stderr)
        return 2

    local = Path(args.pacman_local) if args.pacman_local else find_pacman_local(root)
    if local is None:
        print(
            "coda-install-split: pacman local db not found "
            f"(looked under {root}/var/lib/pacman/local and /var/lib/pacman/local)",
            file=sys.stderr,
        )
        return 2

    pkgs, provides = read_pacman_local(local)
    if not pkgs:
        print(f"coda-install-split: empty pacman db at {local}", file=sys.stderr)
        return 2

    result = classify(
        root,
        seeds,
        pkgs,
        provides,
        include_unpackaged=not args.no_unpackaged,
    )
    result["pacman_local"] = str(local)
    result["seed_files"] = [str(p) for p in seed_files]
    result["seeds"] = seeds

    if args.core_list:
        write_list(Path(args.core_list), result["core_files"], root)
    if args.desktop_list:
        write_list(Path(args.desktop_list), result["desktop_files"], root)
    if args.report:
        Path(args.report).write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    if args.json or not (args.core_list or args.desktop_list or args.report):
        slim = {
            "core_packages": result["core_packages"],
            "desktop_packages": result["desktop_packages"],
            "core_file_count": result["core_file_count"],
            "desktop_file_count": result["desktop_file_count"],
            "pacman_local": result["pacman_local"],
        }
        print(json.dumps(slim, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
