import { createBinding, createComputed, For, onCleanup } from "ags"
import app from "ags/gtk4/app"
import { execAsync } from "ags/process"
import { createPoll } from "ags/time"
import { Astal, Gtk, Gdk } from "ags/gtk4"
import GLib from "gi://GLib"
import AstalBattery from "gi://AstalBattery"
import AstalBluetooth from "gi://AstalBluetooth"
import AstalHyprland from "gi://AstalHyprland"
import AstalWp from "gi://AstalWp"

function launch(tool: string) {
  execAsync(["coda-settings", tool]).catch(console.error)
}

function toggle(name: string) {
  const win = app.get_window(name)
  if (win) win.visible = !win.visible
}

function Workspaces() {
  const hypr = AstalHyprland.get_default()
  if (!hypr) return <box />

  const focused = createBinding(hypr, "focusedWorkspace")
  const workspaces = createBinding(hypr, "workspaces")((wss) =>
    [...wss].sort((a, b) => a.id - b.id),
  )

  return (
    <box class="workspaces" spacing={4}>
      <For each={workspaces}>
        {(ws) => (
          <button
            class={focused((f) => (f?.id === ws.id ? "ws focused" : "ws"))}
            onClicked={() => hypr.dispatch("workspace", String(ws.id))}
          >
            <label label={String(ws.id)} />
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
