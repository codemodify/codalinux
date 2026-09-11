import { createBinding } from "ags"
import app from "ags/gtk4/app"
import { execAsync } from "ags/process"
import { Astal, Gtk, Gdk } from "ags/gtk4"
import Graphene from "gi://Graphene"
import AstalWp from "gi://AstalWp"

const { TOP, BOTTOM, LEFT, RIGHT } = Astal.WindowAnchor

function launch(tool: string) {
  execAsync(["coda-settings", tool]).catch(console.error)
}

function Tile({
  label,
  icon,
  tool,
  hide,
}: {
  label: string
  icon: string
  tool: string
  hide: () => void
}) {
  return (
    <button
      class="tile"
      onClicked={() => {
        hide()
        launch(tool)
      }}
    >
      <box
        orientation={Gtk.Orientation.VERTICAL}
        spacing={8}
        valign={Gtk.Align.CENTER}
      >
        <image iconName={icon} pixelSize={32} />
        <label label={label} />
      </box>
    </button>
  )
}

function Volume() {
  const wp = AstalWp.get_default()
  const speaker = wp?.defaultSpeaker
  if (!speaker) return <box />

  return (
    <box class="volume-row" spacing={10} hexpand>
      <image iconName="audio-volume-high-symbolic" pixelSize={18} />
      <slider
        hexpand
        min={0}
        max={1}
        value={createBinding(speaker, "volume")}
        onChangeValue={({ value }) => speaker.set_volume(value)}
      />
      <button onClicked={() => launch("audio")}>
        <label label="Mixer" />
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
        <label class="control-title" xalign={0} label="Control center" />
        <label
          class="control-sub"
          xalign={0}
          label="Official settings apps — Wi-Fi uses iwd (impala), not NetworkManager."
          wrap
        />
        <Volume />
        <box class="tiles" spacing={8}>
          <Tile
            label="Wi-Fi"
            icon="network-wireless-symbolic"
            tool="wifi"
            hide={hide}
          />
          <Tile
            label="Bluetooth"
            icon="bluetooth-symbolic"
            tool="bluetooth"
            hide={hide}
          />
          <Tile
            label="Audio"
            icon="audio-headphones-symbolic"
            tool="audio"
            hide={hide}
          />
        </box>
        <box class="tiles" spacing={8}>
          <Tile
            label="Webcam"
            icon="camera-web-symbolic"
            tool="webcam"
            hide={hide}
          />
          <Tile
            label="Appearance"
            icon="preferences-desktop-theme-symbolic"
            tool="appearance"
            hide={hide}
          />
          <Tile
            label="Input"
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
            <label label="Network status" />
          </button>
          <button
            hexpand
            onClicked={() => {
              hide()
              execAsync("hyprlock").catch(console.error)
            }}
          >
            <label label="Lock" />
          </button>
        </box>
      </box>
    </window>
  )
}
