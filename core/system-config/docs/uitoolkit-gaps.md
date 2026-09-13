# uitoolkit gaps (Coda Settings)

`system-config-gui` is built only with [`github.com/codemodify/uitoolkit@dev`](https://github.com/codemodify/uitoolkit). No second UI stack.

## What we used (present)

- `New` / `NewWindow` / `Run` / `WritePNG` (headless)
- `NewTitleBar`, `NewStatusBar`, `NewColumn`, `NewRow`, `NewPad`, `NewSplitter`
- `NewListView` (sidebar)
- `NewTableView` (display outputs, PCI table)
- `NewTitle`, `NewLabel`, `NewButton`

That is enough for a two-page Settings shell (Display + Devices).

## Gaps that would make Settings more complete

None of these blocked a first GUI. They are the next toolkit asks:

1. **Live data binding / invalidate-from-goroutine** — pages rebuild with `SetContent` after Refresh/Apply. A `Invalidate`/`Rebuild` helper that is safe from a `watch` callback would avoid tearing down the sidebar selection.
2. **Table cell widgets** — `TableView` is string-only. Scale as in-cell radio/combo would need buttons below the table (what we do) or cells that host `Component`s.
3. **Nav rail / settings category widget** — we compose `ListView` + `Splitter`. A dedicated nav with icons + sections would match a desktop Settings app more closely.
4. **Form / property grid** — locale and future network pages are key/value forms; today that is labels + fields stacked in a column.
5. **Watch / event source** — protocol `watch` is a one-shot snapshot. Toolkit does not need this, but a timer or fd-watch helper on the app loop would make udev-driven Devices updates cleaner than rebuild-on-button.

## Not a toolkit gap

- TUI is `system-config-tui` (text stub). uitoolkit is the GUI toolkit, not a terminal widget set.
- Display apply still needs Hyprland + `system-config-apply`; the GUI only talks to D.
