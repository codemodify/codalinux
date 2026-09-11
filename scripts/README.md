# Scripts

| Script | Purpose |
| --- | --- |
| `compose-package-lists.sh` | Build `archiso/packages.x86_64`, `install/packages.txt`, and the `packages` array in `user_configuration.json` (needs `python3`) |
| `check-package-lists.sh` | Reject forbidden / unofficial names |
| `build-iso.sh` | Documented `mkarchiso` wrapper (Arch host required) |
| `hooks/nvidia.sh` | NVIDIA detect/install placeholder (no-op) |

Run compose + check after editing `packages/*.txt`.
