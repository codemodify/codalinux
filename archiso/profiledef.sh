#!/usr/bin/env bash
# shellcheck disable=SC2034
# CodaLinux archiso profile. UEFI + systemd-boot only (no BIOS/syslinux).

iso_name="codalinux"
iso_label="CODA_$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y%m)"
iso_publisher="CodaLinux <https://github.com/codemodify/codalinux>"
iso_application="CodaLinux Live/Install"
iso_version="$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y.%m.%d)"
install_dir="coda"
buildmodes=('iso')
bootmodes=('uefi.systemd-boot')
pacman_conf="pacman.conf"
airootfs_image_type="squashfs"
airootfs_image_tool_options=('-comp' 'xz' '-Xbcj' 'x86,arm64' '-b' '1M' '-Xdict-size' '1M')
file_permissions=(
  ["/etc/shadow"]="0:0:400"
  ["/root"]="0:0:750"
  ["/root/.bash_profile"]="0:0:644"
  ["/usr/local/lib/codalinux/apply-os-release.sh"]="0:0:755"
  ["/usr/local/bin/coda-install"]="0:0:755"
)
