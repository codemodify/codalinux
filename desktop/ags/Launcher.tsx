import { For, createState } from "ags"
import app from "ags/gtk4/app"
import { Astal, Gtk, Gdk } from "ags/gtk4"
import AstalApps from "gi://AstalApps"
import Graphene from "gi://Graphene"

const { TOP, BOTTOM, LEFT, RIGHT } = Astal.WindowAnchor

export default function Launcher() {
  let contentbox: Gtk.Box
  let searchentry: Gtk.Entry
  let win: Astal.Window

  const apps = new AstalApps.Apps()
  const [list, setList] = createState(new Array<AstalApps.Application>())

  function search(text: string) {
    if (text === "") setList(apps.fuzzy_query("").slice(0, 10))
    else setList(apps.fuzzy_query(text).slice(0, 10))
  }

  function launch(appEntry?: AstalApps.Application) {
    if (!appEntry) return
    win.visible = false
    appEntry.launch()
  }

  function onKey(
    _e: Gtk.EventControllerKey,
    keyval: number,
    _: number,
    mod: number,
  ) {
    if (keyval === Gdk.KEY_Escape) {
      win.visible = false
      return
    }
    if (keyval === Gdk.KEY_Return) {
      launch(list.get()[0])
      return
    }
    if (mod === Gdk.ModifierType.ALT_MASK) {
      for (const i of [1, 2, 3, 4, 5, 6, 7, 8, 9] as const) {
        if (keyval === Gdk[`KEY_${i}`]) {
          launch(list.get()[i - 1])
          return
        }
      }
    }
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
      name="launcher"
      class="Launcher"
      visible={false}
      anchor={TOP | BOTTOM | LEFT | RIGHT}
      exclusivity={Astal.Exclusivity.IGNORE}
      keymode={Astal.Keymode.EXCLUSIVE}
      application={app}
      onNotifyVisible={({ visible }) => {
        if (visible) {
          search(searchentry.get_text())
          searchentry.grab_focus()
        } else {
          searchentry.set_text("")
        }
      }}
    >
      <Gtk.EventControllerKey onKeyPressed={onKey} />
      <Gtk.GestureClick onPressed={onClick} />
      <box
        $={(self) => (contentbox = self)}
        class="launcher-content"
        valign={Gtk.Align.CENTER}
        halign={Gtk.Align.CENTER}
        orientation={Gtk.Orientation.VERTICAL}
        spacing={8}
      >
        <label class="launcher-title" label="Launch" xalign={0} />
        <entry
          $={(self) => (searchentry = self)}
          placeholderText="Search applications"
          onNotifyText={({ text }) => search(text)}
        />
        <Gtk.Separator visible={list((l) => l.length > 0)} />
        <box orientation={Gtk.Orientation.VERTICAL} spacing={4}>
          <For each={list}>
            {(entry, index) => (
              <button class="app" onClicked={() => launch(entry)}>
                <box spacing={10}>
                  <image iconName={entry.iconName} pixelSize={28} />
                  <label hexpand xalign={0} label={entry.name} />
                  <label
                    class="hint"
                    label={index((i) => `Alt+${i + 1}`)}
                  />
                </box>
              </button>
            )}
          </For>
        </box>
      </box>
    </window>
  )
}
