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
  for local_bin in coda-wallpaper coda-hyprpaper coda-hyprland coda-hypr-ws coda-ags coda-settings coda-sandbox coda-install; do
    if [[ -f "${scripts_src}/${local_bin}" ]]; then
      install -m 0755 "${scripts_src}/${local_bin}" "/usr/local/bin/${local_bin}"
      log "wrapper → /usr/local/bin/${local_bin}"
    fi
  done
  if [[ -f "${scripts_src}/coda-install-config.py" ]]; then
    install -d /usr/local/lib/codalinux
    install -m 0755 "${scripts_src}/coda-install-config.py" \
      /usr/local/lib/codalinux/coda-install-config.py
    log "helper → /usr/local/lib/codalinux/coda-install-config.py"
  fi
fi

# Optional host-built system-config (ISO already ships these after rebuild).
sc_bin=""
for sc_bin in \
  "${share}/core/system-config/bin" \
  "${share}/bin"; do
  [[ -d "${sc_bin}" ]] || continue
  install -d /usr/local/bin /usr/local/lib/codalinux
  local_bin=""
  for local_bin in system-config system-configd system-config-apply \
                   system-config-report system-config-tui; do
    if [[ -x "${sc_bin}/${local_bin}" ]]; then
      install -m 0755 "${sc_bin}/${local_bin}" "/usr/local/bin/${local_bin}"
      log "system-config → /usr/local/bin/${local_bin}"
    fi
  done
  if [[ -x "${sc_bin}/system-config-gui" ]]; then
    install -m 0755 "${sc_bin}/system-config-gui" \
      /usr/local/lib/codalinux/system-config-gui
    log "system-config-gui → /usr/local/lib/codalinux/system-config-gui"
  fi
  break
done
if [[ -x "${share}/scripts/system-config-gui" ]]; then
  install -m 0755 "${share}/scripts/system-config-gui" \
    /usr/local/bin/system-config-gui
  log "helper → /usr/local/bin/system-config-gui"
fi
if [[ -x "${share}/scripts/system-config-apply-launch" ]]; then
  install -d /usr/local/lib/codalinux
  install -m 0755 "${share}/scripts/system-config-apply-launch" \
    /usr/local/lib/codalinux/system-config-apply-launch
  log "helper → /usr/local/lib/codalinux/system-config-apply-launch"
fi

desktop_user() {
  if id live >/dev/null 2>&1; then
    printf '%s' live
    return 0
  fi
  if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != root ]]; then
    printf '%s' "${SUDO_USER}"
    return 0
  fi
  printf '%s' live
}

as_desktop() {
  local user="$1"
  shift
  local home uid runtime wayland sig sock inst
  home="$(getent passwd "${user}" | cut -d: -f6 || true)"
  home="${home:-/home/${user}}"
  [[ -d "${home}" ]] || home=/tmp
  uid="$(id -u "${user}")"
  runtime="${XDG_RUNTIME_DIR:-/run/user/${uid}}"
  if [[ ! -d "${runtime}" ]]; then
    runtime="/run/user/${uid}"
  fi
  wayland="${WAYLAND_DISPLAY:-}"
  if [[ -z "${wayland}" && -d "${runtime}" ]]; then
    for sock in "${runtime}"/wayland-*; do
      [[ -e "${sock}" ]] || continue
      [[ "${sock}" == *.lock ]] && continue
      wayland="$(basename "${sock}")"
      break
    done
  fi
  wayland="${wayland:-wayland-1}"
  sig="${HYPRLAND_INSTANCE_SIGNATURE:-}"
  if [[ -z "${sig}" ]]; then
    for inst in "${runtime}/hypr"/* /tmp/hypr/*; do
      [[ -d "${inst}" ]] || continue
      sig="$(basename "${inst}")"
      break
    done
  fi
  local -a cmd=(
    sudo -u "${user}" -- env
    "HOME=${home}"
    "USER=${user}"
    "LOGNAME=${user}"
    "XDG_RUNTIME_DIR=${runtime}"
    "WAYLAND_DISPLAY=${wayland}"
    "XDG_SESSION_TYPE=wayland"
    "XDG_CURRENT_DESKTOP=${XDG_CURRENT_DESKTOP:-Hyprland}"
  )
  if [[ -n "${sig}" ]]; then
    cmd+=("HYPRLAND_INSTANCE_SIGNATURE=${sig}")
  fi
  cmd+=(bash -lc)
  local quoted=""
  local arg
  for arg in "$@"; do
    quoted+="$(printf '%q ' "${arg}")"
  done
  "${cmd[@]}" "cd \"\$HOME\" 2>/dev/null || cd /tmp; ${quoted}"
}

restart_session_tools() {
  local user
  user="$(desktop_user)"
  if ! id "${user}" >/dev/null 2>&1; then
    log "no desktop user ${user}; skip session restart"
    return 1
  fi
  as_desktop "${user}" killall swaybg hyprpaper coda-hyprpaper coda-wallpaper \
    >/dev/null 2>&1 || true
  if command -v coda-wallpaper >/dev/null 2>&1; then
    as_desktop "${user}" coda-wallpaper >/dev/null 2>&1 &
  elif command -v coda-hyprpaper >/dev/null 2>&1; then
    as_desktop "${user}" coda-hyprpaper >/dev/null 2>&1 &
  fi
  if command -v coda-ags >/dev/null 2>&1; then
    as_desktop "${user}" coda-ags quit >/dev/null 2>&1 || true
    as_desktop "${user}" coda-ags >/dev/null 2>&1 &
  fi
  as_desktop "${user}" hyprctl reload >/dev/null 2>&1 || true
  log "restarted session tools as ${user} (never as root from /root)"
  return 0
}

if restart_session_tools; then
  :
else
  cat <<'EOF'
Copied. Restart from a Hyprland terminal (user live):

  cd ~
  killall swaybg hyprpaper; coda-wallpaper
  coda-ags quit; coda-ags &
  hyprctl reload

If swaybg is missing on this ISO: pacman -S --noconfirm swaybg
(or rebuild). Some hyprland.lua changes still need a session restart.
EOF
fi
