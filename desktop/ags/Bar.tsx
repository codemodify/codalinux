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
type TaskGroup = {
  key: string
  clients: HyprClient[]
  icon: string
  title: string
  focused: boolean
  minimized: boolean
}

function launch(tool: string) {
  execAsync(["coda-settings", tool]).catch(console.error)
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

function groupTasks(
  clients: HyprClient[],
  focused: HyprClient | null,
  apps: AstalApps.Apps,
): TaskGroup[] {
  const groups = new Map<string, HyprClient[]>()
  for (const client of clients) {
    if (!isTaskClient(client)) continue
    const key = (clientClass(client) || client.title || "app").toLowerCase()
    const list = groups.get(key) ?? []
    list.push(client)
    groups.set(key, list)
  }

  const focusedAddr = focused ? clientAddr(focused) : ""
  const tasks: TaskGroup[] = []
  for (const [key, members] of groups) {
    members.sort(
      (a, b) => (b.focusHistoryId ?? 0) - (a.focusHistoryId ?? 0),
    )
    const cls = clientClass(members[0])
    const titles = members.map((c) => c.title).filter(Boolean)
    const title =
      members.length === 1
        ? titles[0] || prettyClass(cls)
        : `${prettyClass(cls)} (${members.length})`
    tasks.push({
      key,
      clients: members,
      icon: iconFor(apps, cls, titles[0] || ""),
      title,
      focused: members.some((c) => clientAddr(c) === focusedAddr),
      minimized: members.every((c) => isMinimized(c)),
    })
  }
  tasks.sort((a, b) => a.key.localeCompare(b.key))
  return tasks
}

function activateTask(group: TaskGroup, hypr: AstalHyprland.Hyprland) {
  const focused = hypr.focusedClient
  const focusedAddr = focused ? clientAddr(focused) : ""
  const inGroup = group.clients.some((c) => clientAddr(c) === focusedAddr)

  if (inGroup && focused) {
    if (group.clients.length === 1) {
      hyprWs("toggle", [focusedAddr])
      return
    }
    const index = group.clients.findIndex((c) => clientAddr(c) === focusedAddr)
    const next = group.clients[(index + 1) % group.clients.length]
    hyprWs("focus", [clientAddr(next)])
    return
  }

  hyprWs("focus", [clientAddr(group.clients[0])])
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
  const tasks = createComputed(() =>
    groupTasks(clients() ?? [], focused() ?? null, apps),
  )

  return (
    <box class="taskbar" spacing={4} hexpand>
      <For each={tasks}>
        {(group) => (
          <button
            class={
              group.focused
                ? "task focused"
                : group.minimized
                  ? "task minimized"
                  : "task"
            }
            tooltipText={group.clients.map((c) => c.title || clientClass(c)).join("\n")}
            onClicked={() => activateTask(group, hypr)}
          >
            <box spacing={6}>
              <image iconName={group.icon} pixelSize={16} />
              <label
                class="task-title"
                label={group.title}
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
      <button class="audio" onClicked={() => launch("audio")}>
        <image iconName="audio-volume-high-symbolic" pixelSize={16} />
      </button>
    )
  }

  const volume = createBinding(speaker, "volume")((v) =>
    `${Math.round(v * 100)}%`,
  )
  const muted = createBinding(speaker, "mute")

  return (
    <button class="audio" onClicked={() => launch("audio")}>
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
      tooltipText="Wi-Fi (iwd / impala)"
      onClicked={() => launch("wifi")}
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
      onClicked={() => launch("bluetooth")}
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
            tooltipText="Control center"
            onClicked={() => toggle("control-center")}
          >
            <image iconName="preferences-system-symbolic" pixelSize={16} />
          </button>
        </box>
      </centerbox>
    </window>
  )
}
