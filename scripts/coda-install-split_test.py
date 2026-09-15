#!/usr/bin/env python3
import importlib.util
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

_SPEC = importlib.util.spec_from_file_location(
    "coda_install_split",
    Path(__file__).with_name("coda-install-split.py"),
)
mod = importlib.util.module_from_spec(_SPEC)
assert _SPEC.loader is not None
_SPEC.loader.exec_module(mod)


def _write_pkg(local: Path, name: str, version: str, depends: list[str], files: list[str]) -> None:
    d = local / f"{name}-{version}"
    d.mkdir(parents=True)
    desc = ["%NAME%", name, "", "%VERSION%", version, "", "%DEPENDS%"]
    desc.extend(depends)
    desc.extend(["", "%PROVIDES%", ""])
    (d / "desc").write_text("\n".join(desc) + "\n", encoding="utf-8")
    (d / "files").write_text("%FILES%\n" + "\n".join(files) + "\n", encoding="utf-8")


class SplitTests(unittest.TestCase):
    def test_parse_pkg_list_ignores_comments(self):
        text = "# comment\nbase\n\nlinux  # kernel\n"
        self.assertEqual(mod.parse_pkg_list(text), ["base", "linux"])

    def test_dep_name_strips_version(self):
        self.assertEqual(mod._dep_name("glibc>=2.38"), "glibc")
        self.assertEqual(mod._dep_name("None"), "")

    def test_expand_seeds_follows_depends(self):
        pkgs = {
            "base": {"name": "base", "depends": ["filesystem", "glibc"], "provides": [], "files": []},
            "filesystem": {"name": "filesystem", "depends": ["glibc"], "provides": [], "files": []},
            "glibc": {"name": "glibc", "depends": [], "provides": [], "files": []},
            "hyprland": {"name": "hyprland", "depends": ["glibc"], "provides": [], "files": []},
        }
        provides = {k: k for k in pkgs}
        core = mod.expand_seeds(["base"], pkgs, provides)
        self.assertEqual(core, {"base", "filesystem", "glibc"})
        self.assertNotIn("hyprland", core)

    def test_classify_sends_hyprland_to_desktop(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            local = root / "var/lib/pacman/local"
            local.mkdir(parents=True)
            _write_pkg(
                local,
                "linux",
                "1-1",
                ["glibc"],
                ["usr/", "usr/lib/", "usr/lib/modules/", "usr/lib/modules/k/vmlinuz"],
            )
            _write_pkg(local, "glibc", "1-1", [], ["usr/", "usr/lib/", "usr/lib/libc.so.6"])
            _write_pkg(
                local,
                "hyprland",
                "1-1",
                ["glibc"],
                ["usr/", "usr/bin/", "usr/bin/Hyprland"],
            )
            (root / "usr/lib/modules/k").mkdir(parents=True)
            (root / "usr/lib/modules/k/vmlinuz").write_bytes(b"k")
            (root / "usr/lib/libc.so.6").write_bytes(b"c")
            (root / "usr/bin").mkdir(parents=True)
            (root / "usr/bin/Hyprland").write_bytes(b"h")
            (root / "usr/local/bin").mkdir(parents=True)
            (root / "usr/local/bin/coda-install").write_text("i", encoding="utf-8")
            (root / "usr/local/bin/coda-hyprland").write_text("d", encoding="utf-8")

            pkgs, provides = mod.read_pacman_local(local)
            result = mod.classify(root, ["linux"], pkgs, provides, include_unpackaged=True)
            self.assertIn("linux", result["core_packages"])
            self.assertIn("glibc", result["core_packages"])
            self.assertIn("hyprland", result["desktop_packages"])
            self.assertIn("/usr/lib/modules/k/vmlinuz", result["core_files"])
            self.assertIn("/usr/bin/Hyprland", result["desktop_files"])
            self.assertNotIn("/usr/bin/Hyprland", result["core_files"])
            self.assertIn("/usr/local/bin/coda-install", result["core_files"])
            self.assertIn("/usr/local/bin/coda-hyprland", result["desktop_files"])
            self.assertNotIn("/usr/local/bin/coda-hyprland", result["core_files"])

    def test_installer_lib_stays_on_core(self):
        """coda-install-* helpers must not follow Hyprland onto coda-data."""
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            local = root / "var/lib/pacman/local"
            local.mkdir(parents=True)
            _write_pkg(local, "filesystem", "1-1", [], ["usr/bin/bash"])
            (root / "usr/bin").mkdir(parents=True)
            (root / "usr/bin/bash").write_text("sh", encoding="utf-8")
            libdir = root / "usr/local/lib/codalinux"
            libdir.mkdir(parents=True)
            (libdir / "coda-install-lib.sh").write_text("coda_need_root() { :; }\n", encoding="utf-8")
            (libdir / "coda-desktop-mount").write_text("mount\n", encoding="utf-8")
            (root / "usr/local/bin").mkdir(parents=True)
            (root / "usr/local/bin/coda-slot").write_text("slot\n", encoding="utf-8")
            (root / "usr/local/bin/coda-hyprland").write_text("hypr\n", encoding="utf-8")

            pkgs, provides = mod.read_pacman_local(local)
            result = mod.classify(root, ["filesystem"], pkgs, provides, include_unpackaged=True)
            self.assertIn("/usr/local/lib/codalinux/coda-install-lib.sh", result["core_files"])
            self.assertNotIn("/usr/local/lib/codalinux/coda-install-lib.sh", result["desktop_files"])
            self.assertIn("/usr/local/bin/coda-slot", result["core_files"])
            self.assertIn("/usr/local/lib/codalinux/coda-desktop-mount", result["core_files"])
            self.assertNotIn("/usr/local/bin/coda-hyprland", result["core_files"])

    def test_home_var_boot_skipped(self):
        self.assertTrue(mod.should_skip("/var/lib/pacman/local"))
        self.assertTrue(mod.should_skip("/home/live"))
        self.assertTrue(mod.should_skip("/boot/vmlinuz-linux"))
        self.assertFalse(mod.should_skip("/usr/bin/bash"))

    def test_filesystem_dir_nodes_do_not_claim_coda_hyprland(self):
        """filesystem owns /usr/local/bin/; that must not list coda-hyprland on core."""
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            local = root / "var/lib/pacman/local"
            local.mkdir(parents=True)
            _write_pkg(
                local,
                "filesystem",
                "1-1",
                [],
                [
                    "usr/",
                    "usr/local/",
                    "usr/local/bin/",
                    "usr/local/lib/",
                    "usr/bin/",
                    "usr/bin/bash",
                ],
            )
            (root / "usr/local/bin").mkdir(parents=True)
            (root / "usr/local/lib").mkdir(parents=True)
            (root / "usr/bin").mkdir(parents=True)
            (root / "usr/bin/bash").write_text("sh", encoding="utf-8")
            (root / "usr/local/bin/coda-install").write_text("i", encoding="utf-8")
            (root / "usr/local/bin/coda-hyprland").write_text("d", encoding="utf-8")
            (root / "usr/local/bin/coda-ags").write_text("a", encoding="utf-8")

            pkgs, provides = mod.read_pacman_local(local)
            result = mod.classify(root, ["filesystem"], pkgs, provides, include_unpackaged=True)
            for path in result["core_files"]:
                self.assertFalse(
                    path.endswith("/"),
                    f"core list must not include directory {path!r} (rsync -a would recurse)",
                )
            self.assertIn("/usr/bin/bash", result["core_files"])
            self.assertIn("/usr/local/bin/coda-install", result["core_files"])
            self.assertIn("/usr/local/bin/coda-hyprland", result["desktop_files"])
            self.assertNotIn("/usr/local/bin/coda-hyprland", result["core_files"])
            self.assertNotIn("/usr/local/bin/", result["core_files"])
            self.assertNotIn("/usr/local/", result["core_files"])

            core_list = root / "core.list"
            mod.write_list(core_list, result["core_files"], root)
            listed = core_list.read_text(encoding="utf-8").splitlines()
            self.assertNotIn("usr/local/bin/", listed)
            self.assertNotIn("usr/local/bin/coda-hyprland", listed)
            self.assertIn("usr/local/bin/coda-install", listed)

            if shutil.which("rsync") is None:
                return
            dest = root / "slot"
            dest.mkdir()
            subprocess.run(
                [
                    "rsync",
                    "-lptgoDHAX",
                    "--files-from",
                    str(core_list),
                    f"{root}/",
                    f"{dest}/",
                ],
                check=True,
            )
            self.assertTrue((dest / "usr/local/bin/coda-install").is_file())
            self.assertFalse(
                (dest / "usr/local/bin/coda-hyprland").exists(),
                "core rsync must not copy coda-hyprland onto the slot",
            )
            self.assertFalse((dest / "usr/local/bin/coda-ags").exists())

    def test_usr_merge_and_modules_stay_on_core(self):
        """Trailing-slash usr-merge links + .ko files must rsync onto the slot."""
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            local = root / "var/lib/pacman/local"
            local.mkdir(parents=True)
            _write_pkg(
                local,
                "filesystem",
                "1-1",
                [],
                [
                    "bin/",
                    "lib/",
                    "lib64/",
                    "sbin/",
                    "usr/",
                    "usr/bin/",
                    "usr/lib/",
                    "usr/sbin/",
                    "usr/bin/bash",
                ],
            )
            _write_pkg(
                local,
                "linux",
                "1-1",
                ["filesystem"],
                [
                    "usr/lib/modules/",
                    "usr/lib/modules/k/",
                    "usr/lib/modules/k/vmlinuz",
                    "usr/lib/modules/k/modules.dep",
                    "usr/lib/modules/k/kernel/fs/fat/vfat.ko.zst",
                    "usr/lib/modules/k/kernel/fs/fat/fat.ko.zst",
                    "usr/lib/modules/k/kernel/fs/ext4/ext4.ko.zst",
                    "usr/lib/modules/k/kernel/drivers/virtio/virtio_blk.ko.zst",
                ],
            )
            (root / "usr/bin").mkdir(parents=True)
            (root / "usr/lib").mkdir(parents=True)
            (root / "usr/sbin").mkdir(parents=True)
            (root / "bin").symlink_to("usr/bin")
            (root / "lib").symlink_to("usr/lib")
            (root / "lib64").symlink_to("usr/lib")
            (root / "sbin").symlink_to("usr/sbin")
            (root / "usr/bin/bash").write_text("sh", encoding="utf-8")
            moddir = root / "usr/lib/modules/k/kernel"
            (moddir / "fs/fat").mkdir(parents=True)
            (moddir / "fs/ext4").mkdir(parents=True)
            (moddir / "drivers/virtio").mkdir(parents=True)
            (root / "usr/lib/modules/k/vmlinuz").write_bytes(b"k")
            (root / "usr/lib/modules/k/modules.dep").write_text("vfat:\n", encoding="utf-8")
            (moddir / "fs/fat/vfat.ko.zst").write_bytes(b"m")
            (moddir / "fs/fat/fat.ko.zst").write_bytes(b"m")
            (moddir / "fs/ext4/ext4.ko.zst").write_bytes(b"m")
            (moddir / "drivers/virtio/virtio_blk.ko.zst").write_bytes(b"m")

            self.assertFalse(mod.is_directory_entry("/lib/", root))
            self.assertFalse(mod.is_directory_entry("lib/", root))
            self.assertTrue(mod.is_directory_entry("/usr/lib/", root))

            pkgs, provides = mod.read_pacman_local(local)
            result = mod.classify(root, ["linux"], pkgs, provides, include_unpackaged=False)
            for link in ("/bin", "/lib", "/lib64", "/sbin"):
                self.assertIn(link, result["core_files"])
            self.assertIn("/usr/lib/modules/k/kernel/fs/fat/vfat.ko.zst", result["core_files"])
            self.assertIn("/usr/lib/modules/k/modules.dep", result["core_files"])
            self.assertNotIn("/lib/", result["core_files"])

            core_list = root / "core.list"
            mod.write_list(core_list, result["core_files"], root)
            listed = core_list.read_text(encoding="utf-8").splitlines()
            self.assertIn("lib", listed)
            self.assertNotIn("lib/", listed)
            self.assertIn("usr/lib/modules/k/kernel/fs/fat/vfat.ko.zst", listed)

            if shutil.which("rsync") is None:
                return
            dest = root / "slot"
            dest.mkdir()
            subprocess.run(
                [
                    "rsync",
                    "-lptgoDHAX",
                    "--files-from",
                    str(core_list),
                    f"{root}/",
                    f"{dest}/",
                ],
                check=True,
            )
            self.assertTrue((dest / "lib").is_symlink())
            self.assertEqual((dest / "lib").readlink().as_posix(), "usr/lib")
            self.assertTrue((dest / "usr/lib/modules/k/kernel/fs/fat/vfat.ko.zst").is_file())
            self.assertTrue((dest / "usr/lib/modules/k/modules.dep").is_file())

    def test_desktop_priority_moves_session_wrapper(self):
        core, desktop = mod.apply_desktop_priority(
            ["/usr/local/bin/coda-hyprland", "/usr/bin/bash"],
            ["/usr/bin/Hyprland"],
        )
        self.assertEqual(core, ["/usr/bin/bash"])
        self.assertIn("/usr/local/bin/coda-hyprland", desktop)


if __name__ == "__main__":
    unittest.main()
