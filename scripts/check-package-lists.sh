#!/usr/bin/env bash
# Reject policy violations in packages/*.txt (AUR helpers, unofficial
# desktop bits, NetworkManager, firewalls, Calamares, Plymouth).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
failed=0

# Names that must never appear as package list tokens.
forbidden_regex='^(yay|paru|pikaur|pamac|pamac-aur|calamares|networkmanager|network-manager-applet|firewalld|ufw|plymouth|aylurs-gtk-shell|aylurs-gtk-shell-git|libastal|libastal-git|libastal-meta|xlibre-xserver|xlibre-meta|xlibre-xserver-git)$'

while IFS= read -r -d '' file; do
  while IFS= read -r raw; do
    line="${raw%%#*}"
    line="$(echo "${line}" | tr -d '[:space:]')"
    [[ -z "${line}" ]] && continue
    if echo "${line}" | grep -Eq "${forbidden_regex}"; then
      echo "forbidden package in ${file#"${root}"/}: ${line}" >&2
      failed=1
    fi
  done < "${file}"
done < <(find "${root}/packages" -type f -name '*.txt' -print0 | sort -z)

if [[ "${failed}" -ne 0 ]]; then
  echo "Package lists must stay official-Arch-repos-only. See DESIGN.md." >&2
  exit 1
fi

echo "Package lists OK (no forbidden names)."
