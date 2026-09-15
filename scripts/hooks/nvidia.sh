#!/usr/bin/env bash
# NVIDIA proprietary install path — hook only, no detection yet.
#
# Locked decision: when an NVIDIA GPU is present, install the official
# extra packages in packages/nvidia.txt. Do not add a Coda repo, do not
# enable [multilib] unless a later decision requires lib32-nvidia-utils.
#
# TODO:
#   1. Detect NVIDIA VGA/3D controllers (lspci) on the target machine.
#   2. Distinguish nvidia vs nvidia-open (Turing+ / open kernel modules).
#   3. Install packages/nvidia.txt into the archinstall target or live system.
#   4. Decide whether the live ISO always includes nouveau/Mesa only.
#   5. Wire this script from install/profiles/codalinux.py — after detection
#      exists. Calling it now must be a no-op besides this message.
set -euo pipefail

echo "TODO: NVIDIA auto-detect is not implemented. See packages/nvidia.txt and docs/TODO.md." >&2
exit 0
