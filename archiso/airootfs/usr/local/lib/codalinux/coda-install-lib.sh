#!/usr/bin/env bash
# Shared helpers for coda-install / coda-slot. Sourced, not executed.
# Offline payload = live airootfs split (core → slot, desktop → coda-data).
# Never pacstrap.

if [[ -n "${CODA_INSTALL_LIB_SOURCED:-}" ]]; then
  return 0
fi
CODA_INSTALL_LIB_SOURCED=1

coda_log() { printf '%s: %s\n' "${CODA_INSTALL_LOG_PREFIX:-coda-install}" "$*"; }
coda_die() { printf '%s: %s\n' "${CODA_INSTALL_LOG_PREFIX:-coda-install}" "$*" >&2; exit 1; }

coda_need_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    coda_die "must run as root (live ISO: sudo -E; QGA guest-exec is already root)"
  fi
}

coda_part_path() {
  local disk="$1" n="$2"
  if [[ "${disk}" =~ [0-9]$ ]]; then
    printf '%s' "${disk}p${n}"
  else
    printf '%s' "${disk}${n}"
  fi
}

coda_list_disks() {
  lsblk -dnpo NAME,SIZE,TYPE,MODEL 2>/dev/null | awk '
    $3 == "disk" {
      name = $1
      base = name
      sub(".*/", "", base)
      if (base ~ /^(loop|sr|ram|zram|dm-|md)/) next
      print $0
    }
  '
}

coda_first_disk() {
  local line name
  while IFS= read -r line; do
    name="${line%% *}"
    if [[ -b "${name}" ]]; then
      printf '%s' "${name}"
      return 0
    fi
  done < <(coda_list_disks)
  return 1
}

coda_resolve_disk() {
  local preset="${CODA_INSTALL_DISK:-}"
  case "${preset}" in
    auto|first|AUTO|FIRST)
      coda_first_disk || coda_die "no usable disk found (lsblk)"
      ;;
    "")
      return 1
      ;;
    *)
      if [[ ! -b "${preset}" ]]; then
        coda_die "CODA_INSTALL_DISK=${preset} is not a block device"
      fi
      printf '%s' "${preset}"
      ;;
  esac
}

coda_by_partlabel() {
  local label="$1"
  if [[ -e "/dev/disk/by-partlabel/${label}" ]]; then
    readlink -f "/dev/disk/by-partlabel/${label}"
    return 0
  fi
  return 1
}

coda_wait_partlabel() {
  local label="$1" n=0
  while [[ "${n}" -lt 50 ]]; do
    if coda_by_partlabel "${label}" >/dev/null; then
      coda_by_partlabel "${label}"
      return 0
    fi
    sleep 0.2
    n=$((n + 1))
    partprobe >/dev/null 2>&1 || true
    udevadm settle >/dev/null 2>&1 || true
  done
  coda_die "partition PARTLABEL=${label} did not appear"
}

coda_layout_bin() {
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  for p in \
    "${CODA_INSTALL_LAYOUT:-}" \
    /usr/local/lib/codalinux/coda-install-layout.py \
    "${here}/coda-install-layout.py"
  do
    [[ -n "${p}" && -f "${p}" ]] && { printf '%s' "${p}"; return 0; }
  done
  coda_die "coda-install-layout.py not found"
}

coda_plan_json() {
  local disk="$1"
  python3 "$(coda_layout_bin)" plan "${disk}" --json
}

coda_find_source() {
  if [[ -n "${CODA_INSTALL_SOURCE:-}" ]]; then
    [[ -d "${CODA_INSTALL_SOURCE}" ]] || coda_die "CODA_INSTALL_SOURCE is not a directory"
    printf '%s' "${CODA_INSTALL_SOURCE}"
    return 0
  fi
  if [[ -d /run/archiso/airootfs/usr ]]; then
    printf '%s' /run/archiso/airootfs
    return 0
  fi
  if [[ -d /run/archiso/airootfs ]]; then
    printf '%s' /run/archiso/airootfs
    return 0
  fi
  printf '%s' /
}

coda_source_bytes() {
  local src="$1"
  du -sb --exclude=/proc --exclude=/sys --exclude=/dev --exclude=/run \
    --exclude=/tmp --exclude=/mnt --exclude=/media --exclude=/lost+found \
    "${src}" 2>/dev/null | awk '{print $1}'
}

coda_rsync_root() {
  local src="$1" dest="$2"
  local exclude_home_var="${3:-0}"
  [[ -d "${src}" ]] || coda_die "source root missing: ${src}"
  mkdir -p "${dest}"
  local -a args=(
    -aHAX
    --numeric-ids
    --info=stats1
    --exclude=/proc
    --exclude=/sys
    --exclude=/dev
    --exclude=/run
    --exclude=/tmp
    --exclude=/mnt
    --exclude=/media
    --exclude=/lost+found
    --exclude=/boot/syslinux
    --exclude=/var/tmp
  )
  if [[ "${exclude_home_var}" == 1 ]]; then
    args+=(--exclude=/home --exclude=/var)
  fi
  # Never copy an in-progress install mount back into itself.
  if [[ "${src}" == / ]]; then
    args+=(--exclude="${dest}")
  fi
  coda_log "offline copy ${src} → ${dest} (no pacstrap, no mirrors)"
  rsync "${args[@]}" "${src}/" "${dest}/"
}

coda_find_split() {
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  for p in \
    "${CODA_INSTALL_SPLIT:-}" \
    /usr/local/lib/codalinux/coda-install-split.py \
    "${here}/coda-install-split.py"
  do
    [[ -n "${p}" && -f "${p}" ]] && { printf '%s' "${p}"; return 0; }
  done
  coda_die "coda-install-split.py not found"
}

coda_find_desktop_mount() {
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  for p in \
    "${CODA_DESKTOP_MOUNT:-}" \
    /usr/local/lib/codalinux/coda-desktop-mount \
    "${here}/coda-desktop-mount"
  do
    [[ -n "${p}" && -x "${p}" ]] && { printf '%s' "${p}"; return 0; }
  done
  return 1
}

coda_session_wrapper_names() {
  printf '%s\n' \
    coda-hyprland coda-ags coda-hypr-ws coda-hyprlock coda-hyprpaper \
    coda-wallpaper coda-settings coda-sandbox coda-sync-desktop-from-host \
    ags astal \
    system-config system-configd system-config-apply \
    system-config-report system-config-tui system-config-gui
}

coda_rsync_filelist() {
  local src="$1" dest="$2" list="$3" delete="${4:-0}" kind="${5:-}"
  [[ -d "${src}" ]] || coda_die "source root missing: ${src}"
  [[ -f "${list}" ]] || coda_die "file list missing: ${list}"
  mkdir -p "${dest}"
  # -a includes -r; a listed directory would recurse (filesystem owns
  # /usr/local/bin/). Lists are files/symlinks only; drop -r anyway.
  local -a args=(
    -lptgoDHAX
    --numeric-ids
    --info=stats1
    --files-from="${list}"
    --exclude=/proc
    --exclude=/sys
    --exclude=/dev
    --exclude=/run
    --exclude=/tmp
    --exclude=/mnt
    --exclude=/media
    --exclude=/lost+found
    --exclude=/boot
    --exclude=/home
    --exclude=/var
  )
  if [[ "${delete}" == 1 ]]; then
    args+=(--delete)
  fi
  if [[ "${kind}" == core ]]; then
    local wrap
    while IFS= read -r wrap; do
      args+=(--exclude="usr/local/bin/${wrap}")
    done < <(coda_session_wrapper_names)
    args+=(--exclude=usr/local/lib/codalinux/system-config-gui)
    args+=(--exclude=usr/bin/Hyprland)
    args+=(--exclude=usr/bin/hyprland)
  fi
  if [[ "${src}" == / ]]; then
    args+=(--exclude="${dest}")
  fi
  rsync "${args[@]}" "${src}/" "${dest}/"
}

coda_purge_leaked_desktop() {
  local dest="$1" desktop_list="$2"
  local rel wrap
  [[ -d "${dest}" ]] || return 0
  if [[ -f "${desktop_list}" ]]; then
    while IFS= read -r rel; do
      [[ -n "${rel}" ]] || continue
      if [[ -e "${dest}/${rel}" || -L "${dest}/${rel}" ]]; then
        if [[ -d "${dest}/${rel}" && ! -L "${dest}/${rel}" ]]; then
          continue
        fi
        rm -f "${dest}/${rel}"
      fi
    done <"${desktop_list}"
  fi
  while IFS= read -r wrap; do
    rm -f "${dest}/usr/local/bin/${wrap}"
  done < <(coda_session_wrapper_names)
  rm -f "${dest}/usr/local/lib/codalinux/system-config-gui"
  rm -f "${dest}/usr/bin/Hyprland" "${dest}/usr/bin/hyprland"
}

coda_rsync_var_home() {
  local src="$1" data="$2"
  mkdir -p "${data}/home" "${data}/var"
  if [[ -d "${src}/var" ]]; then
    rsync -aHAX --numeric-ids \
      --exclude=/var/tmp \
      "${src}/var/" "${data}/var/"
  fi
  if [[ -d "${src}/home" ]]; then
    rsync -aHAX --numeric-ids "${src}/home/" "${data}/home/"
  fi
}

coda_write_extension_release() {
  local desktop="$1"
  mkdir -p "${desktop}/usr/lib/extension-release.d" \
    "${desktop}/etc/extension-release.d"
  cat >"${desktop}/usr/lib/extension-release.d/extension-release.coda-desktop" <<'EOF'
ID=_any
ARCHITECTURE=x86-64
EOF
  cat >"${desktop}/etc/extension-release.d/extension-release.coda-desktop" <<'EOF'
ID=_any
ARCHITECTURE=x86-64
EOF
  mkdir -p "${desktop}"
  printf 'coda-desktop\n' >"${desktop}/.coda-desktop-payload"
}

coda_install_desktop_mount_unit() {
  local dest="$1"
  local src_unit src_bin
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  src_bin="$(coda_find_desktop_mount || true)"
  src_unit=""
  for p in \
    /usr/local/lib/codalinux/coda-desktop-mount.service \
    "${here}/coda-desktop-mount.service"
  do
    [[ -f "${p}" ]] && { src_unit="${p}"; break; }
  done
  mkdir -p "${dest}/usr/local/lib/codalinux" "${dest}/etc/systemd/system"
  if [[ -n "${src_bin}" ]]; then
    install -m 0755 "${src_bin}" "${dest}/usr/local/lib/codalinux/coda-desktop-mount"
  fi
  if [[ -n "${src_unit}" ]]; then
    install -m 0644 "${src_unit}" "${dest}/etc/systemd/system/coda-desktop-mount.service"
  else
    cat >"${dest}/etc/systemd/system/coda-desktop-mount.service" <<'EOF'
[Unit]
Description=Merge CodaLinux desktop from coda-data onto the core slot
DefaultDependencies=no
After=local-fs.target coda-data.mount
Wants=coda-data.mount
Before=display-manager.service greetd.service graphical.target
ConditionPathIsDirectory=/coda/data

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/lib/codalinux/coda-desktop-mount
SuccessExitStatus=0

[Install]
WantedBy=multi-user.target
WantedBy=graphical.target
EOF
  fi
  mkdir -p "${dest}/etc/systemd/system/multi-user.target.wants" \
    "${dest}/etc/systemd/system/graphical.target.wants"
  ln -sfn /etc/systemd/system/coda-desktop-mount.service \
    "${dest}/etc/systemd/system/multi-user.target.wants/coda-desktop-mount.service"
  ln -sfn /etc/systemd/system/coda-desktop-mount.service \
    "${dest}/etc/systemd/system/graphical.target.wants/coda-desktop-mount.service"
}

coda_split_offline() {
  # Classify src and rsync core → dest, desktop → data/desktop.
  # 4th arg: 1 (default) seed /home+/var onto data (first install);
  #          0 skip (coda-slot must not replace user home).
  local src="$1" dest="$2" data="$3" seed_home_var="${4:-1}"
  local work lists core_list desktop_list report
  work="$(mktemp -d /tmp/coda-split.XXXXXX)"
  core_list="${work}/core.list"
  desktop_list="${work}/desktop.list"
  report="${work}/report.json"
  coda_log "classifying live airootfs (core → slot, desktop → coda-data)"
  python3 "$(coda_find_split)" \
    --root "${src}" \
    --core-list "${core_list}" \
    --desktop-list "${desktop_list}" \
    --report "${report}" \
    || coda_die "coda-install-split.py failed (need live pacman db)"
  local core_n desktop_n
  core_n="$(wc -l <"${core_list}" | tr -d ' ')"
  desktop_n="$(wc -l <"${desktop_list}" | tr -d ' ')"
  coda_log "split lists: ${core_n} core files, ${desktop_n} desktop files"
  if [[ "${core_n}" -lt 50 ]]; then
    coda_die "core file list is too small (${core_n}); refusing empty slot"
  fi
  if [[ "${desktop_n}" -lt 10 ]]; then
    coda_die "desktop file list is too small (${desktop_n}); Hyprland would be missing from data"
  fi
  mkdir -p "${dest}" "${data}/desktop"
  coda_log "offline core ${src} → ${dest}"
  coda_rsync_filelist "${src}" "${dest}" "${core_list}" 0 core
  coda_purge_leaked_desktop "${dest}" "${desktop_list}"
  coda_log "offline desktop ${src} → ${data}/desktop"
  coda_rsync_filelist "${src}" "${data}/desktop" "${desktop_list}" 1
  if [[ "${seed_home_var}" == 1 ]]; then
    coda_rsync_var_home "${src}" "${data}"
  fi
  coda_write_extension_release "${data}/desktop"
  mkdir -p "${dest}/etc/coda"
  printf '%s\n' /coda/data/desktop >"${dest}/etc/coda/desktop-root"
  cp -a "${report}" "${dest}/etc/coda/split.json" 2>/dev/null || true
  mkdir -p "${dest}/home" "${dest}/var"
  coda_install_desktop_mount_unit "${dest}"
  rm -rf "${work}"
}

coda_wipe_live_bits() {
  local dest="$1"
  rm -f "${dest}/etc/mkinitcpio.conf.d/archiso.conf"
  rm -f "${dest}/etc/systemd/system/coda-live-setup.service"
  rm -f "${dest}/etc/systemd/system/multi-user.target.wants/coda-live-setup.service"
  rm -f "${dest}/etc/systemd/system/getty@tty2.service.d/autologin.conf"
  rm -rf "${dest}/etc/systemd/system/getty@tty2.service.d"
  rm -f "${dest}/etc/machine-id"
  rm -f "${dest}/var/lib/dbus/machine-id"
}

coda_write_mkinitcpio() {
  local dest="$1"
  cat >"${dest}/etc/mkinitcpio.conf" <<'EOF'
# CodaLinux installed slot — stock hooks, no archiso.
MODULES=()
BINARIES=()
FILES=()
HOOKS=(base udev autodetect microcode modconf kms keyboard keymap consolefont block filesystems fsck)
COMPRESSION="zstd"
EOF
}

coda_write_fstab() {
  local dest="$1" slot="${2:-a}"
  cat >"${dest}/etc/fstab" <<EOF
# CodaLinux A/B + data. Labels are GPT PARTLABEL values.
PARTLABEL=coda-${slot}     /           ext4    defaults,noatime  0 1
PARTLABEL=coda-esp   /boot       vfat    umask=0077        0 2
PARTLABEL=coda-data  /coda/data  ext4    defaults,noatime  0 2
/coda/data/home      /home       none    bind              0 0
/coda/data/var       /var        none    bind              0 0
# Desktop payload is /coda/data/desktop (not a fstab bind).
# coda-desktop-mount.service overlays it onto /usr and /etc after local-fs.
EOF
}

coda_write_slot_marker() {
  local dest="$1" slot="$2" empty="${3:-0}"
  mkdir -p "${dest}/etc/coda"
  printf '%s\n' "${slot}" >"${dest}/etc/coda/slot"
  if [[ "${empty}" == 1 ]]; then
    printf '1\n' >"${dest}/etc/coda/empty"
  else
    rm -f "${dest}/etc/coda/empty"
  fi
}

coda_bozeman() {
  local dest="$1"
  ln -sfn /usr/share/zoneinfo/America/Denver "${dest}/etc/localtime"
  printf 'LANG=en_US.UTF-8\n' >"${dest}/etc/locale.conf"
  printf 'KEYMAP=us\n' >"${dest}/etc/vconsole.conf"
  if [[ -f "${dest}/etc/locale.gen" ]]; then
    sed -i 's/^#en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' "${dest}/etc/locale.gen" || true
  fi
  printf 'coda\n' >"${dest}/etc/hostname"
  mkdir -p "${dest}/etc/sudoers.d"
  printf '%%wheel ALL=(ALL:ALL) ALL\n' >"${dest}/etc/sudoers.d/wheel"
  chmod 0440 "${dest}/etc/sudoers.d/wheel"
}

coda_chroot() {
  local dest="$1"
  shift
  if command -v arch-chroot >/dev/null 2>&1; then
    arch-chroot -S "${dest}" "$@"
  else
    chroot "${dest}" "$@"
  fi
}

coda_bind_dev() {
  local dest="$1"
  mkdir -p "${dest}/proc" "${dest}/sys" "${dest}/dev" "${dest}/run" "${dest}/tmp"
  mount --bind /proc "${dest}/proc"
  mount --bind /sys "${dest}/sys"
  mount --bind /dev "${dest}/dev"
  if [[ -d /run ]]; then
    mount --bind /run "${dest}/run" 2>/dev/null || true
  fi
}

coda_unbind_dev() {
  local dest="$1"
  umount "${dest}/run" 2>/dev/null || true
  umount "${dest}/dev" 2>/dev/null || true
  umount "${dest}/sys" 2>/dev/null || true
  umount "${dest}/proc" 2>/dev/null || true
}

coda_find_kernel() {
  local dest="$1"
  local p
  for p in "${dest}"/usr/lib/modules/*/vmlinuz; do
    [[ -f "${p}" ]] || continue
    printf '%s' "${p}"
    return 0
  done
  for p in \
    /usr/lib/modules/*/vmlinuz \
    /run/archiso/bootmnt/coda/boot/x86_64/vmlinuz-linux \
    /run/archiso/bootmnt/*/boot/x86_64/vmlinuz-linux \
    /boot/vmlinuz-linux
  do
    [[ -f "${p}" ]] || continue
    printf '%s' "${p}"
    return 0
  done
  return 1
}

coda_kver_from_kernel() {
  local path="$1"
  if [[ "${path}" == */usr/lib/modules/*/vmlinuz ]]; then
    basename "$(dirname "${path}")"
    return 0
  fi
  uname -r
}

coda_copy_ucode() {
  local dest_dir="$1"
  local src
  mkdir -p "${dest_dir}"
  for name in amd-ucode.img intel-ucode.img; do
    for src in \
      "/boot/${name}" \
      "/run/archiso/bootmnt/coda/boot/${name}" \
      /run/archiso/bootmnt/*/boot/"${name}" \
      "/mnt/boot/${name}"
    do
      if [[ -f "${src}" ]]; then
        cp -a "${src}" "${dest_dir}/${name}"
        break
      fi
    done
  done
}

coda_install_boot_files() {
  local dest="$1" slot="$2" esp="${3:-${dest}/boot}"
  local kpath kver
  kpath="$(coda_find_kernel "${dest}")" || coda_die "no vmlinuz found under ${dest} or live ISO"
  kver="$(coda_kver_from_kernel "${kpath}")"
  mkdir -p "${esp}/coda/${slot}"
  cp -a "${kpath}" "${esp}/coda/${slot}/vmlinuz-linux"
  coda_copy_ucode "${esp}/coda/${slot}"
  coda_write_mkinitcpio "${dest}"
  cat >"${dest}/etc/mkinitcpio.d/linux.preset" <<EOF
PRESETS=('default')
ALL_kver="/boot/coda/${slot}/vmlinuz-linux"
default_image="/boot/coda/${slot}/initramfs-linux.img"
EOF
  coda_log "mkinitcpio for slot ${slot} (kver=${kver})"
  if ! coda_chroot "${dest}" mkinitcpio -k "${kver}" -g "/boot/coda/${slot}/initramfs-linux.img"; then
    coda_die "mkinitcpio failed for slot ${slot}"
  fi
  [[ -f "${esp}/coda/${slot}/vmlinuz-linux" ]] || coda_die "kernel missing after boot install"
  [[ -f "${esp}/coda/${slot}/initramfs-linux.img" ]] || coda_die "initramfs missing after mkinitcpio"
}

coda_write_loader_entry() {
  local esp="$1" slot="$2" title="$3"
  mkdir -p "${esp}/loader/entries"
  local ucode=""
  if [[ -f "${esp}/coda/${slot}/intel-ucode.img" ]]; then
    ucode+=$'initrd /coda/'"${slot}"$'/intel-ucode.img\n'
  fi
  if [[ -f "${esp}/coda/${slot}/amd-ucode.img" ]]; then
    ucode+=$'initrd /coda/'"${slot}"$'/amd-ucode.img\n'
  fi
  cat >"${esp}/loader/entries/coda-${slot}.conf" <<EOF
title   ${title}
linux   /coda/${slot}/vmlinuz-linux
${ucode}initrd  /coda/${slot}/initramfs-linux.img
options root=PARTLABEL=coda-${slot} rw rootfstype=ext4
EOF
}

coda_write_loader_conf() {
  local esp="$1" default_slot="${2:-a}"
  mkdir -p "${esp}/loader"
  cat >"${esp}/loader/loader.conf" <<EOF
default coda-${default_slot}.conf
timeout 3
console-mode max
editor no
EOF
}

coda_bootctl_install() {
  local dest="$1"
  # Must register EFI variables so set-oneshot / set-default persist in OVMF.
  if command -v bootctl >/dev/null 2>&1; then
    bootctl install --esp-path="${dest}/boot" \
      || coda_chroot "${dest}" bootctl install --esp-path=/boot
  else
    coda_chroot "${dest}" bootctl install --esp-path=/boot
  fi
}

coda_create_user() {
  local dest="$1"
  local user="${CODA_INSTALL_USER:-user}"
  local password="${CODA_INSTALL_PASSWORD:-1}"
  local root_pw="${CODA_INSTALL_ROOT_PASSWORD:-1}"
  if coda_chroot "${dest}" id "${user}" >/dev/null 2>&1; then
    coda_log "user ${user} already exists"
  else
    coda_chroot "${dest}" useradd -m -G wheel,video,audio,input,render,storage,lp \
      -s /bin/bash "${user}" || coda_die "useradd ${user} failed"
  fi
  coda_chroot "${dest}" usermod -aG wheel,video,audio,input,render,storage,lp "${user}" || true
  printf '%s:%s\n' "${user}" "${password}" | coda_chroot "${dest}" chpasswd
  printf 'root:%s\n' "${root_pw}" | coda_chroot "${dest}" chpasswd
  if coda_chroot "${dest}" id live >/dev/null 2>&1; then
    coda_chroot "${dest}" userdel live >/dev/null 2>&1 || true
  fi
  rm -rf "${dest}/home/live"
}

coda_seed_data() {
  local dest="$1" data="$2"
  mkdir -p "${data}/home" "${data}/var"
  if [[ -d "${dest}/home" ]] && [[ -n "$(ls -A "${dest}/home" 2>/dev/null || true)" ]]; then
    rsync -aHAX "${dest}/home/" "${data}/home/"
  fi
  if [[ -d "${dest}/var" ]] && [[ -n "$(ls -A "${dest}/var" 2>/dev/null || true)" ]]; then
    rsync -aHAX "${dest}/var/" "${data}/var/"
  fi
  rm -rf "${dest}/home" "${dest}/var"
  mkdir -p "${dest}/home" "${dest}/var"
}

coda_bind_data() {
  local dest="$1" data="$2"
  mkdir -p "${dest}/home" "${dest}/var" "${dest}/coda/data"
  if ! mountpoint -q "${dest}/coda/data"; then
    mount --bind "${data}" "${dest}/coda/data"
  fi
  mount --bind "${data}/home" "${dest}/home"
  mount --bind "${data}/var" "${dest}/var"
}

coda_umount_tree() {
  local dest="$1"
  local m
  for m in \
    "${dest}/home" \
    "${dest}/var" \
    "${dest}/coda/data" \
    "${dest}/boot" \
    "${dest}/run" \
    "${dest}/dev" \
    "${dest}/sys" \
    "${dest}/proc"
  do
    umount "${m}" 2>/dev/null || true
  done
  umount "${dest}" 2>/dev/null || true
}

coda_find_post() {
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  for p in \
    "${CODA_INSTALL_POST:-}" \
    /usr/local/lib/codalinux/coda-install-post.sh \
    /usr/share/codalinux/install/coda-install-post.sh \
    "${here}/coda-install-post.sh"
  do
    [[ -n "${p}" && -x "${p}" ]] && { printf '%s' "${p}"; return 0; }
  done
  return 1
}

coda_active_slot() {
  if [[ -f /etc/coda/slot ]]; then
    tr -d '[:space:]' </etc/coda/slot
    return 0
  fi
  local src
  src="$(findmnt -n -o SOURCE / 2>/dev/null || true)"
  case "${src}" in
    *coda-a*) printf 'a'; return 0 ;;
    *coda-b*) printf 'b'; return 0 ;;
  esac
  return 1
}

coda_other_slot() {
  local s="${1:-}"
  [[ -n "${s}" ]] || s="$(coda_active_slot || true)"
  case "${s}" in
    a) printf 'b' ;;
    b) printf 'a' ;;
    *) return 1 ;;
  esac
}

coda_mount_existing() {
  local dest="$1" slot="$2"
  local root_dev esp_dev data_dev
  root_dev="$(coda_wait_partlabel "coda-${slot}")"
  esp_dev="$(coda_wait_partlabel coda-esp)"
  data_dev="$(coda_wait_partlabel coda-data)"
  mkdir -p "${dest}"
  mount "${root_dev}" "${dest}"
  mkdir -p "${dest}/boot" "${dest}/coda/data" "${dest}/home" "${dest}/var"
  mount "${esp_dev}" "${dest}/boot"
  mount "${data_dev}" "${dest}/coda/data"
  mkdir -p "${dest}/coda/data/home" "${dest}/coda/data/var"
  mount --bind "${dest}/coda/data/home" "${dest}/home"
  mount --bind "${dest}/coda/data/var" "${dest}/var"
  coda_bind_dev "${dest}"
}
