#!/usr/bin/env bash
# Smoke assert: live wrappers stay executable in git, airootfs, and
# archiso file_permissions (mkarchiso sets unlisted files to 644).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
failed=0

need_bins=(
  coda-ags
  coda-hypr-ws
  coda-hyprland
  coda-hyprlock
  coda-hyprpaper
  coda-wallpaper
  coda-install
  coda-settings
  coda-sandbox
  coda-sync-desktop-from-host
)

log_fail() { printf 'check-wrapper-modes: %s\n' "$*" >&2; failed=1; }

for bin in "${need_bins[@]}"; do
  overlay="${root}/archiso/airootfs/usr/local/bin/${bin}"
  if [[ ! -f "${overlay}" ]]; then
    log_fail "missing airootfs wrapper: ${overlay}"
    continue
  fi
  if [[ ! -x "${overlay}" ]]; then
    log_fail "not executable (airootfs): ${overlay}"
  fi
  mode="$(stat -c '%a' "${overlay}" 2>/dev/null || stat -f '%OLp' "${overlay}")"
  if [[ "${mode}" != 755 && "${mode}" != 0755 ]]; then
    log_fail "airootfs mode ${mode} (want 755): ${overlay}"
  fi
  if ! grep -qF "[\"/usr/local/bin/${bin}\"]=\"0:0:755\"" \
      "${root}/archiso/profiledef.sh"; then
    log_fail "archiso/profiledef.sh file_permissions missing 755 for /usr/local/bin/${bin}"
  fi
done

for src in coda-ags coda-hyprland coda-hyprlock coda-hyprpaper coda-wallpaper \
           coda-hypr-ws coda-install coda-settings coda-sandbox; do
  f="${root}/scripts/${src}"
  if [[ ! -f "${f}" ]]; then
    log_fail "missing source wrapper: ${f}"
    continue
  fi
  if [[ ! -x "${f}" ]]; then
    log_fail "not executable (scripts/): ${f}"
  fi
done

if ! grep -qx 'swaybg' "${root}/packages/desktop.txt"; then
  log_fail "packages/desktop.txt must list swaybg (live/VM wallpaper fallback)"
fi

if ! grep -qx 'bubblewrap' "${root}/packages/sandbox.txt"; then
  log_fail "packages/sandbox.txt must list bubblewrap (coda-sandbox backbone)"
fi

if [[ "${failed}" -ne 0 ]]; then
  echo "Wrapper modes / profiledef file_permissions / swaybg pin failed." >&2
  exit 1
fi

echo "Wrapper modes OK (airootfs +x, profiledef 755, swaybg listed)."
