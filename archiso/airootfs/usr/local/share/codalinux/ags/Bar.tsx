import { createBinding, createComputed, For, onCleanup } from "ags"
import app from "ags/gtk4/app"
import { exec, execAsync } from "ags/process"
import { createPoll } from "ags/time"
import { Astal, Gtk, Gdk } from "ags/gtk4"
import GLib from "gi://GLib"
import Pango from "gi://Pango"
import AstalApps from "gi://AstalApps"
import AstalBattery from "gi://AstalBattery"
import AstalBluetooth from "gi://AstalBluetooth"
import AstalHyprland from "gi://AstalHyprland"
import AstalWp from "gi://AstalWp"

type HyprClient = AstalHyprland.Client
type TaskItem = {
  key: string
  client: HyprClient
  icon: string
  title: string
  focused: boolean
  minimized: boolean
}

/** Last regular workspace id per client, so minimized windows stay on that workspace's bar. */
const lastRegularWs = new Map<string, number>()

function launchSystemConfig() {
  execAsync(["system-config-gui"]).catch(console.error)
}

function toggle(name: string) {
  const win = app.get_window(name)
  if (win) win.visible = !win.visible
}

function hyprWs(cmd: string, extra: string[] = []) {
  execAsync(["coda-hypr-ws", cmd, ...extra]).catch(console.error)
}

function clientClass(client: HyprClient) {
  return (client.initialClass || client.class || "").trim()
}

function clientAddr(client: HyprClient) {
  return String(client.address || "")
}

function isSpecialWorkspace(ws?: AstalHyprland.Workspace | null) {
  if (!ws) return true
  const name = ws.name || ""
  return ws.id < 0 || name.startsWith("special")
}

function isMinimized(client: HyprClient) {
  const name = client.workspace?.name || ""
  return name === "special:minimized" || name === "special:min"
}

function isTaskClient(client: HyprClient) {
  if (client.mapped === false) return false
  if (client.hidden && !isMinimized(client)) return false
  const ws = client.workspace
  if (!ws) return false
  if (isSpecialWorkspace(ws) && !isMinimized(client)) return false
  return Boolean(clientClass(client) || client.title)
}

function iconFor(apps: AstalApps.Apps, cls: string, title: string) {
  const queries = [cls, cls.split(".").pop() || "", title.split(/\s+/)[0] || ""]
  for (const query of queries) {
    if (!query) continue
    const hit = apps.fuzzy_query(query)[0]
    if (hit?.iconName) return hit.iconName
  }

  const display = Gdk.Display.get_default()
  if (display) {
    const theme = Gtk.IconTheme.get_for_display(display)
    const names = [
      cls.toLowerCase(),
      cls.toLowerCase().replace(/\./g, "-"),
      `${cls.toLowerCase()}-symbolic`,
    ]
    for (const name of names) {
      if (name && theme.has_icon(name)) return name
    }
  }
  return "application-x-executable-symbolic"
}

function prettyClass(cls: string) {
  const leaf = cls.split(".").pop() || cls
  if (!leaf) return "App"
  return leaf.charAt(0).toUpperCase() + leaf.slice(1)
}

function rememberRegularWorkspace(client: HyprClient) {
  const ws = client.workspace
  const addr = clientAddr(client)
  if (!addr || !ws) return
  if (ws.id > 0 && !isSpecialWorkspace(ws)) {
    lastRegularWs.set(addr, ws.id)
  }
}

function pruneWorkspaceMemory(clients: HyprClient[]) {
  const live = new Set(clients.map(clientAddr).filter(Boolean))
  for (const addr of lastRegularWs.keys()) {
    if (!live.has(addr)) lastRegularWs.delete(addr)
  }
}

function belongsToFocusedWorkspace(
  client: HyprClient,
  focusedWs: AstalHyprland.Workspace | null,
) {
  if (!focusedWs || focusedWs.id <= 0) {
    return !isSpecialWorkspace(client.workspace) || isMinimized(client)
  }
  if (client.workspace?.id === focusedWs.id) return true
  if (isMinimized(client) && lastRegularWs.get(clientAddr(client)) === focusedWs.id) {
    return true
  }
  return false
}

function listTasks(
  clients: HyprClient[],
  focused: HyprClient | null,
  focusedWs: AstalHyprland.Workspace | null,
  apps: AstalApps.Apps,
): TaskItem[] {
  pruneWorkspaceMemory(clients)
  const focusedAddr = focused ? clientAddr(focused) : ""
  const tasks: TaskItem[] = []
  for (const client of clients) {
    rememberRegularWorkspace(client)
    if (!isTaskClient(client)) continue
    if (!belongsToFocusedWorkspace(client, focusedWs)) continue
    const cls = clientClass(client)
    const title = client.title || prettyClass(cls)
    tasks.push({
      key: clientAddr(client) || `${cls}-${title}`,
      client,
      icon: iconFor(apps, cls, title),
      title,
      focused: clientAddr(client) === focusedAddr,
      minimized: isMinimized(client),
    })
  }
  tasks.sort(
    (a, b) => (a.client.focusHistoryId ?? 0) - (b.client.focusHistoryId ?? 0),
  )
  return tasks
}

function activateTask(item: TaskItem, hypr: AstalHyprland.Hyprland) {
  const addr = clientAddr(item.client)
  if (!addr) return
  const focused = hypr.focusedClient
  if (focused && clientAddr(focused) === addr) {
    hyprWs("toggle", [addr])
    return
  }
  hyprWs("focus", [addr])
}

function closeTask(client: HyprClient) {
  const addr = clientAddr(client)
  if (!addr) return
  const kill = (client as HyprClient & { kill?: () => void }).kill
  if (typeof kill === "function") {
    try {
      kill.call(client)
      return
    } catch {
      /* fall through to hyprctl */
    }
  }
  execAsync(["hyprctl", "dispatch", "closewindow", `address:${addr}`]).catch(
    console.error,
  )
}

function desktopAppFor(apps: AstalApps.Apps, client: HyprClient) {
  const cls = clientClass(client)
  const queries = [
    cls,
    cls.split(".").pop() || "",
    (client.initialClass || "").trim(),
    (client.title || "").split(/\s+/)[0] || "",
  ]
  for (const query of queries) {
    if (!query) continue
    const hit = apps.fuzzy_query(query)[0]
    if (hit) return hit
  }
  return null
}

function launchNewInstance(client: HyprClient, apps: AstalApps.Apps) {
  const found = desktopAppFor(apps, client)
  if (found) {
    try {
      found.launch()
      return
    } catch {
      const exe =
        (found as AstalApps.Application & { executable?: string }).executable ||
        ""
      if (exe) {
        execAsync(["hyprctl", "dispatch", "exec", exe]).catch(console.error)
        return
      }
    }
  }
  const cls = clientClass(client)
  const fallback = (cls.split(".").pop() || cls).toLowerCase()
  if (fallback) {
    execAsync(["hyprctl", "dispatch", "exec", fallback]).catch(console.error)
  }
}

type MenuHost = Gtk.Button & {
  _codaClient?: HyprClient
  _codaApps?: AstalApps.Apps
  _codaMenu?: boolean
}

function attachTaskMenu(
  button: Gtk.Button,
  client: HyprClient,
  apps: AstalApps.Apps,
) {
  const host = button as MenuHost
  host._codaClient = client
  host._codaApps = apps
  if (host._codaMenu) return
  host._codaMenu = true

  const pop = new Gtk.Popover()
  pop.set_parent(button)
  pop.set_autohide(true)
  pop.set_has_arrow(false)
  pop.add_css_class("task-menu")

  const box = new Gtk.Box({
    orientation: Gtk.Orientation.VERTICAL,
    spacing: 2,
  })
  const closeBtn = new Gtk.Button({ label: "Close" })
  closeBtn.add_css_class("task-menu-item")
  closeBtn.connect("clicked", () => {
    if (host._codaClient) closeTask(host._codaClient)
    pop.popdown()
  })
  const newBtn = new Gtk.Button({ label: "New instance" })
  newBtn.add_css_class("task-menu-item")
  newBtn.connect("clicked", () => {
    if (host._codaClient && host._codaApps) {
      launchNewInstance(host._codaClient, host._codaApps)
    }
    pop.popdown()
  })
  box.append(closeBtn)
  box.append(newBtn)
  pop.set_child(box)

  const right = new Gtk.GestureClick({ button: Gdk.BUTTON_SECONDARY })
  right.connect("pressed", () => {
    pop.popup()
  })
  button.add_controller(right)
}

function Workspaces() {
  const hypr = AstalHyprland.get_default()
  if (!hypr) return <box />

  const focused = createBinding(hypr, "focusedWorkspace")
  const workspaces = createBinding(hypr, "workspaces")((wss) =>
    [...wss].filter((ws) => ws.id > 0).sort((a, b) => a.id - b.id),
  )

  return (
    <box class="workspaces" spacing={4}>
      <For each={workspaces}>
        {(ws) => (
          <button
            class={focused((f) => (f?.id === ws.id ? "ws focused" : "ws"))}
            onClicked={() => hyprWs("switch", [String(ws.id)])}
          >
            <label label={String(ws.id)} />
          </button>
        )}
      </For>
      <button
        class="ws-action"
        tooltipText="New workspace"
        onClicked={() => hyprWs("add")}
      >
        <label label="+" />
      </button>
      <button
        class="ws-action"
        tooltipText="Remove empty workspace"
        onClicked={() => hyprWs("remove")}
      >
        <label label="−" />
      </button>
    </box>
  )
}

function StackToggle() {
  const mode = createPoll("tile", 750, () => {
    try {
      const out = exec(["coda-hypr-ws", "mode"]).trim()
      return out === "stack" ? "stack" : "tile"
    } catch {
      return "tile"
    }
  })

  return (
    <button
      class={mode((m) => (m === "stack" ? "stack-toggle stack" : "stack-toggle"))}
      tooltipText={mode((m) =>
        m === "stack"
          ? "Overlapping float — click for tiling"
          : "Tiling — click for overlapping float",
      )}
      onClicked={() => hyprWs("toggle-stack")}
    >
      <label label={mode((m) => (m === "stack" ? "Stack" : "Tile"))} />
    </button>
  )
}

function Taskbar() {
  const hypr = AstalHyprland.get_default()
  if (!hypr) return <box />

  const apps = new AstalApps.Apps()
  const clients = createBinding(hypr, "clients")
  const focused = createBinding(hypr, "focusedClient")
  const focusedWs = createBinding(hypr, "focusedWorkspace")
  const tasks = createComputed(() =>
    listTasks(
      clients() ?? [],
      focused() ?? null,
      focusedWs() ?? null,
      apps,
    ),
  )

  return (
    <box class="taskbar" spacing={4} hexpand>
      <For each={tasks}>
        {(item) => (
          <button
            class={
              item.focused
                ? "task focused"
                : item.minimized
                  ? "task minimized"
                  : "task"
            }
            tooltipText={item.title}
            onClicked={() => activateTask(item, hypr)}
            $={(self) => attachTaskMenu(self, item.client, apps)}
          >
            <box spacing={6}>
              <image iconName={item.icon} pixelSize={16} />
              <label
                class="task-title"
                label={item.title}
                maxWidthChars={18}
                ellipsize={Pango.EllipsizeMode.END}
              />
            </box>
          </button>
        )}
      </For>
    </box>
  )
}

function Clock() {
  const time = createPoll("", 1000, () => {
    return GLib.DateTime.new_now_local().format("%a %b %-d  %H:%M")!
  })

  return (
    <button class="clock" onClicked={() => toggle("control-center")}>
      <label label={time} />
    </button>
  )
}

function Audio() {
  const wp = AstalWp.get_default()
  const speaker = wp?.defaultSpeaker
  if (!speaker) {
    return (
      <button class="audio" onClicked={() => launchSystemConfig()}>
        <image iconName="audio-volume-high-symbolic" pixelSize={16} />
      </button>
    )
  }

  const volume = createBinding(speaker, "volume")((v) =>
    `${Math.round(v * 100)}%`,
  )
  const muted = createBinding(speaker, "mute")

  return (
    <button class="audio" onClicked={() => launchSystemConfig()}>
      <box spacing={6}>
        <image
          iconName={muted((m) =>
            m ? "audio-volume-muted-symbolic" : "audio-volume-high-symbolic",
          )}
          pixelSize={16}
        />
        <label label={volume} />
      </box>
    </button>
  )
}

function Wifi() {
  return (
    <button
      class="wifi"
      tooltipText="Wi-Fi (system-config)"
      onClicked={() => launchSystemConfig()}
    >
      <image iconName="network-wireless-symbolic" pixelSize={16} />
    </button>
  )
}

function Bluetooth() {
  const bt = AstalBluetooth.get_default()
  const enabled = createBinding(bt, "isPowered")
  const connected = createBinding(bt, "isConnected")
  const icon = createComputed(() => {
    if (!enabled()) return "bluetooth-disabled-symbolic"
    return connected() ? "bluetooth-active-symbolic" : "bluetooth-symbolic"
  })

  return (
    <button
      class="bluetooth"
      tooltipText="Bluetooth"
      onClicked={() => launchSystemConfig()}
    >
      <image iconName={icon} pixelSize={16} />
    </button>
  )
}

function Battery() {
  const battery = AstalBattery.get_default()
  if (!battery) return <box />

  const present = createBinding(battery, "isPresent")
  const percent = createBinding(battery, "percentage")((p) =>
    `${Math.floor(p * 100)}%`,
  )

  return (
    <box visible={present} class="battery" spacing={6}>
      <image iconName="battery-symbolic" pixelSize={16} />
      <label label={percent} />
    </box>
  )
}

export default function Bar({ gdkmonitor }: { gdkmonitor: Gdk.Monitor }) {
  let win: Astal.Window
  const { TOP, LEFT, RIGHT } = Astal.WindowAnchor

  onCleanup(() => {
    win.destroy()
  })

  return (
    <window
      $={(self) => (win = self)}
      visible
      class="Bar"
      namespace="coda-bar"
      name={`bar-${gdkmonitor.connector}`}
      gdkmonitor={gdkmonitor}
      exclusivity={Astal.Exclusivity.EXCLUSIVE}
      anchor={TOP | LEFT | RIGHT}
      application={app}
    >
      <centerbox class="bar-inner">
        <box $type="start" spacing={8} hexpand>
          <button class="brand" onClicked={() => toggle("launcher")}>
            <label label="Coda" />
          </button>
          <Workspaces />
          <StackToggle />
          <Taskbar />
        </box>
        <box $type="center">
          <Clock />
        </box>
        <box $type="end" spacing={4} hexpand halign={Gtk.Align.END}>
          <Audio />
          <Wifi />
          <Bluetooth />
          <Battery />
          <button
            class="settings"
            tooltipText="Settings"
            onClicked={() => toggle("control-center")}
          >
            <image iconName="preferences-system-symbolic" pixelSize={16} />
          </button>
        </box>
      </centerbox>
    </window>
  )
}
