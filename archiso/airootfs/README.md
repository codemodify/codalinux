# airootfs overlay

Files here are copied onto the live image **before** packages install. Branding that packages own (`os-release`) is restored by `branding/hooks/`.

`scripts/build-iso.sh` also copies `desktop/`, `sessions/wayland/`, and `branding/` into this tree at build time — do not duplicate those files here.

Service enablement uses systemd presets (`etc/systemd/system-preset/80-codalinux.preset`) plus wants/ symlinks for hosts that do not apply presets on first boot.
