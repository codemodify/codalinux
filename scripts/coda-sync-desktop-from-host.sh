#!/usr/bin/env bash
# Run inside the live guest after the virtio-9p share is mounted.
# Copies hypr configs, Horos wallpaper, AGS, and wallpaper wrappers
# from the host share into live paths. Restart coda-wallpaper / AGS
# from a Hyprland terminal if this is not already one.
set -euo pipefail

share="${1:-${CODA_HOST_SHARE:-/mnt/coda-host}}"

log() { printf 'coda-sync-desktop: %s\n' "$*"; }
die() { printf 'coda-sync-desktop: %s\n' "$*" >&2; exit 1; }

[[ -d "${share}" ]] || die "share not mounted or missing: ${share} (mount the 9p tag first)"

hypr_src=""
ags_src=""
wall_src=""

if [[ -d "${share}/desktop/hypr" ]]; then
  hypr_src="${share}/desktop/hypr"
elif [[ -d "${share}/hypr" ]]; then
  hypr_src="${share}/hypr"
fi
if [[ -d "${share}/desktop/ags" ]]; then
  ags_src="${share}/desktop/ags"
elif [[ -d "${share}/ags" ]]; then
  ags_src="${share}/ags"
fi
if [[ -f "${share}/branding/wallpapers/default.png" ]]; then
  wall_src="${share}/branding/wallpapers/default.png"
elif [[ -f "${share}/wallpapers/default.png" ]]; then
  wall_src="${share}/wallpapers/default.png"
fi

[[ -n "${hypr_src}" || -n "${ags_src}" || -n "${wall_src}" ]] || \
  die "no desktop/hypr, desktop/ags, or branding/wallpapers under ${share}"

if [[ "${EUID}" -ne 0 ]]; then
  die "run as root (tty2) so /etc/xdg/hypr and /usr/local/share can be written"
fi

copy_hypr_dir() {
  local dest="$1"
  install -d "${dest}"
  local f
  for f in "${hypr_src}"/*; do
    [[ -f "${f}" ]] || continue
    case "${f}" in
      *.md) continue ;;
    esac
    install -m 0644 "${f}" "${dest}/"
  done
  log "hypr → ${dest}"
}

if [[ -n "${hypr_src}" ]]; then
  copy_hypr_dir /etc/xdg/hypr
  copy_hypr_dir /root/.config/hypr
  if [[ -d /home/live ]]; then
    copy_hypr_dir /home/live/.config/hypr
    if id live >/dev/null 2>&1; then
      chown -R live:live /home/live/.config/hypr
    fi
  fi
  if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != root ]]; then
    copy_hypr_dir "/home/${SUDO_USER}/.config/hypr"
  fi
fi

if [[ -n "${wall_src}" ]]; then
  install -d /usr/share/backgrounds/codalinux
  install -m 0644 "${wall_src}" /usr/share/backgrounds/codalinux/default.png
  log "wallpaper → /usr/share/backgrounds/codalinux/default.png"
fi

if [[ -n "${ags_src}" ]]; then
  install -d /usr/local/share/codalinux/ags
  cp -a "${ags_src}/." /usr/local/share/codalinux/ags/
  log "ags → /usr/local/share/codalinux/ags/"
fi

scripts_src=""
if [[ -d "${share}/scripts" ]]; then
  scripts_src="${share}/scripts"
fi
if [[ -n "${scripts_src}" ]]; then
  install -d /usr/local/bin
  local_bin=""
  for local_bin in coda-wallpaper coda-hyprpaper coda-hyprland; do
    if [[ -f "${scripts_src}/${local_bin}" ]]; then
      install -m 0755 "${scripts_src}/${local_bin}" "/usr/local/bin/${local_bin}"
      log "wrapper → /usr/local/bin/${local_bin}"
    fi
  done
fi

restart_session_tools() {
  if [[ -z "${HYPRLAND_INSTANCE_SIGNATURE:-}" ]]; then
    return 1
  fi
  killall swaybg hyprpaper coda-hyprpaper coda-wallpaper 2>/dev/null || true
  if command -v coda-wallpaper >/dev/null 2>&1; then
    coda-wallpaper >/dev/null 2>&1 &
  elif command -v coda-hyprpaper >/dev/null 2>&1; then
    coda-hyprpaper >/dev/null 2>&1 &
  fi
  if command -v coda-ags >/dev/null 2>&1; then
    coda-ags quit >/dev/null 2>&1 || true
    coda-ags >/dev/null 2>&1 &
  fi
  hyprctl reload >/dev/null 2>&1 || true
  return 0
}

if restart_session_tools; then
  log "restarted coda-wallpaper / coda-ags and hyprctl reload"
else
  cat <<'EOF'
Copied. Restart from a Hyprland terminal (user live):

  killall swaybg hyprpaper; coda-wallpaper
  coda-ags quit; coda-ags &
  hyprctl reload

If swaybg is missing on this ISO: pacman -S --noconfirm swaybg
(or rebuild). Some hyprland.lua changes still need a session restart.
EOF
fi
