import { createBinding, createComputed } from "ags"
import app from "ags/gtk4/app"
import { exec, execAsync } from "ags/process"
import { createPoll } from "ags/time"
import { Astal, Gtk, Gdk } from "ags/gtk4"
import Graphene from "gi://Graphene"
import AstalBluetooth from "gi://AstalBluetooth"
import AstalWp from "gi://AstalWp"

const { TOP, BOTTOM, LEFT, RIGHT } = Astal.WindowAnchor

function launch(tool: string) {
  execAsync(["coda-settings", tool]).catch(console.error)
}

function Tile({
  label,
  hint,
  icon,
  tool,
  hide,
}: {
  label: string
  hint: string
  icon: string
  tool: string
  hide: () => void
}) {
  return (
    <button
      class="tile"
      tooltipText={hint}
      onClicked={() => {
        hide()
        launch(tool)
      }}
    >
      <box
        orientation={Gtk.Orientation.VERTICAL}
        spacing={6}
        valign={Gtk.Align.CENTER}
      >
        <image iconName={icon} pixelSize={28} />
        <label label={label} />
        <label class="tile-hint" label={hint} wrap xalign={0.5} />
      </box>
    </button>
  )
}

function Volume() {
  const wp = AstalWp.get_default()
  const speaker = wp?.defaultSpeaker
  if (!speaker) {
    return (
      <box class="status-row" spacing={10} hexpand>
        <image iconName="audio-volume-muted-symbolic" pixelSize={18} />
        <label hexpand xalign={0} label="Audio — open Mixer to configure PipeWire" />
        <button onClicked={() => launch("audio")}>
          <label label="Mixer" />
        </button>
      </box>
    )
  }

  const volume = createBinding(speaker, "volume")
  const muted = createBinding(speaker, "mute")
  const label = createComputed(() => {
    if (muted()) return "Muted"
    return `${Math.round((volume() ?? 0) * 100)}%`
  })

  return (
    <box class="volume-row" spacing={10} hexpand>
      <image
        iconName={muted((m) =>
          m ? "audio-volume-muted-symbolic" : "audio-volume-high-symbolic",
        )}
        pixelSize={18}
      />
      <slider
        hexpand
        min={0}
        max={1}
        value={volume}
        onChangeValue={({ value }) => speaker.set_volume(value)}
      />
      <label class="hint" label={label} widthChars={5} />
      <button
        tooltipText="Mute"
        onClicked={() => speaker.set_mute(!speaker.mute)}
      >
        <label label="Mute" />
      </button>
      <button onClicked={() => launch("audio")}>
        <label label="Mixer" />
      </button>
    </box>
  )
}

function NetworkStatus() {
  const status = createPoll("Checking network…", 4000, () => {
    try {
      const out = exec(["networkctl", "is-online"]).trim().toLowerCase()
      if (out.includes("online")) return "Online — systemd-networkd + iwd"
      return "Offline — open Wi-Fi (impala) to join a network"
    } catch {
      return "Offline — open Wi-Fi (impala) to join a network"
    }
  })

  return (
    <box class="status-row" spacing={10} hexpand>
      <image iconName="network-wireless-symbolic" pixelSize={18} />
      <label hexpand xalign={0} wrap label={status} />
      <button onClicked={() => launch("wifi")}>
        <label label="Wi-Fi" />
      </button>
    </box>
  )
}

function BluetoothStatus() {
  const bt = AstalBluetooth.get_default()
  const enabled = createBinding(bt, "isPowered")
  const connected = createBinding(bt, "isConnected")
  const text = createComputed(() => {
    if (!enabled()) return "Bluetooth off"
    return connected() ? "Bluetooth connected" : "Bluetooth on — no device"
  })
  const icon = createComputed(() => {
    if (!enabled()) return "bluetooth-disabled-symbolic"
    return connected() ? "bluetooth-active-symbolic" : "bluetooth-symbolic"
  })

  return (
    <box class="status-row" spacing={10} hexpand>
      <image iconName={icon} pixelSize={18} />
      <label hexpand xalign={0} label={text} />
      <button onClicked={() => launch("bluetooth")}>
        <label label="Devices" />
      </button>
    </box>
  )
}

export default function ControlCenter() {
  let contentbox: Gtk.Box
  let win: Astal.Window
  const hide = () => {
    if (win) win.visible = false
  }

  function onKey(
    _e: Gtk.EventControllerKey,
    keyval: number,
  ) {
    if (keyval === Gdk.KEY_Escape) hide()
  }

  function onClick(_e: Gtk.GestureClick, _: number, x: number, y: number) {
    const [, rect] = contentbox.compute_bounds(win)
    const position = new Graphene.Point({ x, y })
    if (!rect.contains_point(position)) {
      hide()
      return true
    }
  }

  return (
    <window
      $={(self) => (win = self)}
      name="control-center"
      class="ControlCenter"
      visible={false}
      anchor={TOP | BOTTOM | LEFT | RIGHT}
      exclusivity={Astal.Exclusivity.IGNORE}
      keymode={Astal.Keymode.EXCLUSIVE}
      application={app}
    >
      <Gtk.EventControllerKey onKeyPressed={onKey} />
      <Gtk.GestureClick onPressed={onClick} />
      <box
        $={(self) => (contentbox = self)}
        class="control-content"
        valign={Gtk.Align.START}
        halign={Gtk.Align.END}
        orientation={Gtk.Orientation.VERTICAL}
        spacing={12}
      >
        <label class="control-title" xalign={0} label="Settings" />
        <label
          class="control-sub"
          xalign={0}
          label="Wi-Fi uses iwd (impala). Audio, Bluetooth, webcam, and appearance open official Arch apps."
          wrap
        />
        <Volume />
        <NetworkStatus />
        <BluetoothStatus />
        <label class="section" xalign={0} label="Open" />
        <box class="tiles" spacing={8}>
          <Tile
            label="Wi-Fi"
            hint="impala"
            icon="network-wireless-symbolic"
            tool="wifi"
            hide={hide}
          />
          <Tile
            label="Bluetooth"
            hint="blueman"
            icon="bluetooth-symbolic"
            tool="bluetooth"
            hide={hide}
          />
          <Tile
            label="Audio"
            hint="pavucontrol"
            icon="audio-headphones-symbolic"
            tool="audio"
            hide={hide}
          />
        </box>
        <box class="tiles" spacing={8}>
          <Tile
            label="Webcam"
            hint="snapshot"
            icon="camera-web-symbolic"
            tool="webcam"
            hide={hide}
          />
          <Tile
            label="Appearance"
            hint="nwg-look"
            icon="preferences-desktop-theme-symbolic"
            tool="appearance"
            hide={hide}
          />
          <Tile
            label="Input"
            hint="keys & touchpad"
            icon="input-keyboard-symbolic"
            tool="input"
            hide={hide}
          />
        </box>
        <box spacing={8}>
          <button
            hexpand
            onClicked={() => {
              hide()
              launch("netstatus")
            }}
          >
            <label label="Network details" />
          </button>
          <button
            hexpand
            onClicked={() => {
              hide()
              execAsync("hyprlock").catch(console.error)
            }}
          >
            <label label="Lock screen" />
          </button>
        </box>
      </box>
    </window>
  )
}
