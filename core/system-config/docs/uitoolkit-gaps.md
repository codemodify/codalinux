# uitoolkit gaps (Coda Settings)

`system-config-gui` ships on uitoolkit `dev` (v0.19.x, pin `134847f`) by composing existing widgets. No second UI stack. Do not block the GUI on the items below.

Inspected uitoolkit `@dev` tip `f0d7673` (2026-09-16): **no PrefsPage / NavRail** (or equivalents) in `export.go` / `widgets/`. The toolkit Settings dogfood (`cmd/uitksettings`) is itself composed: `ListView` + `Splitter` + title chrome. Tip `@dev` also wants `paintengine2d@v0.11.0`, which is not a published tag (latest is `v0.10.0`), so this tree stays on the working pin.

## What we composed (this pass)

| Need | What we use |
| --- | --- |
| Settings shell | `TitleBar` + `Splitter` + `ListView` rail + per-section page chrome. Not Mail `TreeView`. |
| Nav rail feel | `ListView` of grouped sections (Hardware / Connectivity / Session / System). Dirty sections append `•`. No first-class NavRail; current pin also lacks `ListView.Sidebar`. |
| Prefs page chrome | Shared header (title, group, blurb, status chip) + `ScrollView` body + pinned footer (`Apply` + `Refresh` + per-section hint). `NewPanel` groups fields on each page. |
| Per-section Apply | Unchanged: dirty vs that section’s baseline; nav does not arm Apply. Devices is Refresh only. |
| Unix socket + JSON-RPC NDJSON | App-level client (`internal/client` + `internal/rpc`). Not a toolkit Socket API. |
| Display scale | `Slider` + `NumberField` + Apply. No built-in scale page. |
| Network / audio / bluetooth / input / datetime / locale / session / power / printers / users / storage | Same composed shell + `TableView` / `TextField` / `Switch` / `Slider`. Bluetooth PIN is an `Overlay` + `NewPanel` dialog (`Pair…`). |

## Still toolkit-blocked

- First-class PrefsPage / NavRail (or `ListView.Sidebar` + `NewForm` on a pin we can take). Tip `@dev` has `ListView.Sidebar` and `NewForm`/`NewGrid`, but the module is not consumable until `paintengine2d@v0.11.0` exists.
- TableView weak for large sort/filter/column resize
- No shared daemon RPC client in toolkit
