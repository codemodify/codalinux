#!/usr/bin/env bash
# Run inside the live guest after the virtio-9p share is mounted.
# Copies hypr configs, Horos wallpaper, AGS, and wallpaper wrappers
# from the host share into live paths, then restarts session tools as
# the seat user (never as root from tty2).
#
# as_desktop must not inherit root's XDG_RUNTIME_DIR (/run/user/0).
# That is the usual tty2 case: Hyprland/AGS run as live on tty1, sync
# is invoked as root, and a wrong runtime means WAYLAND_DISPLAY /
# HYPRLAND_INSTANCE_SIGNATURE never resolve. coda-ags then exits.
set -euo pipefail

log() { printf 'coda-sync-desktop: %s\n' "$*"; }
die() { printf 'coda-sync-desktop: %s\n' "$*" >&2; exit 1; }

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

coda_user_uid() {
  if [[ -n "${CODA_TEST_UID:-}" ]]; then
    printf '%s' "${CODA_TEST_UID}"
    return 0
  fi
  id -u "$1"
}

coda_user_home() {
  if [[ -n "${CODA_TEST_HOME:-}" ]]; then
    printf '%s' "${CODA_TEST_HOME}"
    return 0
  fi
  local home
  home="$(getent passwd "$1" | cut -d: -f6 || true)"
  home="${home:-/home/$1}"
  [[ -d "${home}" ]] || home=/tmp
  printf '%s' "${home}"
}

coda_runtime_root() {
  printf '%s' "${CODA_RUNTIME_ROOT:-/run/user}"
}

coda_tmp_hypr() {
  printf '%s' "${CODA_TMP_HYPR:-/tmp/hypr}"
}

# Seat runtime is always <runtime-root>/<uid>. Caller's XDG_RUNTIME_DIR is
# trusted only when it already is that path (Hyprland terminal as live).
# Root on tty2 has XDG_RUNTIME_DIR=/run/user/0 — that must be ignored.
coda_desktop_runtime() {
  local user="$1"
  local uid
  uid="$(coda_user_uid "${user}")"
  printf '%s' "$(coda_runtime_root)/${uid}"
}

coda_discover_wayland() {
  local runtime="$1"
  local sock
  [[ -d "${runtime}" ]] || return 1
  for sock in "${runtime}"/wayland-*; do
    [[ -e "${sock}" ]] || continue
    [[ "${sock}" == *.lock ]] && continue
    printf '%s' "$(basename "${sock}")"
    return 0
  done
  return 1
}

coda_discover_hypr_sig() {
  local runtime="$1"
  local tmp_hypr inst best="" best_mtime=-1 mtime
  tmp_hypr="$(coda_tmp_hypr)"
  for inst in "${runtime}/hypr"/* "${tmp_hypr}"/*; do
    [[ -d "${inst}" ]] || continue
    mtime="$(stat -c '%Y' "${inst}" 2>/dev/null || echo 0)"
    if [[ -z "${best}" || "${mtime}" -ge "${best_mtime}" ]]; then
      best="$(basename "${inst}")"
      best_mtime="${mtime}"
    fi
  done
  printf '%s' "${best}"
}

# Sets CODA_DS_{HOME,UID,RUNTIME,WAYLAND,SIG} for $1 (desktop user).
coda_desktop_session_env() {
  local user="$1"
  local caller_rt
  CODA_DS_UID="$(coda_user_uid "${user}")"
  CODA_DS_HOME="$(coda_user_home "${user}")"
  CODA_DS_RUNTIME="$(coda_desktop_runtime "${user}")"
  CODA_DS_WAYLAND=""
  CODA_DS_SIG=""
  caller_rt="${XDG_RUNTIME_DIR:-}"
  # Trust inherited Wayland/HIS only when they already point at this seat.
  if [[ -n "${caller_rt}" && "${caller_rt}" == "${CODA_DS_RUNTIME}" ]]; then
    CODA_DS_WAYLAND="${WAYLAND_DISPLAY:-}"
    CODA_DS_SIG="${HYPRLAND_INSTANCE_SIGNATURE:-}"
  fi
  if [[ -z "${CODA_DS_WAYLAND}" ]]; then
    CODA_DS_WAYLAND="$(coda_discover_wayland "${CODA_DS_RUNTIME}" || true)"
  fi
  if [[ -z "${CODA_DS_SIG}" ]]; then
    CODA_DS_SIG="$(coda_discover_hypr_sig "${CODA_DS_RUNTIME}")"
  fi
}

coda_session_display_ready() {
  [[ -n "${CODA_DS_RUNTIME:-}" && -n "${CODA_DS_WAYLAND:-}" ]] || return 1
  [[ -e "${CODA_DS_RUNTIME}/${CODA_DS_WAYLAND}" ]] || return 1
  return 0
}

# KEY=VALUE lines for sudo env. Used by as_desktop and host tests.
coda_as_desktop_env_lines() {
  local user="$1"
  coda_desktop_session_env "${user}"
  printf 'HOME=%s\n' "${CODA_DS_HOME}"
  printf 'USER=%s\n' "${user}"
  printf 'LOGNAME=%s\n' "${user}"
  printf 'XDG_RUNTIME_DIR=%s\n' "${CODA_DS_RUNTIME}"
  printf 'XDG_SESSION_TYPE=wayland\n'
  printf 'XDG_CURRENT_DESKTOP=%s\n' "${XDG_CURRENT_DESKTOP:-Hyprland}"
  if [[ -n "${CODA_DS_WAYLAND}" ]]; then
    printf 'WAYLAND_DISPLAY=%s\n' "${CODA_DS_WAYLAND}"
  fi
  if [[ -n "${CODA_DS_SIG}" ]]; then
    printf 'HYPRLAND_INSTANCE_SIGNATURE=%s\n' "${CODA_DS_SIG}"
  fi
}

coda_ags_pids() {
  local user="$1"
  if [[ -n "${CODA_TEST_AGS_PIDFILE:-}" ]]; then
    cat "${CODA_TEST_AGS_PIDFILE}" 2>/dev/null || true
    return 0
  fi
  pgrep -u "${user}" -f '(^|/)(coda-ags|ags)([ ]|$)' 2>/dev/null || true
}

coda_ags_running_as() {
  local user="$1"
  local pids
  pids="$(coda_ags_pids "${user}")"
  [[ -n "${pids}" ]]
}

coda_wait_ags_gone() {
  local user="$1"
  local timeout_s="${2:-8}"
  local n=0
  local max=$((timeout_s * 10))
  while [[ "${n}" -lt "${max}" ]]; do
    if ! coda_ags_running_as "${user}"; then
      return 0
    fi
    sleep "${CODA_AGS_POLL_S:-0.1}"
    n=$((n + 1))
  done
  ! coda_ags_running_as "${user}"
}

coda_wait_ags_up() {
  local user="$1"
  local ticks="${2:-${CODA_AGS_START_WAIT_TICKS:-20}}"
  local n=0
  while [[ "${n}" -lt "${ticks}" ]]; do
    if coda_ags_running_as "${user}"; then
      return 0
    fi
    sleep "${CODA_AGS_POLL_S:-0.1}"
    n=$((n + 1))
  done
  coda_ags_running_as "${user}"
}

as_desktop() {
  local user="$1"
  shift
  local -a cmd=(sudo -u "${user}" -- env)
  local line
  while IFS= read -r line; do
    [[ -n "${line}" ]] || continue
    cmd+=("${line}")
  done < <(coda_as_desktop_env_lines "${user}")
  cmd+=(bash -lc)
  local quoted=""
  local arg
  for arg in "$@"; do
    quoted+="$(printf '%q ' "${arg}")"
  done
  "${cmd[@]}" "cd \"\$HOME\" 2>/dev/null || cd /tmp; ${quoted}"
}

# Quit, wait, start with retry, verify a process as the seat user.
coda_restart_ags() {
  local user="$1"
  local tries=0
  local max="${CODA_AGS_START_TRIES:-4}"
  local starter

  coda_desktop_session_env "${user}"
  log "session ${user} uid=${CODA_DS_UID} XDG_RUNTIME_DIR=${CODA_DS_RUNTIME} WAYLAND=${CODA_DS_WAYLAND:-<none>} HIS=${CODA_DS_SIG:-<none>}"

  if ! coda_session_display_ready; then
    log "AGS not restarted: display not ready for ${user}"
    log "  XDG_RUNTIME_DIR=${CODA_DS_RUNTIME} WAYLAND=${CODA_DS_WAYLAND:-<none>} HIS=${CODA_DS_SIG:-<none>}"
    log "  Hyprland must be up on tty1 (wayland socket under the seat runtime)."
    log "  Retry sync after the session paints; do not start AGS as root."
    return 1
  fi
  if [[ -z "${CODA_DS_SIG}" ]]; then
    log "warning: no HYPRLAND_INSTANCE_SIGNATURE under ${CODA_DS_RUNTIME}/hypr (bar may still start)"
  fi

  as_desktop "${user}" coda-ags quit >/dev/null 2>&1 || true
  if ! coda_wait_ags_gone "${user}" "${CODA_AGS_QUIT_WAIT_S:-5}"; then
    as_desktop "${user}" killall -q ags coda-ags >/dev/null 2>&1 || true
    coda_wait_ags_gone "${user}" "${CODA_AGS_KILL_WAIT_S:-3}" || true
  fi

  while [[ "${tries}" -lt "${max}" ]]; do
    tries=$((tries + 1))
    as_desktop "${user}" coda-ags >/dev/null 2>&1 &
    starter=$!
    if coda_wait_ags_up "${user}"; then
      log "AGS running as ${user} (try ${tries})"
      return 0
    fi
    wait "${starter}" 2>/dev/null || true
    log "AGS start try ${tries}/${max} did not stay up as ${user}"
    sleep "${CODA_AGS_RETRY_S:-0.3}"
  done
  log "AGS failed to stay up as ${user} after ${max} tries"
  log "  runtime=${CODA_DS_RUNTIME} wayland=${CODA_DS_WAYLAND} his=${CODA_DS_SIG:-<none>}"
  return 1
}

copy_hypr_dir() {
  local dest="$1"
  local src="$2"
  install -d "${dest}"
  local f
  for f in "${src}"/*; do
    [[ -f "${f}" ]] || continue
    case "${f}" in
      *.md) continue ;;
    esac
    install -m 0644 "${f}" "${dest}/"
  done
  log "hypr → ${dest}"
}

coda_copy_from_share() {
  local share="$1"
  local hypr_src="" ags_src="" wall_src="" scripts_src="" sc_bin="" local_bin="" helper=""

  [[ -d "${share}" ]] || die "share not mounted or missing: ${share} (mount the 9p tag first)"

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

  if [[ -n "${hypr_src}" ]]; then
    copy_hypr_dir /etc/xdg/hypr "${hypr_src}"
    copy_hypr_dir /root/.config/hypr "${hypr_src}"
    if [[ -d /home/live ]]; then
      copy_hypr_dir /home/live/.config/hypr "${hypr_src}"
      if id live >/dev/null 2>&1; then
        chown -R live:live /home/live/.config/hypr
      fi
    fi
    if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != root ]]; then
      copy_hypr_dir "/home/${SUDO_USER}/.config/hypr" "${hypr_src}"
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

  if [[ -d "${share}/scripts" ]]; then
    scripts_src="${share}/scripts"
  fi
  if [[ -n "${scripts_src}" ]]; then
    install -d /usr/local/bin
    for local_bin in coda-wallpaper coda-hyprpaper coda-hyprland coda-hypr-ws coda-ags coda-settings coda-sandbox coda-install coda-slot; do
      if [[ -f "${scripts_src}/${local_bin}" ]]; then
        install -m 0755 "${scripts_src}/${local_bin}" "/usr/local/bin/${local_bin}"
        log "wrapper → /usr/local/bin/${local_bin}"
      fi
    done
    install -d /usr/local/lib/codalinux
    for helper in coda-install-config.py coda-install-lib.sh coda-install-layout.py \
                  coda-install-ab.sh coda-install-post.sh coda-install-verify.sh; do
      if [[ -f "${scripts_src}/${helper}" ]]; then
        install -m 0755 "${scripts_src}/${helper}" \
          "/usr/local/lib/codalinux/${helper}"
        log "helper → /usr/local/lib/codalinux/${helper}"
      fi
    done
  fi

  for sc_bin in \
    "${share}/core/system-config/bin" \
    "${share}/bin"; do
    [[ -d "${sc_bin}" ]] || continue
    install -d /usr/local/bin /usr/local/lib/codalinux
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
    coda_restart_ags "${user}" || true
  fi
  as_desktop "${user}" hyprctl reload >/dev/null 2>&1 || true
  log "restarted session tools as ${user} (never as root from /root)"
  return 0
}

coda_sync_desktop_main() {
  local share="${1:-${CODA_HOST_SHARE:-/mnt/coda-host}}"
  coda_copy_from_share "${share}"
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
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  coda_sync_desktop_main "${1:-${CODA_HOST_SHARE:-/mnt/coda-host}}"
fi
