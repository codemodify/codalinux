#!/usr/bin/env bash
# Partition ESP+OS-A+OS-B+data and offline-install the live image into OS-A.
# Called by coda-install. Not an archinstall/pacstrap path.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=coda-install-lib.sh
. "${CODA_INSTALL_LIB:-${here}/coda-install-lib.sh}"
CODA_INSTALL_LOG_PREFIX=coda-install-ab

usage() {
  cat <<'EOF'
Usage: coda-install-ab.sh --disk DEV

Wipe DEV and install CodaLinux offline:
  GPT: coda-esp (FAT32) + coda-a + coda-b (ext4) + coda-data (ext4)
  Copy live airootfs into OS-A (no mirrors)
  systemd-boot entries A (default) and B (placeholder)
  data bind-mounted at /home and /var

Environment:
  CODA_INSTALL_DISK     same as --disk
  CODA_INSTALL_USER     default user
  CODA_INSTALL_PASSWORD default 1
  CODA_INSTALL_TARGET   mount root (default /mnt)
  CODA_INSTALL_SOURCE   override live payload
EOF
}

disk=""
target="${CODA_INSTALL_TARGET:-/mnt}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --disk) disk="$2"; shift 2 ;;
    --target) target="$2"; shift 2 ;;
    *) coda_die "unknown argument: $1" ;;
  esac
done

disk="${disk:-${CODA_INSTALL_DISK:-}}"
[[ -n "${disk}" ]] || coda_die "missing --disk / CODA_INSTALL_DISK"
[[ -b "${disk}" ]] || coda_die "${disk} is not a block device"

coda_need_root
command -v sgdisk >/dev/null || coda_die "sgdisk missing (gptfdisk)"
command -v rsync >/dev/null || coda_die "rsync missing"
command -v mkfs.fat >/dev/null || coda_die "mkfs.fat missing (dosfstools)"
command -v mkfs.ext4 >/dev/null || coda_die "mkfs.ext4 missing (e2fsprogs)"

plan_json="$(coda_plan_json "${disk}")" || coda_die "$(python3 "$(coda_layout_bin)" check "${disk}" 2>&1 || true)"
esp_mib="$(printf '%s' "${plan_json}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["esp_mib"])')"
a_mib="$(printf '%s' "${plan_json}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["slot_a_mib"])')"
b_mib="$(printf '%s' "${plan_json}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["slot_b_mib"])')"

coda_log "plan on ${disk}: ESP ${esp_mib}MiB  A ${a_mib}MiB  B ${b_mib}MiB  data=remainder"
python3 "$(coda_layout_bin)" plan "${disk}"

src="$(coda_find_source)"
coda_log "offline source: ${src}"
if [[ "${src}" == / ]]; then
  coda_log "airootfs not mounted; copying live / with exclusions (still offline)"
fi

# Fail early if the live payload cannot fit the slot.
src_bytes="$(coda_source_bytes "${src}" || echo 0)"
slot_bytes=$((a_mib * 1024 * 1024))
if [[ "${src_bytes}" -gt 0 && $((src_bytes + src_bytes / 10)) -gt "${slot_bytes}" ]]; then
  coda_die "live payload (${src_bytes} bytes) does not fit OS-A (${slot_bytes} bytes)"
fi

coda_log "unmounting any existing filesystems on ${disk}"
coda_umount_tree "${target}"
# Swap or leftover mounts on this disk.
while read -r mp; do
  [[ -n "${mp}" ]] || continue
  umount "${mp}" 2>/dev/null || umount -l "${mp}" 2>/dev/null || true
done < <(lsblk -lnpo NAME,MOUNTPOINT "${disk}" | awk '$2 != "" {print $2}')
swapoff -a >/dev/null 2>&1 || true

coda_log "wiping ${disk} and writing GPT (coda-esp / coda-a / coda-b / coda-data)"
wipefs -a "${disk}" >/dev/null 2>&1 || true
sgdisk --zap-all "${disk}" >/dev/null
sgdisk \
  -n "1:0:+${esp_mib}MiB" -t 1:EF00 -c 1:coda-esp \
  -n "2:0:+${a_mib}MiB" -t 2:8300 -c 2:coda-a \
  -n "3:0:+${b_mib}MiB" -t 3:8300 -c 3:coda-b \
  -n 4:0:0 -t 4:8300 -c 4:coda-data \
  "${disk}" >/dev/null
partprobe "${disk}" >/dev/null 2>&1 || true
udevadm settle >/dev/null 2>&1 || true

esp_dev="$(coda_wait_partlabel coda-esp)"
a_dev="$(coda_wait_partlabel coda-a)"
b_dev="$(coda_wait_partlabel coda-b)"
data_dev="$(coda_wait_partlabel coda-data)"

coda_log "formatting ${esp_dev} FAT32, ${a_dev}/${b_dev}/${data_dev} ext4"
mkfs.fat -F32 -n CODAESP "${esp_dev}" >/dev/null
mkfs.ext4 -F -L coda-a "${a_dev}" >/dev/null
mkfs.ext4 -F -L coda-b "${b_dev}" >/dev/null
mkfs.ext4 -F -L coda-data "${data_dev}" >/dev/null

mkdir -p "${target}"
mount "${a_dev}" "${target}"
mkdir -p "${target}/boot" "${target}/coda/data"
mount "${esp_dev}" "${target}/boot"
data_mnt="${target}/coda/data"
mount "${data_dev}" "${data_mnt}"

coda_rsync_root "${src}" "${target}" 0
coda_wipe_live_bits "${target}"
coda_seed_data "${target}" "${data_mnt}"
coda_bind_data "${target}" "${data_mnt}"
coda_bind_dev "${target}"
coda_write_fstab "${target}" a
coda_bozeman "${target}"
coda_write_slot_marker "${target}" a 0
printf '%s\n' "${plan_json}" >"${target}/etc/coda/layout.json"
mkdir -p "${data_mnt}/home" "${data_mnt}/var"

coda_create_user "${target}"
if ! coda_chroot "${target}" systemd-machine-id-setup >/dev/null 2>&1; then
  coda_log "systemd-machine-id-setup skipped"
fi
if [[ -x "${target}/usr/bin/locale-gen" || -x "${target}/usr/sbin/locale-gen" ]]; then
  coda_chroot "${target}" locale-gen >/dev/null 2>&1 || true
fi

coda_bootctl_install "${target}"
coda_install_boot_files "${target}" a "${target}/boot"
coda_write_loader_entry "${target}/boot" a "CodaLinux (slot A)"
coda_write_loader_conf "${target}/boot" a
# Placeholder B entry: no kernels yet. systemd-boot will skip a missing linux.
cat >"${target}/boot/loader/entries/coda-b.conf" <<'EOF'
title   CodaLinux (slot B — empty)
linux   /coda/b/vmlinuz-linux
initrd  /coda/b/initramfs-linux.img
options root=PARTLABEL=coda-b rw rootfstype=ext4
EOF
if command -v bootctl >/dev/null 2>&1; then
  bootctl set-default coda-a.conf --esp-path="${target}/boot" >/dev/null 2>&1 || true
fi

# Empty OS-B marker so coda-slot can find a formatted inactive slot.
mkdir -p /tmp/coda-empty-b
mount "${b_dev}" /tmp/coda-empty-b
mkdir -p /tmp/coda-empty-b/etc/coda
printf 'b\n' >/tmp/coda-empty-b/etc/coda/slot
printf '1\n' >/tmp/coda-empty-b/etc/coda/empty
umount /tmp/coda-empty-b
rmdir /tmp/coda-empty-b 2>/dev/null || true

post="$(coda_find_post || true)"
if [[ -n "${post}" ]]; then
  coda_log "running ${post}"
  CODA_INSTALL_USER="${CODA_INSTALL_USER:-user}" \
    "${post}" --user "${CODA_INSTALL_USER:-user}" --target "${target}"
else
  coda_die "coda-install-post.sh missing"
fi

coda_write_fstab "${target}" a

coda_log "install into OS-A complete (default boot: coda-a.conf)"
coda_log "leave ${target} mounted for verify, or reboot from disk"
