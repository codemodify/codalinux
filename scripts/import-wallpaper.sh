#!/usr/bin/env bash
# Copy a host still into branding/wallpapers/default.png.
# Default source is the Plasma Horos 5K file the live wallpaper came from.
# Never runs gen-wallpaper.py and never overwrites unless --force.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dest="${root}/branding/wallpapers/default.png"
force=0
src="${CODA_WALLPAPER_SRC:-/usr/share/wallpapers/Horos/contents/images/5120x2880.png}"

usage() {
  cat <<EOF
Usage: import-wallpaper.sh [--force] [SOURCE.png]
Copy SOURCE into branding/wallpapers/default.png.
Default SOURCE is ${src}
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --force) force=1; shift ;;
    -h|--help) usage; exit 0 ;;
    --) shift; break ;;
    -*) echo "unknown option: $1" >&2; usage >&2; exit 1 ;;
    *) src="$1"; shift ;;
  esac
done

if [[ ! -f "${src}" ]]; then
  echo "import-wallpaper: missing ${src}" >&2
  echo "The chat attachment was not persisted here; pass the Horos PNG path." >&2
  exit 1
fi
if [[ -f "${dest}" && "${force}" -ne 1 ]]; then
  echo "import-wallpaper: ${dest} exists (use --force to replace a real asset)" >&2
  exit 1
fi

install -d "$(dirname "${dest}")"
install -m 0644 "${src}" "${dest}"
echo "import-wallpaper: installed ${src} -> ${dest} ($(wc -c < "${dest}") bytes)"
