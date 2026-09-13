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
# mkarchiso applies only these modes. Unlisted airootfs files become 644
# (coda-hyprpaper shipped non-executable once; wallpaper never started).
file_permissions=(
  ["/root"]="0:0:750"
  ["/root/.bash_profile"]="0:0:644"
  ["/root/customize_airootfs.sh"]="0:0:755"
  ["/usr/local/lib/codalinux/apply-os-release.sh"]="0:0:755"
  ["/usr/local/lib/codalinux/apply-locale.sh"]="0:0:755"
  ["/usr/local/lib/codalinux/coda-live-setup.sh"]="0:0:755"
  ["/usr/local/lib/codalinux/coda-pacman-init.sh"]="0:0:755"
  ["/usr/local/lib/codalinux/coda-install-config.py"]="0:0:755"
  ["/usr/local/lib/codalinux/coda-install-post.sh"]="0:0:755"
  ["/usr/local/bin/coda-install"]="0:0:755"
  ["/usr/local/bin/coda-hyprland"]="0:0:755"
  ["/usr/local/bin/coda-settings"]="0:0:755"
  ["/usr/local/bin/coda-sandbox"]="0:0:755"
  ["/usr/local/bin/coda-ags"]="0:0:755"
  ["/usr/local/bin/coda-hypr-ws"]="0:0:755"
  ["/usr/local/bin/coda-hyprlock"]="0:0:755"
  ["/usr/local/bin/coda-hyprpaper"]="0:0:755"
  ["/usr/local/bin/coda-wallpaper"]="0:0:755"
  ["/usr/local/bin/coda-sync-desktop-from-host"]="0:0:755"
  ["/usr/local/bin/ags"]="0:0:755"
  ["/usr/local/bin/system-config"]="0:0:755"
  ["/usr/local/bin/system-configd"]="0:0:755"
  ["/usr/local/bin/system-config-apply"]="0:0:755"
  ["/usr/local/bin/system-config-report"]="0:0:755"
  ["/usr/local/bin/system-config-tui"]="0:0:755"
  ["/usr/local/bin/system-config-gui"]="0:0:755"
  ["/usr/local/lib/codalinux/system-config-apply-launch"]="0:0:755"
  ["/usr/local/share/codalinux/system-config/guest-smoke.sh"]="0:0:755"
  ["/usr/local/share/codalinux/system-config/guest-e2e-all.sh"]="0:0:755"
)
