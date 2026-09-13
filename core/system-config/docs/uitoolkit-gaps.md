# uitoolkit gaps (Coda Settings)

`system-config-gui` ships on uitoolkit `dev` (v0.19.x) by composing existing widgets — same pattern as the Mail dogfood app. No second UI stack. Do not block the GUI on the items below.

## Remaining gaps

- No PrefsPage / NavRail framework
- TableView weak for large sort/filter/column resize
- No shared daemon RPC client in toolkit
- No Form/Grid layout helper

## How we compose instead

| Need | What we use |
| --- | --- |
| Unix socket + JSON-RPC NDJSON | App-level client (`internal/client` + `internal/rpc`). Not a toolkit Socket API. |
| Two-pane Settings + pinned Apply | `Splitter` + `Column.AddFlex` + Primary `Apply` row (Settings dogfood). |
| Sidebar | `Splitter` + `TreeView` (Mail) / `ListView` (Settings). No first-class NavRail. |
| Devices / PCI | `TableView` (alpha: cap huge lists; no toolkit sort/filter). |
| Display scale | `Slider` + `NumberField` + Apply. No built-in scale page. |
| Network / audio / bluetooth / input / datetime / locale / session / power | Same two-pane + `TableView` / `TextField` / `Switch` / `Slider`. No PrefsPage. |
