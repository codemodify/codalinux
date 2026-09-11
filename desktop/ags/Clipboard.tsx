import { For, createState } from "ags"
import app from "ags/gtk4/app"
import { execAsync } from "ags/process"
import { Astal, Gtk, Gdk } from "ags/gtk4"
import Graphene from "gi://Graphene"
import Pango from "gi://Pango"

const { TOP, BOTTOM, LEFT, RIGHT } = Astal.WindowAnchor

export default function Clipboard() {
  let contentbox: Gtk.Box
  let win: Astal.Window
  const [items, setItems] = createState(new Array<string>())

  function refresh() {
    execAsync(["cliphist", "list"])
      .then((out) =>
        setItems(
          String(out)
            .split("\n")
            .map((line) => line.trim())
            .filter(Boolean)
            .slice(0, 16),
        ),
      )
      .catch(() => setItems([]))
  }

  function paste(line: string) {
    win.visible = false
    execAsync([
      "bash",
      "-lc",
      `printf '%s\\n' "$1" | cliphist decode | wl-copy`,
      "coda-clip",
      line,
    ]).catch(console.error)
  }

  function onKey(_e: Gtk.EventControllerKey, keyval: number) {
    if (keyval === Gdk.KEY_Escape) win.visible = false
  }

  function onClick(_e: Gtk.GestureClick, _: number, x: number, y: number) {
    const [, rect] = contentbox.compute_bounds(win)
    const position = new Graphene.Point({ x, y })
    if (!rect.contains_point(position)) {
      win.visible = false
      return true
    }
  }

  return (
    <window
      $={(self) => (win = self)}
      name="clipboard"
      class="Clipboard"
      visible={false}
      anchor={TOP | BOTTOM | LEFT | RIGHT}
      exclusivity={Astal.Exclusivity.IGNORE}
      keymode={Astal.Keymode.EXCLUSIVE}
      application={app}
      onNotifyVisible={({ visible }) => {
        if (visible) refresh()
      }}
    >
      <Gtk.EventControllerKey onKeyPressed={onKey} />
      <Gtk.GestureClick onPressed={onClick} />
      <box
        $={(self) => (contentbox = self)}
        class="clipboard-content"
        valign={Gtk.Align.CENTER}
        halign={Gtk.Align.CENTER}
        orientation={Gtk.Orientation.VERTICAL}
        spacing={8}
      >
        <label class="launcher-title" xalign={0} label="Clipboard" />
        <box orientation={Gtk.Orientation.VERTICAL} spacing={4}>
          <For each={items}>
            {(line) => (
              <button class="app" onClicked={() => paste(line)}>
                <label
                  hexpand
                  xalign={0}
                  ellipsize={Pango.EllipsizeMode.END}
                  label={line.replace(/^\S+\s+/, "") || line}
                />
              </button>
            )}
          </For>
        </box>
      </box>
    </window>
  )
}
