import { createBinding, For } from "ags"
import app from "ags/gtk4/app"
import GLib from "gi://GLib"
import Bar from "./Bar"
import Clipboard from "./Clipboard"
import ControlCenter from "./ControlCenter"
import Launcher from "./Launcher"
import NotificationPopups from "./NotificationPopups"
import style from "./style.css"

function toggleNamed(name: string): string {
  const win = app.get_window(name)
  if (!win) return `unknown window ${name}`
  win.visible = !win.visible
  return "ok"
}

app.start({
  css: style,
  gtkTheme: "Adwaita",
  instanceName: "coda",
  requestHandler(request, res) {
    const [, argv] = GLib.shell_parse_argv(request)
    if (!argv) return res("argv parse error")

    switch (argv[0]) {
      case "toggle":
        return res(toggleNamed(argv[1] ?? ""))
      case "open": {
        const win = app.get_window(argv[1] ?? "")
        if (win) win.visible = true
        return res(win ? "ok" : `unknown window ${argv[1]}`)
      }
      case "close": {
        const win = app.get_window(argv[1] ?? "")
        if (win) win.visible = false
        return res(win ? "ok" : `unknown window ${argv[1]}`)
      }
      default:
        return res("unknown command")
    }
  },
  main() {
    Launcher()
    ControlCenter()
    Clipboard()
    NotificationPopups()

    const monitors = createBinding(app, "monitors")
    return (
      <For each={monitors}>
        {(monitor) => <Bar gdkmonitor={monitor} />}
      </For>
    )
  },
})
