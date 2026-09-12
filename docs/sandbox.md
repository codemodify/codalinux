# coda-sandbox

Day-to-day packages go in a **user-owned disposable Arch root**, not on the host. The helper is [`scripts/coda-sandbox`](../scripts/coda-sandbox). Isolation is **upstream [bubblewrap](https://github.com/containers/bubblewrap)** (`bwrap`, **LGPL-2.1-or-later**). CodaLinux does not reimplement bwrap and does not require Docker, Distrobox, or **sudo**.

Product rules live in [DESIGN.md](../DESIGN.md#core-desktop-and-sandboxes).

## Commands

```bash
coda-sandbox create dev
coda-sandbox install dev postgresql
coda-sandbox install dev redis git
coda-sandbox pacman dev -S extra/foo
coda-sandbox enter dev
coda-sandbox run dev postgres --version
coda-sandbox list
coda-sandbox destroy dev --force
coda-sandbox path dev
```

| Command | What it does |
| --- | --- |
| `create [name]` | Bootstrap `base` with `pacman --root` in a user namespace |
| `clone src dst` | Copy an existing sandbox |
| `pacman [name] …` | Pass-through `pacman --root` (also via user namespace) |
| `install [name] pkgs…` | `pacman --noconfirm -S`. Name is used only if that sandbox already exists |
| `enter [name]` | `bwrap` with the sandbox as `/`, then a login shell |
| `run [name] cmd…` | Same namespace, then `cmd` |
| `list` / `path` / `destroy` | Inventory, print root, delete |

Default name is `default`. Persist database files (or other state you must keep after `destroy`) with `--bind HOST:GUEST` or by keeping them under `$HOME` (bound unless `--no-home`).

## One root, many packages

A named sandbox is **one Arch root**, not one app. `~/.coda/sandbox/dev` can hold `postgresql`, `redis`, `git`, and anything else you `coda-sandbox install dev …` / `coda-sandbox pacman dev …` into it. Create a second name only when you want a **separate** disposable root (different experiment, throwaway vs long-lived). Do not assume a 1:1 app↔sandbox mapping.

## Layout

Documented path: **`~/.coda/sandbox/<name>`** (singular `sandbox`). The name directory is the sandbox; the pacman tree lives inside it:

```
~/.coda/sandbox/<name>/         sandbox directory
~/.coda/sandbox/<name>/root     pacman --root tree
~/.coda/sandbox/<name>/meta
~/.coda/cache/pacman            shared cache for this user
```

Not `/var/coda/sandbox`, not `/var/lib/coda/…`, and not XDG `~/.local/share/…` as the primary path. `~/.coda` is created on first use (`0700`). The pacman cache is **shared across this user’s named roots** (one download of `base`, many installs into one or more names). Override with `--store` / `--cache` or `CODA_SANDBOX_STORE` / `CODA_SANDBOX_CACHE`.

`enter` / `run` bind the sandbox as `/`, plus `/proc`, `/dev`, read-only `/sys`, host `resolv.conf`, `~/.coda/cache/pacman` at `/var/cache/pacman/pkg` inside the sandbox, and `$HOME` by default.

## How user-namespace pacman works

Host `pacman --root` wants to extract as uid 0. Instead of sudo, `coda-sandbox` runs:

```text
unshare --user --map-root-user [--keep-caps] -- pacman --root <tree> …
```

Inside that namespace the user **is** root, so `chown`/`mknod` during extract succeed. On the host, ns uid 0 maps to the real uid, so every file is owned by the creating user. The host keyring is copied into `~/.coda/cache/pacman/gnupg-host` so pacman does not write `/etc/pacman.d/gnupg`. `pacstrap` is not used (it expects a real root).

Caveats:

- Unprivileged user namespaces must be allowed (`max_user_namespaces` > 0; some kernels also have `kernel.unprivileged_userns_clone=1`).
- Only the caller’s uid is mapped. Package files that want *other* system uids may warn or land as the overflow uid; the tree is still user-owned and disposable.
- Setuid bits inside the sandbox do not grant host root.
- `enter`/`run` stay as the real user (no uid remap) so those user-owned files stay writable.
- The host must be Arch/CodaLinux with `pacman`. A Debian/Ubuntu cloud agent can exercise `unshare` and layout paths, not a full `create` of `base`.

## Host requirements

- Arch Linux or CodaLinux, official `core`/`extra`, working keyring, network for the first `create`
- `bubblewrap` (`packages/sandbox.txt`) and `unshare` (`util-linux`, already in `base.txt`)
- Unprivileged user namespaces

## What this is not

- Not a read-only A/B core OS. The live/install image is still a mutable desktop root.
- Not Docker, Distrobox, or Firejail.
- Not a Coda pacman repository. Packages stay official Arch names.
- Not a sudo wrapper.
