# coda-sandbox

Day-to-day packages go in a **disposable Arch root**, not on the host. The helper is [`scripts/coda-sandbox`](../scripts/coda-sandbox). Isolation is **upstream [bubblewrap](https://github.com/containers/bubblewrap)** (`bwrap`, **LGPL-2.1-or-later**). CodaLinux does not reimplement bwrap and does not require Docker or Distrobox for v1.

Product rules live in [DESIGN.md](../DESIGN.md#core-desktop-and-sandboxes).

## Commands

```bash
sudo coda-sandbox create db
sudo coda-sandbox install db postgresql
sudo coda-sandbox pacman db -S extra/foo
sudo coda-sandbox enter db
sudo coda-sandbox run db postgres --version
coda-sandbox list
sudo coda-sandbox destroy db --force
coda-sandbox path db
```

| Command | What it does |
| --- | --- |
| `create [name]` | Bootstrap `base` into a new root (`pacstrap` if present, else `pacman --root`) |
| `clone src dst` | Copy an existing sandbox |
| `pacman [name] …` | Pass-through `pacman --root <root> --cachedir <shared cache>` |
| `install [name] pkgs…` | `pacman --noconfirm -S`. Name is used only if that sandbox already exists |
| `enter [name]` | `bwrap` with the sandbox as `/`, then a login shell |
| `run [name] cmd…` | Same namespace, then `cmd` |
| `list` / `path` / `destroy` | Inventory, print root, delete |

Default name is `default`. Persist database files (or other state you must keep after `destroy`) with `--bind HOST:GUEST` or by keeping them under `$HOME` (bound into the sandbox unless `--no-home`).

## Layout

Root (when writable / when running as root):

```
/var/lib/coda/sandboxes/<name>/root    pacman --root tree
/var/lib/coda/sandboxes/<name>/meta
/var/cache/coda/pacman                 shared package cache
```

Otherwise: `~/.local/share/coda/sandboxes` and `~/.cache/coda/pacman`. Override with `--store`, `--cache`, or `CODA_SANDBOX_STORE` / `CODA_SANDBOX_CACHE`.

`enter` / `run` bind the sandbox as `/`, plus `/proc`, `/dev`, read-only `/sys`, host `resolv.conf`, the shared cache at `/var/cache/pacman/pkg`, and `$HOME` by default. Host `/usr` is **not** the sandbox `/usr`.

## Host requirements

- Arch Linux or CodaLinux with official `core`/`extra`, a working keyring, and network for the first `create`
- `bubblewrap` (composed from `packages/sandbox.txt`)
- `pacman`; `pacstrap` (`arch-install-scripts`) is used when present (live ISO has it)
- **root** for `create`, `clone`, `pacman`/`install`, and `destroy` (package extract is uid 0)

`create` does not work from a non-Arch host that lacks pacman.

## What this is not

- Not a read-only A/B core OS. The live/install image is still a mutable desktop root.
- Not Docker, Distrobox, or Firejail.
- Not a Coda pacman repository. Packages stay official Arch names.