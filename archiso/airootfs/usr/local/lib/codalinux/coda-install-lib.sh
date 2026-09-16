#!/usr/bin/env bash
# Shared helpers for coda-install / coda-slot. Sourced, not executed.
# Offline payload = live airootfs split (core → slot, desktop → coda-data).
# Never pacstrap.

# Re-load if a stale CODA_INSTALL_LIB_SOURCED=1 leaked into the environment
# without the helpers (fresh qga_exec of coda-slot boot-test/promote).
# When this file is sourced from a function, `return` pops that function —
# callers must source us at script top-level so a stub cannot skip fallbacks.
if declare -F coda_need_root >/dev/null 2>&1; then
  return 0
fi
CODA_INSTALL_LIB_SOURCED=1

coda_log() { printf '%s: %s\n' "${CODA_INSTALL_LOG_PREFIX:-coda-install}" "$*"; }
coda_die() { printf '%s: %s\n' "${CODA_INSTALL_LOG_PREFIX:-coda-install}" "$*" >&2; exit 1; }

coda_live_lib_path() {
  printf '%s' /usr/local/lib/codalinux/coda-install-lib.sh
}

coda_share_lib_path() {
  printf '%s' /usr/share/codalinux/install/coda-install-lib.sh
}

coda_lib_file_has_need_root() {
  local f="$1"
  [[ -f "${f}" && -s "${f}" ]] || return 1
  grep -q 'coda_need_root()' "${f}"
}

coda_same_inode() {
  local a="$1" b="$2"
  [[ -e "${a}" || -L "${a}" ]] || return 1
  [[ -e "${b}" || -L "${b}" ]] || return 1
  [[ "$(stat -c '%d:%i' "${a}" 2>/dev/null || echo x)" == "$(stat -c '%d:%i' "${b}" 2>/dev/null || echo y)" ]]
}

# cp of a file onto itself (same inode via bind/overlay) opens dest O_TRUNC
# first and empties the live helper. Copy via a temp + rename instead.
coda_copy_file_safe() {
  local src="$1" dest="$2"
  [[ -e "${src}" || -L "${src}" ]] || return 0
  mkdir -p "$(dirname "${dest}")"
  if coda_same_inode "${src}" "${dest}"; then
    return 0
  fi
  local tmp
  tmp="$(mktemp "${dest}.XXXXXX")"
  cp -a "${src}" "${tmp}"
  mv -f "${tmp}" "${dest}"
}

coda_copy_tree_safe() {
  local src="$1" dest="$2"
  [[ -d "${src}" ]] || return 0
  mkdir -p "${dest}"
  if coda_same_inode "${src}" "${dest}"; then
    return 0
  fi
  cp -a "${src}/." "${dest}/"
}

coda_snapshot_live_helpers() {
  local dest="${1:-${CODA_LIVE_HELPER_SNAP:-/tmp/coda-live-helpers}}"
  local src share
  src="$(coda_live_lib_path)"
  share="$(coda_share_lib_path)"
  mkdir -p "${dest}"
  if coda_lib_file_has_need_root "${src}"; then
    coda_copy_file_safe "${src}" "${dest}/coda-install-lib.sh"
    if ! coda_lib_file_has_need_root "${share}"; then
      mkdir -p "$(dirname "${share}")"
      coda_copy_file_safe "${src}" "${share}" || true
    fi
  elif coda_lib_file_has_need_root "${share}"; then
    coda_copy_file_safe "${share}" "${dest}/coda-install-lib.sh"
  fi
}

coda_restore_live_helpers_if_broken() {
  local snap="${1:-${CODA_LIVE_HELPER_SNAP:-/tmp/coda-live-helpers}}"
  local live share from=""
  live="$(coda_live_lib_path)"
  share="$(coda_share_lib_path)"
  if coda_lib_file_has_need_root "${live}"; then
    return 0
  fi
  if coda_lib_file_has_need_root "${snap}/coda-install-lib.sh"; then
    from="${snap}/coda-install-lib.sh"
  elif coda_lib_file_has_need_root "${share}"; then
    from="${share}"
  fi
  [[ -n "${from}" ]] || return 1
  mkdir -p "$(dirname "${live}")"
  coda_copy_file_safe "${from}" "${live}"
  coda_log "restored ${live} from ${from} (next coda-slot must still load helpers)"
  coda_lib_file_has_need_root "${live}"
}

coda_need_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    coda_die "must run as root (live ISO: sudo -E; QGA guest-exec is already root)"
  fi
}

coda_pick_esp() {
  # Prefer an ESP that already has coda-*.conf. Live ISO /boot/loader is
  # the ISO's systemd-boot, not the installed disk.
  local target="${1:-/mnt/coda-slot}"
  if [[ -f /boot/loader/entries/coda-a.conf || -f /boot/loader/entries/coda-b.conf ]]; then
    printf '%s' /boot
    return 0
  fi
  if [[ -f "${target}/boot/loader/entries/coda-a.conf" \
     || -f "${target}/boot/loader/entries/coda-b.conf" ]]; then
    printf '%s' "${target}/boot"
    return 0
  fi
  return 1
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
  if coda_same_inode / "${dest}"; then
    coda_die "refusing to rsync onto the live root (would clobber running helpers)"
  fi
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

coda_ensure_usr_merge() {
  # kmod/modprobe/modinfo resolve MODULEDIR via /lib/modules/$kver.
  # Files-only rsync drops pacman dir nodes; filesystem lists lib/ as a
  # directory even when it is the usr-merge symlink. Recreate the four
  # compat links so offline mkinitcpio can see the module tree.
  local dest="$1"
  local name target destpath
  [[ -d "${dest}/usr/lib" || -L "${dest}/usr/lib" ]] \
    || coda_die "slot missing /usr/lib (usr-merge target)"
  [[ -d "${dest}/usr/bin" || -L "${dest}/usr/bin" ]] \
    || coda_die "slot missing /usr/bin (usr-merge target)"
  mkdir -p "${dest}/usr/sbin"
  for spec in "bin:usr/bin" "lib:usr/lib" "lib64:usr/lib" "sbin:usr/sbin"; do
    name="${spec%%:*}"
    target="${spec#*:}"
    destpath="${dest}/${name}"
    if [[ -L "${destpath}" ]]; then
      continue
    fi
    if [[ -d "${destpath}" ]]; then
      if [[ -z "$(find "${destpath}" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null || true)" ]]; then
        rmdir "${destpath}"
      else
        coda_die "slot /${name} is a directory; expected usr-merge symlink → ${target}"
      fi
    elif [[ -e "${destpath}" ]]; then
      coda_die "slot /${name} exists and is not a usr-merge symlink"
    fi
    ln -s "${target}" "${destpath}"
  done
}

coda_modtree_has_kos() {
  local tree="$1"
  [[ -d "${tree}" ]] || return 1
  [[ -n "$(find "${tree}" -type f \( -name '*.ko' -o -name '*.ko.*' \) -print -quit 2>/dev/null || true)" ]]
}

coda_module_ko() {
  local tree="$1" name="$2"
  find "${tree}" -type f \( -name "${name}.ko" -o -name "${name}.ko.*" \) -print -quit 2>/dev/null || true
}

coda_module_is_builtin() {
  # modules.builtin lines look like kernel/fs/ext4/ext4.ko (no compression).
  local tree="$1" name="$2"
  local builtin="${tree}/modules.builtin"
  [[ -f "${builtin}" ]] || return 1
  grep -Eq "(^|/)${name}\.ko(\.|$)" "${builtin}"
}

coda_module_present() {
  local tree="$1" name="$2"
  [[ -n "$(coda_module_ko "${tree}" "${name}")" ]] && return 0
  coda_module_is_builtin "${tree}" "${name}"
}

coda_modtree_bootable() {
  # A kver dir is bootable if vfat/fat/ext4 are loadable or built-in.
  local root="$1" kdir
  [[ -d "${root}" ]] || return 1
  for kdir in "${root}"/*; do
    [[ -d "${kdir}" ]] || continue
    coda_module_present "${kdir}" vfat || continue
    coda_module_present "${kdir}" fat || continue
    coda_module_present "${kdir}" ext4 || continue
    return 0
  done
  return 1
}

coda_sync_kernel_modules() {
  # File-list rsync may already have dropped a partial modules tree onto
  # the slot. Never merge onto it with -H (hardlink speedup against stubs).
  # Wipe dest and copy a known-good source as real files.
  local src="$1" dest="$2"
  local from="" candidate
  for candidate in \
    "${src}/usr/lib/modules" \
    /usr/lib/modules \
    /run/archiso/airootfs/usr/lib/modules
  do
    if coda_modtree_bootable "${candidate}"; then
      from="${candidate}"
      break
    fi
  done
  if [[ -z "${from}" ]]; then
    for candidate in \
      "${src}/usr/lib/modules" \
      /usr/lib/modules \
      /run/archiso/airootfs/usr/lib/modules
    do
      if coda_modtree_has_kos "${candidate}"; then
        from="${candidate}"
        break
      fi
    done
  fi
  [[ -n "${from}" ]] \
    || coda_die "no kernel modules (*.ko*) under ${src}/usr/lib/modules or live /usr/lib/modules"
  rm -rf "${dest}/usr/lib/modules"
  mkdir -p "${dest}/usr/lib/modules"
  coda_log "offline kernel modules ${from} → ${dest}/usr/lib/modules (replace, no hardlinks)"
  rsync -aAX --numeric-ids --info=stats1 "${from}/" "${dest}/usr/lib/modules/"
}

coda_assert_kernel_modules() {
  local dest="$1" kver="$2"
  local tree="${dest}/usr/lib/modules/${kver}"
  local name found
  [[ -n "${kver}" ]] || coda_die "kernel version required to assert module tree"
  [[ -d "${tree}" ]] || coda_die "missing module tree ${tree}"
  if [[ ! -e "${dest}/lib/modules/${kver}" ]]; then
    coda_die "usr-merge /lib does not resolve ${dest}/lib/modules/${kver} (kmod looks here)"
  fi
  for name in vfat fat ext4; do
    found="$(coda_module_ko "${tree}" "${name}")"
    if [[ -n "${found}" ]]; then
      continue
    fi
    if coda_module_is_builtin "${tree}" "${name}"; then
      coda_log "${name} is built-in (${kver})"
      continue
    fi
    coda_die "core slot missing ${name} module (and not in modules.builtin) under ${tree}"
  done
  found="$(find "${tree}" -type f \( -name 'virtio*.ko' -o -name 'virtio*.ko.*' \) -print -quit 2>/dev/null || true)"
  if [[ -z "${found}" ]]; then
    if coda_module_is_builtin "${tree}" virtio_blk || coda_module_is_builtin "${tree}" virtio_pci; then
      coda_log "virtio block/pci is built-in (${kver})"
    else
      coda_die "core slot missing virtio* modules under ${tree}"
    fi
  fi
}

coda_prepare_slot_modules() {
  local dest="$1"
  local src="${2:-}"
  if [[ -z "${src}" ]]; then
    src="$(coda_find_source)"
  fi
  coda_ensure_usr_merge "${dest}"
  coda_sync_kernel_modules "${src}" "${dest}"
}

coda_purge_leaked_desktop() {
  local dest="$1" desktop_list="$2"
  local rel wrap
  [[ -d "${dest}" ]] || return 0
  if coda_same_inode / "${dest}"; then
    coda_log "refusing to purge desktop leaks from the live root"
    return 0
  fi
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
  # greetd.service + PAM must live on the slot. The greetd *binary* stays
  # on coda-data and appears after coda-desktop-mount. A wants symlink to
  # /usr/lib/systemd/system/greetd.service is dangling until sysext merge,
  # so systemd drops greetd from the boot transaction and Hyprland never
  # starts.
  coda_install_slot_greetd "${dest}"
}

coda_install_slot_greetd() {
  # Unit file + PAM only. Do not copy /usr/bin/greetd or Hyprland.
  local dest="$1"
  mkdir -p "${dest}/etc/systemd/system/greetd.service.d" \
    "${dest}/etc/pam.d" \
    "${dest}/etc/systemd/system/multi-user.target.wants" \
    "${dest}/etc/systemd/system/graphical.target.wants"

  local src_unit=""
  local p
  for p in \
    /usr/lib/systemd/system/greetd.service \
    /lib/systemd/system/greetd.service
  do
    if [[ -f "${p}" ]]; then
      src_unit="${p}"
      break
    fi
  done

  if [[ -n "${src_unit}" && ! -L "${src_unit}" ]]; then
    cp -a "${src_unit}" "${dest}/etc/systemd/system/greetd.service"
  elif [[ -n "${src_unit}" ]]; then
    # Follow a vendor symlink once; never leave a dangling /usr/lib link.
    cp -aL "${src_unit}" "${dest}/etc/systemd/system/greetd.service" 2>/dev/null \
      || src_unit=""
  fi
  if [[ ! -f "${dest}/etc/systemd/system/greetd.service" \
     || -L "${dest}/etc/systemd/system/greetd.service" ]]; then
    cat >"${dest}/etc/systemd/system/greetd.service" <<'EOF'
[Unit]
Description=Greeter daemon
Documentation=man:greetd(1)
After=systemd-user-sessions.service plymouth-quit-wait.service
After=getty@tty1.service
Conflicts=getty@tty1.service

[Service]
Type=simple
ExecStart=/usr/bin/greetd
IgnoreSIGPIPE=no
SendSIGHUP=yes
TimeoutStopSec=30
Restart=always
RestartSec=1

[Install]
WantedBy=graphical.target
Alias=display-manager.service
EOF
  fi

  # Do not condition on the greetd binary path: that file appears only
  # after the late sysext merge. A failed condition skips the unit for
  # the whole boot (ConditionResult=no, no start job) — the e2e failure
  # mode. After= the mount unit is enough; Restart= covers a too-early
  # ExecStart.
  rm -f "${dest}/etc/systemd/system/greetd.service.d/coda.conf"
  cat >"${dest}/etc/systemd/system/greetd.service.d/coda-desktop-mount.conf" <<'EOF'
[Unit]
# Binary and session wrappers come from /coda/data/desktop after merge.
After=coda-desktop-mount.service
Wants=coda-desktop-mount.service
EOF

  if [[ -f /etc/pam.d/greetd ]]; then
    cp -a /etc/pam.d/greetd "${dest}/etc/pam.d/greetd"
  else
    cat >"${dest}/etc/pam.d/greetd" <<'EOF'
#%PAM-1.0
auth       include      system-login
account    include      system-login
password   include      system-login
session    include      system-login
EOF
  fi

  # Resolve on the slot before sysext. Do not point at /usr/lib (desktop).
  ln -sfn /etc/systemd/system/greetd.service \
    "${dest}/etc/systemd/system/display-manager.service"
  ln -sfn /etc/systemd/system/greetd.service \
    "${dest}/etc/systemd/system/multi-user.target.wants/greetd.service"
  ln -sfn /etc/systemd/system/greetd.service \
    "${dest}/etc/systemd/system/graphical.target.wants/greetd.service"
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
  coda_prepare_slot_modules "${dest}" "${src}"
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

coda_scrub_live_greetd() {
  # Live ISO drop-in Wants=coda-live-setup. Confext would re-apply it
  # from /coda/data/desktop/etc and leave greetd waiting on a removed
  # live unit. Dangling /usr/lib DM wants are invisible at first boot.
  local dest="$1"
  local link dest_link
  rm -f "${dest}/etc/systemd/system/coda-live-setup.service"
  rm -f "${dest}/etc/systemd/system/multi-user.target.wants/coda-live-setup.service"
  rm -f "${dest}/etc/systemd/system/greetd.service.d/coda.conf"
  for link in \
    etc/systemd/system/display-manager.service \
    etc/systemd/system/multi-user.target.wants/greetd.service \
    etc/systemd/system/graphical.target.wants/greetd.service
  do
    dest_link="${dest}/${link}"
    if [[ -L "${dest_link}" ]]; then
      case "$(readlink "${dest_link}")" in
        /usr/lib/systemd/system/greetd.service|/lib/systemd/system/greetd.service)
          rm -f "${dest_link}"
          ;;
      esac
    fi
  done
}

coda_wipe_live_bits() {
  local dest="$1"
  rm -f "${dest}/etc/mkinitcpio.conf.d/archiso.conf"
  coda_scrub_live_greetd "${dest}"
  rm -f "${dest}/etc/systemd/system/getty@tty2.service.d/autologin.conf"
  rm -rf "${dest}/etc/systemd/system/getty@tty2.service.d"
  rm -f "${dest}/etc/machine-id"
  rm -f "${dest}/var/lib/dbus/machine-id"
}

coda_write_mkinitcpio() {
  local dest="$1"
  mkdir -p "${dest}/etc"
  cat >"${dest}/etc/mkinitcpio.conf" <<'EOF'
# CodaLinux installed slot — stock hooks, no archiso.
MODULES=()
BINARIES=()
FILES=()
HOOKS=(base udev autodetect microcode modconf kms keyboard keymap consolefont block filesystems fsck)
COMPRESSION="zstd"
EOF
}

coda_write_linux_preset() {
  local dest="$1" slot="$2"
  # Core-only file lists omit empty package dirs (mkinitcpio owns
  # /etc/mkinitcpio.d/). The live ISO often has no linux.preset, so
  # rsync never creates the parent. mkdir before the redirect.
  mkdir -p "${dest}/etc/mkinitcpio.d"
  cat >"${dest}/etc/mkinitcpio.d/linux.preset" <<EOF
PRESETS=('default')
ALL_kver="/boot/coda/${slot}/vmlinuz-linux"
default_image="/boot/coda/${slot}/initramfs-linux.img"
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
  coda_prepare_slot_modules "${dest}"
  kpath="$(coda_find_kernel "${dest}")" || coda_die "no vmlinuz found under ${dest} or live ISO"
  kver="$(coda_kver_from_kernel "${kpath}")"
  coda_assert_kernel_modules "${dest}" "${kver}"
  mkdir -p "${esp}/coda/${slot}"
  cp -a "${kpath}" "${esp}/coda/${slot}/vmlinuz-linux"
  coda_copy_ucode "${esp}/coda/${slot}"
  coda_write_mkinitcpio "${dest}"
  coda_write_linux_preset "${dest}" "${slot}"
  coda_log "depmod ${kver} for slot ${slot}"
  if ! coda_chroot "${dest}" depmod -a "${kver}"; then
    coda_die "depmod failed for slot ${slot} (kver=${kver})"
  fi
  [[ -f "${dest}/usr/lib/modules/${kver}/modules.dep" \
    || -f "${dest}/usr/lib/modules/${kver}/modules.dep.bin" ]] \
    || coda_die "depmod did not write modules.dep for ${kver}"
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
