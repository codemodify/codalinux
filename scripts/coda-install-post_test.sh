#!/usr/bin/env bash
# Host-safe: top-level `local` is a runtime error (bash -n does not catch it).
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
post="${root}/scripts/coda-install-post.sh"
fail=0

if ! bash -n "${post}"; then
  echo "coda-install-post_test: bash -n failed" >&2
  exit 1
fi

python3 - "${post}" <<'PY' || fail=1
import re, sys

text = open(sys.argv[1], encoding="utf-8").read().splitlines()
depth = 0
bad = []
for i, line in enumerate(text, 1):
    stripped = line.split("#", 1)[0]
    if re.match(r"^\s*\w+\s*\(\)\s*\{", stripped) or re.match(
        r"^\s*function\s+\w+", stripped
    ):
        depth += stripped.count("{") - stripped.count("}")
        if depth < 0:
            depth = 0
        continue
    opens = stripped.count("{")
    closes = stripped.count("}")
    if depth == 0 and re.search(r"(^|[;&|({])\s*local\s", stripped):
        bad.append(f"{i}:{line.rstrip()}")
    depth += opens - closes
    if depth < 0:
        depth = 0
if bad:
    print("coda-install-post_test: top-level local is a runtime error:", file=sys.stderr)
    for row in bad:
        print(f"  {row}", file=sys.stderr)
    raise SystemExit(1)
print("coda-install-post_test: no top-level local")
PY

# Helper-copy loop must run at script scope (same_root=0 path).
work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT
mkdir -p "${work}/usr/local/lib/codalinux" "${work}/slot/usr/local/lib/codalinux"
for helper in coda-install-lib.sh coda-install-post.sh coda-install-ab.sh \
              coda-install-split.py coda-install-layout.py \
              coda-install-verify.sh coda-install-config.py; do
  printf 'ok\n' >"${work}/usr/local/lib/codalinux/${helper}"
done
# shellcheck disable=SC2164
cd "${work}"
for helper in coda-install-lib.sh coda-install-post.sh coda-install-ab.sh \
              coda-install-split.py coda-install-layout.py \
              coda-install-verify.sh coda-install-config.py; do
  cp -a "usr/local/lib/codalinux/${helper}" \
    "slot/usr/local/lib/codalinux/${helper}"
done
for helper in coda-install-lib.sh coda-install-post.sh; do
  if [[ ! -f "${work}/slot/usr/local/lib/codalinux/${helper}" ]]; then
    echo "coda-install-post_test: helper copy missed ${helper}" >&2
    fail=1
  fi
done

# Same-inode helper copy must not empty the live file (overlay/bind alias).
# shellcheck source=coda-install-lib.sh
. "${root}/scripts/coda-install-lib.sh"
same="${work}/same"
mkdir -p "${same}"
printf 'coda_need_root() { :; }\nLIVE\n' >"${same}/coda-install-lib.sh"
ln -f "${same}/coda-install-lib.sh" "${same}/dest-lib.sh"
coda_copy_file_safe "${same}/coda-install-lib.sh" "${same}/dest-lib.sh"
if ! grep -q LIVE "${same}/coda-install-lib.sh"; then
  echo "coda-install-post_test: same-inode copy emptied the live helper" >&2
  fail=1
fi

# Emptied live file + snapshot must restore helpers for the next coda-slot.
snap="${work}/snap"
mkdir -p "${snap}"
cp -a "${root}/scripts/coda-install-lib.sh" "${snap}/coda-install-lib.sh"
emptied="${work}/emptied-live.sh"
: >"${emptied}"
coda_copy_file_safe "${snap}/coda-install-lib.sh" "${emptied}"
if ! grep -q 'coda_need_root()' "${emptied}"; then
  echo "coda-install-post_test: snapshot restore missed coda_need_root" >&2
  fail=1
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "coda-install-post_test: FAILED" >&2
  exit 1
fi
echo "coda-install-post_test: OK"
exit 0
