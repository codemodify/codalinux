#!/usr/bin/env bash
# Copy the live Coda desktop onto an archinstall target and enable greetd
# for the installed user (default: user). Must run from the live ISO after
# pacstrap — custom_commands are arch-chroot'd and cannot see live /usr/local.
set -euo pipefail

user="${CODA_INSTALL_USER:-user}"
target="${CODA_INSTALL_TARGET:-/mnt}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --user) user="$2"; shift 2 ;;
    --target) target="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: coda-install-post.sh [--user NAME] [--target DIR] [TARGET]"
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

configure_target() {
  local root="$1"
  write_greetd_toml "${root}/etc/greetd"
  rm -f "${root}/etc/systemd/system/coda-live-setup.service"
  rm -f "${root}/etc/systemd/system/multi-user.target.wants/coda-live-setup.service"
  rm -f "${root}/etc/systemd/system/getty@tty2.service.d/autologin.conf"

  install -d "${root}/etc/systemd/system-preset"
  cat >"${root}/etc/systemd/system-preset/80-codalinux.preset" <<'EOF'
enable greetd.service
enable systemd-networkd.service
enable systemd-resolved.service
enable iwd.service
enable bluetooth.service
disable NetworkManager.service
disable firewalld.service
disable cups.service
disable coda-live-setup.service
EOF

  mkdir -p "${root}/etc/skel/.config/hypr"
  if [[ -d "${root}/etc/xdg/hypr" ]]; then
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

  systemctl --root="${root}" enable greetd.service systemd-networkd.service \
    systemd-resolved.service iwd.service bluetooth.service \
    system-config-apply.service || true
  systemctl --root="${root}" disable NetworkManager.service 2>/dev/null || true
  systemctl --root="${root}" disable firewalld.service 2>/dev/null || true
  systemctl --root="${root}" disable cups.service 2>/dev/null || true
  systemctl --root="${root}" disable coda-live-setup.service 2>/dev/null || true
  ln -sfn /usr/lib/systemd/system/greetd.service \
    "${root}/etc/systemd/system/display-manager.service"

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
  log "copying live Coda desktop into ${target} (user=${user})"
  mkdir -p "${target}/usr/local/bin" "${target}/usr/local/lib" "${target}/usr/local/share"

  local_bin=""
  for local_bin in coda-hyprland coda-ags coda-hypr-ws coda-hyprlock \
                   coda-hyprpaper coda-wallpaper coda-settings coda-sandbox \
                   coda-sync-desktop-from-host ags astal \
                   system-config system-configd system-config-apply \
                   system-config-report system-config-tui system-config-gui; do
    copy_if "/usr/local/bin/${local_bin}" "${target}/usr/local/bin/${local_bin}"
  done
  chmod 0755 "${target}/usr/local/bin/"coda-* 2>/dev/null || true
  chmod 0755 "${target}/usr/local/bin/ags" 2>/dev/null || true
  chmod 0755 "${target}/usr/local/bin/astal" 2>/dev/null || true
  chmod 0755 "${target}/usr/local/bin/"system-config* 2>/dev/null || true
  copy_if /usr/local/lib/codalinux/system-config-apply-launch \
    "${target}/usr/local/lib/codalinux/system-config-apply-launch"
  chmod 0755 "${target}/usr/local/lib/codalinux/system-config-apply-launch" 2>/dev/null || true
  copy_if /etc/systemd/system/system-config-apply.service \
    "${target}/etc/systemd/system/system-config-apply.service"
  mkdir -p "${target}/etc/systemd/system/graphical.target.wants" \
    "${target}/etc/systemd/user/graphical-session.target.wants"
  copy_if /etc/systemd/user/system-configd.service \
    "${target}/etc/systemd/user/system-configd.service"
  copy_if /etc/systemd/user/system-config-report.service \
    "${target}/etc/systemd/user/system-config-report.service"
  if [[ -e "${target}/etc/systemd/system/system-config-apply.service" ]]; then
    ln -sfn /etc/systemd/system/system-config-apply.service \
      "${target}/etc/systemd/system/graphical.target.wants/system-config-apply.service"
  fi
  if [[ -e "${target}/etc/systemd/user/system-configd.service" ]]; then
    ln -sfn /etc/systemd/user/system-configd.service \
      "${target}/etc/systemd/user/graphical-session.target.wants/system-configd.service"
  fi
  if [[ -e "${target}/etc/systemd/user/system-config-report.service" ]]; then
    ln -sfn /etc/systemd/user/system-config-report.service \
      "${target}/etc/systemd/user/graphical-session.target.wants/system-config-report.service"
  fi
  copy_if /usr/share/applications/system-config-gui.desktop \
    "${target}/usr/share/applications/system-config-gui.desktop"
  copy_if /usr/share/applications/coda-settings.desktop \
    "${target}/usr/share/applications/coda-settings.desktop"
  rm -f "${target}/etc/systemd/user/default.target.wants/system-configd.service" \
    "${target}/etc/systemd/user/default.target.wants/system-config-report.service"

  copy_tree /usr/local/lib "${target}/usr/local/lib"
  copy_tree /usr/local/share/codalinux "${target}/usr/local/share/codalinux"
  copy_tree /usr/local/share/ags "${target}/usr/local/share/ags"
  copy_tree /usr/local/share/gir-1.0 "${target}/usr/local/share/gir-1.0"
  copy_tree /usr/local/share/glib-2.0 "${target}/usr/local/share/glib-2.0"

  copy_if /etc/ld.so.conf.d/codalinux-usr-local.conf \
    "${target}/etc/ld.so.conf.d/codalinux-usr-local.conf"

  mkdir -p "${target}/usr/share/wayland-sessions"
  copy_if /usr/share/wayland-sessions/codalinux-hyprland.desktop \
    "${target}/usr/share/wayland-sessions/codalinux-hyprland.desktop"

  copy_tree /usr/share/backgrounds/codalinux "${target}/usr/share/backgrounds/codalinux"
  copy_tree /etc/xdg/hypr "${target}/etc/xdg/hypr"
  copy_tree /etc/skel/.config/hypr "${target}/etc/skel/.config/hypr"

  copy_if /etc/pacman.d/hooks/codalinux-os-release.hook \
    "${target}/etc/pacman.d/hooks/codalinux-os-release.hook"
  copy_if /etc/pacman.d/hooks/codalinux-locale.hook \
    "${target}/etc/pacman.d/hooks/codalinux-locale.hook"
  copy_if /usr/local/lib/codalinux/apply-os-release.sh \
    "${target}/usr/local/lib/codalinux/apply-os-release.sh"
  copy_if /usr/local/lib/codalinux/apply-locale.sh \
    "${target}/usr/local/lib/codalinux/apply-locale.sh"
  chmod 0755 "${target}/usr/local/lib/codalinux/"apply-*.sh 2>/dev/null || true

  copy_if /etc/systemd/network/20-wired.network \
    "${target}/etc/systemd/network/20-wired.network"
  copy_if /etc/systemd/network/20-wireless.network \
    "${target}/etc/systemd/network/20-wireless.network"
  copy_if /etc/iwd/main.conf "${target}/etc/iwd/main.conf"
else
  log "running on target root; configuring in place (user=${user})"
fi

configure_target "${target}"

if [[ ! -x "${target}/usr/local/bin/coda-hyprland" ]]; then
  echo "coda-install-post: ${target}/usr/local/bin/coda-hyprland missing after copy" >&2
  exit 1
fi
for sc in system-config system-configd system-config-apply system-config-report system-config-gui; do
  if [[ ! -x "${target}/usr/local/bin/${sc}" ]]; then
    echo "coda-install-post: ${target}/usr/local/bin/${sc} missing (must ship like live)" >&2
    exit 1
  fi
done
if [[ ! -e "${target}/etc/systemd/system/system-config-apply.service" ]]; then
  echo "coda-install-post: system-config-apply.service missing on target" >&2
  exit 1
fi
if [[ ! -e "${target}/etc/systemd/user/system-configd.service" ]]; then
  echo "coda-install-post: system-configd.service missing on target" >&2
  exit 1
fi
if [[ ! -e "${target}/usr/share/applications/system-config-gui.desktop" ]]; then
  echo "coda-install-post: system-config-gui.desktop missing (Settings must ship like live)" >&2
  exit 1
fi
if [[ ! -e "${target}/etc/systemd/user/graphical-session.target.wants/system-configd.service" ]]; then
  echo "coda-install-post: system-configd not wanted by graphical-session.target" >&2
  exit 1
fi
if ! grep -q "user = \"${user}\"" "${target}/etc/greetd/config.toml"; then
  echo "coda-install-post: greetd.toml does not autologin ${user}" >&2
  exit 1
fi
if grep -q 'user = "live"' "${target}/etc/greetd/config.toml"; then
  echo "coda-install-post: greetd.toml still points at live" >&2
  exit 1
fi

log "greetd will autologin ${user} via /usr/local/bin/coda-hyprland"
log "done"
