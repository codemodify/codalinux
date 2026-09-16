#!/usr/bin/env bash
# Wire greetd + core services after the offline split.
# Desktop/session files go to CODA_DESKTOP_ROOT (/coda/data/desktop).
# Core slot gets sshd/QGA/networkd, coda-desktop-mount, and a real
# greetd.service unit + PAM — not the greetd/Hyprland binaries.
# Runs from the live ISO after coda-install-ab.sh / coda-slot.
set -euo pipefail

_post_here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f /usr/local/lib/codalinux/coda-install-lib.sh ]]; then
  # shellcheck source=coda-install-lib.sh
  . /usr/local/lib/codalinux/coda-install-lib.sh
elif [[ -f "${_post_here}/coda-install-lib.sh" ]]; then
  # shellcheck source=coda-install-lib.sh
  . "${_post_here}/coda-install-lib.sh"
fi

user="${CODA_INSTALL_USER:-user}"
target="${CODA_INSTALL_TARGET:-/mnt}"
desktop_root="${CODA_DESKTOP_ROOT:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --user) user="$2"; shift 2 ;;
    --target) target="$2"; shift 2 ;;
    --desktop) desktop_root="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: coda-install-post.sh [--user NAME] [--target DIR] [--desktop DIR] [TARGET]"
      exit 0
      ;;
    --) shift; break ;;
    -*)
      echo "coda-install-post: unknown option $1" >&2
      exit 1
      ;;
    *)
      target="$1"
      shift
      ;;
  esac
done

log() { printf 'coda-install-post: %s\n' "$*"; }

if [[ ! -d "${target}/usr" || ! -d "${target}/etc" ]]; then
  echo "coda-install-post: ${target} is not an installed root (missing usr/ or etc/)" >&2
  exit 1
fi

same_root=0
if [[ "$(stat -c '%d:%i' / 2>/dev/null || echo x)" == "$(stat -c '%d:%i' "${target}" 2>/dev/null || echo y)" ]]; then
  same_root=1
fi

copy_if() {
  local src="$1" dest="$2"
  if [[ -e "${src}" || -L "${src}" ]]; then
    mkdir -p "$(dirname "${dest}")"
    cp -a "${src}" "${dest}"
  fi
}

copy_tree() {
  local src="$1" dest="$2"
  if [[ -d "${src}" ]]; then
    mkdir -p "${dest}"
    cp -a "${src}/." "${dest}/"
  fi
}

install_hypr_configs() {
  local dest="$1"
  mkdir -p "${dest}"
  local src
  for src in /etc/xdg/hypr /etc/skel/.config/hypr; do
    if [[ -d "${src}" ]]; then
      cp -a "${src}/." "${dest}/"
    fi
  done
}

write_greetd_toml() {
  local dest="$1"
  mkdir -p "${dest}"
  cat >"${dest}/config.toml" <<EOF
# Installed system: autologin ${user} into the same session as live.
# Do not point greetd at the live ISO user.

[terminal]
vt = 1

[default_session]
command = "/usr/local/bin/coda-hyprland"
user = "${user}"

[initial_session]
command = "/usr/local/bin/coda-hyprland"
user = "${user}"
EOF
}

session_root() {
  if [[ -n "${desktop_root}" ]]; then
    printf '%s' "${desktop_root}"
  else
    printf '%s' "${target}"
  fi
}

configure_target() {
  local root="$1"
  local session
  session="$(session_root)"
  write_greetd_toml "${session}/etc/greetd"
  if [[ "${session}" != "${root}" ]]; then
    write_greetd_toml "${root}/etc/greetd"
  fi
  rm -f "${root}/etc/systemd/system/coda-live-setup.service"
  rm -f "${root}/etc/systemd/system/multi-user.target.wants/coda-live-setup.service"
  rm -f "${root}/etc/systemd/system/getty@tty2.service.d/autologin.conf"
  if declare -F coda_scrub_live_greetd >/dev/null; then
    coda_scrub_live_greetd "${root}"
    if [[ "${session}" != "${root}" ]]; then
      coda_scrub_live_greetd "${session}"
    fi
  fi

  install -d "${root}/etc/systemd/system-preset"
  cat >"${root}/etc/systemd/system-preset/80-codalinux.preset" <<'EOF'
enable greetd.service
enable systemd-networkd.service
enable systemd-resolved.service
enable iwd.service
enable bluetooth.service
enable qemu-guest-agent.service
enable sshd.service
disable NetworkManager.service
disable firewalld.service
disable cups.service
disable coda-live-setup.service
EOF

  mkdir -p "${session}/etc/skel/.config/hypr" "${root}/etc/skel/.config/hypr"
  if [[ -d "${session}/etc/xdg/hypr" ]]; then
    cp -a "${session}/etc/xdg/hypr/." "${session}/etc/skel/.config/hypr/"
  elif [[ -d "${root}/etc/xdg/hypr" ]]; then
    cp -a "${root}/etc/xdg/hypr/." "${root}/etc/skel/.config/hypr/"
  fi

  if [[ -d "${root}/home/${user}" ]]; then
    mkdir -p "${root}/home/${user}/.config/hypr"
    if [[ -d "${root}/etc/xdg/hypr" ]]; then
      cp -a "${root}/etc/xdg/hypr/." "${root}/home/${user}/.config/hypr/"
    elif [[ -d "${root}/etc/skel/.config/hypr" ]]; then
      cp -a "${root}/etc/skel/.config/hypr/." "${root}/home/${user}/.config/hypr/"
    fi
    if command -v arch-chroot >/dev/null 2>&1 && [[ "${same_root}" -eq 0 ]]; then
      arch-chroot -S "${root}" chown -R "${user}:${user}" "/home/${user}" || true
    elif [[ "${same_root}" -eq 1 ]]; then
      chown -R "${user}:${user}" "${root}/home/${user}" || true
    fi
  fi

  # greetd.service + PAM on the slot so systemd can start the DM after
  # coda-desktop-mount. The greetd binary stays on coda-data.
  if declare -F coda_install_slot_greetd >/dev/null; then
    coda_install_slot_greetd "${root}"
  else
    echo "coda-install-post: coda_install_slot_greetd missing" >&2
    exit 1
  fi

  # Core units exist on the slot. Desktop units (iwd/bluetooth) exist
  # after coda-desktop-mount; enable them as wants symlinks anyway.
  systemctl --root="${root}" enable systemd-networkd.service \
    systemd-resolved.service qemu-guest-agent.service sshd.service || true
  systemctl --root="${root}" enable greetd.service iwd.service \
    bluetooth.service system-config-apply.service 2>/dev/null || true
  systemctl --root="${root}" disable NetworkManager.service 2>/dev/null || true
  systemctl --root="${root}" disable firewalld.service 2>/dev/null || true
  systemctl --root="${root}" disable cups.service 2>/dev/null || true
  systemctl --root="${root}" disable coda-live-setup.service 2>/dev/null || true
  mkdir -p "${root}/etc/systemd/system/multi-user.target.wants"
  ln -sfn /usr/lib/systemd/system/iwd.service \
    "${root}/etc/systemd/system/multi-user.target.wants/iwd.service" 2>/dev/null || true
  ln -sfn /usr/lib/systemd/system/bluetooth.service \
    "${root}/etc/systemd/system/multi-user.target.wants/bluetooth.service" 2>/dev/null || true

  if [[ -x "${root}/usr/local/lib/codalinux/apply-os-release.sh" ]]; then
    if [[ "${same_root}" -eq 1 ]]; then
      "${root}/usr/local/lib/codalinux/apply-os-release.sh" || true
    elif command -v arch-chroot >/dev/null 2>&1; then
      arch-chroot -S "${root}" /usr/local/lib/codalinux/apply-os-release.sh || true
    fi
  fi

  if command -v arch-chroot >/dev/null 2>&1 && [[ "${same_root}" -eq 0 ]]; then
    arch-chroot -S "${root}" usermod -aG wheel,video,audio,input,render,storage,lp "${user}" || true
    if [[ -x "${root}/usr/sbin/ldconfig" || -x "${root}/usr/bin/ldconfig" ]]; then
      arch-chroot -S "${root}" ldconfig -X || true
    fi
  elif [[ "${same_root}" -eq 1 ]]; then
    usermod -aG wheel,video,audio,input,render,storage,lp "${user}" || true
    if command -v ldconfig >/dev/null 2>&1; then
      ldconfig -X || true
    fi
  fi

  if [[ ! -s "${root}/etc/ld.so.conf.d/codalinux-usr-local.conf" ]]; then
    mkdir -p "${root}/etc/ld.so.conf.d"
    printf '%s\n' /usr/local/lib >"${root}/etc/ld.so.conf.d/codalinux-usr-local.conf"
  fi
}

if [[ "${same_root}" -eq 0 ]]; then
  session="$(session_root)"
  log "copying live Coda desktop into ${session} (user=${user}; slot=${target})"
  mkdir -p "${session}/usr/local/bin" "${session}/usr/local/lib" "${session}/usr/local/share"
  mkdir -p "${target}/usr/local/bin" "${target}/usr/local/lib/codalinux"

  local_bin=""
  for local_bin in coda-hyprland coda-ags coda-hypr-ws coda-hyprlock \
                   coda-hyprpaper coda-wallpaper coda-settings coda-sandbox \
                   coda-sync-desktop-from-host ags astal \
                   system-config system-configd system-config-apply \
                   system-config-report system-config-tui system-config-gui; do
    copy_if "/usr/local/bin/${local_bin}" "${session}/usr/local/bin/${local_bin}"
  done
  chmod 0755 "${session}/usr/local/bin/"coda-* 2>/dev/null || true
  chmod 0755 "${session}/usr/local/bin/ags" 2>/dev/null || true
  chmod 0755 "${session}/usr/local/bin/astal" 2>/dev/null || true
  chmod 0755 "${session}/usr/local/bin/"system-config* 2>/dev/null || true
  copy_if /usr/local/lib/codalinux/system-config-gui \
    "${session}/usr/local/lib/codalinux/system-config-gui"
  copy_if /usr/local/lib/codalinux/system-config-apply-launch \
    "${session}/usr/local/lib/codalinux/system-config-apply-launch"
  chmod 0755 "${session}/usr/local/lib/codalinux/system-config-gui" 2>/dev/null || true
  chmod 0755 "${session}/usr/local/lib/codalinux/system-config-apply-launch" 2>/dev/null || true
  copy_if /etc/systemd/system/system-config-apply.service \
    "${session}/etc/systemd/system/system-config-apply.service"
  mkdir -p "${session}/etc/systemd/system/graphical.target.wants" \
    "${session}/etc/systemd/user/graphical-session.target.wants" \
    "${target}/etc/systemd/system/graphical.target.wants" \
    "${target}/etc/systemd/user/graphical-session.target.wants"
  copy_if /etc/systemd/user/system-configd.service \
    "${session}/etc/systemd/user/system-configd.service"
  copy_if /etc/systemd/user/system-config-report.service \
    "${session}/etc/systemd/user/system-config-report.service"
  if [[ -e "${session}/etc/systemd/system/system-config-apply.service" ]]; then
    ln -sfn /etc/systemd/system/system-config-apply.service \
      "${session}/etc/systemd/system/graphical.target.wants/system-config-apply.service"
    ln -sfn /etc/systemd/system/system-config-apply.service \
      "${target}/etc/systemd/system/graphical.target.wants/system-config-apply.service"
  fi
  if [[ -e "${session}/etc/systemd/user/system-configd.service" ]]; then
    ln -sfn /etc/systemd/user/system-configd.service \
      "${session}/etc/systemd/user/graphical-session.target.wants/system-configd.service"
    ln -sfn /etc/systemd/user/system-configd.service \
      "${target}/etc/systemd/user/graphical-session.target.wants/system-configd.service"
  fi
  if [[ -e "${session}/etc/systemd/user/system-config-report.service" ]]; then
    ln -sfn /etc/systemd/user/system-config-report.service \
      "${session}/etc/systemd/user/graphical-session.target.wants/system-config-report.service"
    ln -sfn /etc/systemd/user/system-config-report.service \
      "${target}/etc/systemd/user/graphical-session.target.wants/system-config-report.service"
  fi
  copy_if /usr/share/applications/system-config-gui.desktop \
    "${session}/usr/share/applications/system-config-gui.desktop"
  copy_if /usr/share/applications/coda-settings.desktop \
    "${session}/usr/share/applications/coda-settings.desktop"
  rm -f "${session}/etc/systemd/user/default.target.wants/system-configd.service" \
    "${session}/etc/systemd/user/default.target.wants/system-config-report.service"

  copy_tree /usr/local/lib "${session}/usr/local/lib"
  copy_tree /usr/local/share/codalinux "${session}/usr/local/share/codalinux"
  copy_tree /usr/local/share/ags "${session}/usr/local/share/ags"
  copy_tree /usr/local/share/gir-1.0 "${session}/usr/local/share/gir-1.0"
  copy_tree /usr/local/share/glib-2.0 "${session}/usr/local/share/glib-2.0"

  copy_if /etc/ld.so.conf.d/codalinux-usr-local.conf \
    "${session}/etc/ld.so.conf.d/codalinux-usr-local.conf"

  mkdir -p "${session}/usr/share/wayland-sessions"
  copy_if /usr/share/wayland-sessions/codalinux-hyprland.desktop \
    "${session}/usr/share/wayland-sessions/codalinux-hyprland.desktop"

  copy_tree /usr/share/backgrounds/codalinux "${session}/usr/share/backgrounds/codalinux"
  copy_tree /etc/xdg/hypr "${session}/etc/xdg/hypr"
  copy_tree /etc/skel/.config/hypr "${session}/etc/skel/.config/hypr"

  # Core-slot bits (installer + branding + ssh/networkd).
  copy_if /etc/pacman.d/hooks/codalinux-os-release.hook \
    "${target}/etc/pacman.d/hooks/codalinux-os-release.hook"
  copy_if /etc/pacman.d/hooks/codalinux-locale.hook \
    "${target}/etc/pacman.d/hooks/codalinux-locale.hook"
  copy_if /usr/local/lib/codalinux/apply-os-release.sh \
    "${target}/usr/local/lib/codalinux/apply-os-release.sh"
  copy_if /usr/local/lib/codalinux/apply-locale.sh \
    "${target}/usr/local/lib/codalinux/apply-locale.sh"
  chmod 0755 "${target}/usr/local/lib/codalinux/"apply-*.sh 2>/dev/null || true
  for core_bin in coda-install coda-slot; do
    copy_if "/usr/local/bin/${core_bin}" "${target}/usr/local/bin/${core_bin}"
  done
  copy_if /usr/local/lib/codalinux/coda-desktop-mount \
    "${target}/usr/local/lib/codalinux/coda-desktop-mount"
  for helper in coda-install-lib.sh coda-install-post.sh coda-install-ab.sh \
                coda-install-split.py coda-install-layout.py \
                coda-install-verify.sh coda-install-config.py; do
    copy_if "/usr/local/lib/codalinux/${helper}" \
      "${target}/usr/local/lib/codalinux/${helper}"
  done
  chmod 0755 "${target}/usr/local/bin/coda-install" 2>/dev/null || true
  chmod 0755 "${target}/usr/local/bin/coda-slot" 2>/dev/null || true
  chmod 0755 "${target}/usr/local/lib/codalinux/coda-desktop-mount" 2>/dev/null || true
  chmod 0755 "${target}/usr/local/lib/codalinux/"coda-install* 2>/dev/null || true

  copy_if /etc/systemd/network/20-wired.network \
    "${target}/etc/systemd/network/20-wired.network"
  copy_if /etc/systemd/network/20-wireless.network \
    "${target}/etc/systemd/network/20-wireless.network"
  copy_if /etc/iwd/main.conf "${session}/etc/iwd/main.conf"
  copy_if /etc/ssh/sshd_config.d/10-codalinux.conf \
    "${target}/etc/ssh/sshd_config.d/10-codalinux.conf"
else
  log "running on target root; configuring in place (user=${user})"
fi

configure_target "${target}"

session="$(session_root)"
if [[ ! -x "${session}/usr/local/bin/coda-hyprland" ]]; then
  echo "coda-install-post: ${session}/usr/local/bin/coda-hyprland missing after copy" >&2
  exit 1
fi
for sc in system-config system-configd system-config-apply system-config-report system-config-gui; do
  if [[ ! -x "${session}/usr/local/bin/${sc}" ]]; then
    echo "coda-install-post: ${session}/usr/local/bin/${sc} missing (must ship like live)" >&2
    exit 1
  fi
done
if [[ ! -e "${session}/etc/systemd/system/system-config-apply.service" ]]; then
  echo "coda-install-post: system-config-apply.service missing on desktop payload" >&2
  exit 1
fi
if [[ ! -e "${session}/etc/systemd/user/system-configd.service" ]]; then
  echo "coda-install-post: system-configd.service missing on desktop payload" >&2
  exit 1
fi
if [[ ! -x "${session}/usr/local/lib/codalinux/system-config-gui" ]]; then
  echo "coda-install-post: CGO system-config-gui binary missing (must ship like live)" >&2
  exit 1
fi
if [[ ! -e "${session}/usr/share/applications/system-config-gui.desktop" ]]; then
  echo "coda-install-post: system-config-gui.desktop missing (Settings must ship like live)" >&2
  exit 1
fi
if [[ ! -e "${target}/etc/systemd/user/graphical-session.target.wants/system-configd.service" \
   && ! -e "${session}/etc/systemd/user/graphical-session.target.wants/system-configd.service" ]]; then
  echo "coda-install-post: system-configd not wanted by graphical-session.target" >&2
  exit 1
fi
if ! grep -q "user = \"${user}\"" "${session}/etc/greetd/config.toml"; then
  echo "coda-install-post: greetd.toml does not autologin ${user}" >&2
  exit 1
fi
if grep -q 'user = "live"' "${session}/etc/greetd/config.toml"; then
  echo "coda-install-post: greetd.toml still points at live" >&2
  exit 1
fi
if [[ -n "${desktop_root}" && -x "${target}/usr/local/bin/coda-hyprland" ]]; then
  echo "coda-install-post: coda-hyprland must not be installed on the core slot" >&2
  exit 1
fi
if [[ ! -f "${target}/usr/local/lib/codalinux/coda-install-lib.sh" ]]; then
  echo "coda-install-post: coda-install-lib.sh missing on the core slot (boot-test/promote need it)" >&2
  exit 1
fi
if [[ ! -f "${target}/etc/systemd/system/greetd.service" \
   || -L "${target}/etc/systemd/system/greetd.service" ]]; then
  echo "coda-install-post: greetd.service must be a real file on the core slot" >&2
  exit 1
fi
if [[ ! -f "${target}/etc/pam.d/greetd" ]]; then
  echo "coda-install-post: /etc/pam.d/greetd missing on the core slot" >&2
  exit 1
fi
if [[ ! -f "${target}/etc/systemd/system/greetd.service.d/coda-desktop-mount.conf" ]]; then
  echo "coda-install-post: greetd drop-in After=coda-desktop-mount missing" >&2
  exit 1
fi
_greetd_want="${target}/etc/systemd/system/multi-user.target.wants/greetd.service"
if [[ -L "${_greetd_want}" ]]; then
  _greetd_link="$(readlink "${_greetd_want}")"
  if [[ "${_greetd_link}" == /usr/lib/systemd/system/greetd.service ]]; then
    echo "coda-install-post: greetd wants must not dangle at /usr/lib (desktop-only)" >&2
    exit 1
  fi
elif [[ ! -e "${_greetd_want}" ]]; then
  echo "coda-install-post: greetd.service is not wanted on the core slot" >&2
  exit 1
fi

log "greetd will autologin ${user} via /usr/local/bin/coda-hyprland"
log "enabled qemu-guest-agent.service and sshd.service"
log "done"
