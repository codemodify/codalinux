"""CodaLinux archinstall profile stub.

Not imported by archinstall yet. Implement and register this (or an
equivalent --script) before treating ISO installs as supported.

Locked behavior this profile must enforce
----------------------------------------
* Bootloader: systemd-boot (UEFI only).
* Filesystem default: ext4 on /.
* Packages: official Arch core/extra from install/packages.txt.
* additional-repositories: always empty (no Coda repo, no XLibre repo).
* Display manager: greetd + sessions/wayland/codalinux-hyprland.desktop.
  Do not install SDDM/GDM/LightDM.
* Network: enable systemd-networkd, systemd-resolved, iwd.
  Do not install or enable NetworkManager.
* Firewall: do not enable firewalld/ufw.
* Branding: install branding/os-release via the pacman hook.
* Desktop: copy desktop/hypr/* (hyprland.lua + companion .conf) to ~/.config/hypr.
* NVIDIA: call scripts/hooks/nvidia.sh only after detection exists.
* AGS: do not pacman -S AUR names; optionally copy desktop/ags/ for a
  later source build.
"""

from __future__ import annotations

# TODO: subclass archinstall.default_profiles.profile.Profile (import path
# varies by archinstall version). Register as a custom desktop profile.

TODO = [
    "Load install/packages.txt and pass it to the installer package list",
    "Enable greetd.service as the display manager",
    "Install archiso/airootfs networkd + iwd units onto the target",
    "Disable NetworkManager if a parent desktop profile pulled it in",
    "Install branding hook and session desktop file",
    "Copy Hyprland companion configs into the user skel",
    "Leave CUPS and NVIDIA off unless explicitly requested",
]


def main() -> None:
    raise NotImplementedError(
        "CodaLinux archinstall profile is a stub. See install/README.md and docs/TODO.md."
    )


if __name__ == "__main__":
    main()
