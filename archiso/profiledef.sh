#!/usr/bin/env bash
# shellcheck disable=SC2034
# CodaLinux archiso profile stub. Values follow official profiledef.sh
# conventions; bootmodes are UEFI + systemd-boot only (no BIOS/syslinux).

iso_name="codalinux"
iso_label="CODA_$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y%m)"
iso_publisher="CodaLinux <https://github.com/codemodify/codalinux>"
iso_application="CodaLinux Live/Install"
iso_version="$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y.%m.%d)"
install_dir="coda"
buildmodes=('iso')

# Current archiso combined mode. Older mkarchiso wants:
#   uefi-x64.systemd-boot.esp uefi-x64.systemd-boot.eltorito
# Do not add bios.syslinux.*.
bootmodes=('uefi.systemd-boot')

# Leave unset to let mkarchiso fill uname -m on recent archiso; x86_64 is the v1 target.
arch="x86_64"
pacman_conf="pacman.conf"
airootfs_image_type="squashfs"
airootfs_image_tool_options=('-comp' 'xz' '-Xbcj' 'x86' '-b' '1M' '-Xdict-size' '1M')
file_permissions=(
  ["/etc/shadow"]="0:0:400"
  ["/etc/gshadow"]="0:0:400"
  ["/root"]="0:0:750"
  ["/usr/local/lib/codalinux/apply-os-release.sh"]="0:0:755"
)
