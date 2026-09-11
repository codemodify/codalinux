import { createBinding, createComputed, createState } from "ags"
import app from "ags/gtk4/app"
import { exec, execAsync } from "ags/process"
import { createPoll } from "ags/time"
import { Astal, Gtk, Gdk } from "ags/gtk4"
import Graphene from "gi://Graphene"
import AstalBluetooth from "gi://AstalBluetooth"
import AstalWp from "gi://AstalWp"

const { TOP, BOTTOM, LEFT, RIGHT } = Astal.WindowAnchor

type Page =
  | "overview"
  | "display"
  | "sound"
  | "network"
  | "bluetooth"
  | "appearance"
  | "input"
  | "about"

function launch(tool: string) {
  execAsync(["coda-settings", tool]).catch(console.error)
}

function applyDisplay(mode: string) {
  execAsync(["coda-settings", "display", "apply", mode]).catch(console.error)
}

function Volume() {
  const wp = AstalWp.get_default()
  const speaker = wp?.defaultSpeaker
  if (!speaker) {
    return (
      <box class="status-row" spacing={10} hexpand>
        <image iconName="audio-volume-muted-symbolic" pixelSize={18} />
        <label
          hexpand
          xalign={0}
          label="Audio — open Mixer to configure PipeWire"
        />
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

function DisplayPanel() {
  const info = createPoll("Reading displays…", 2500, () => {
    try {
      const raw = exec(["hyprctl", "-j", "monitors"])
      const mons = JSON.parse(String(raw)) as Array<{
        name?: string
        width?: number
        height?: number
        refreshRate?: number
        scale?: number
        focused?: boolean
        availableModes?: string[]
      }>
      if (!Array.isArray(mons) || mons.length === 0) {
        return "No monitors reported by Hyprland."
      }
      return mons
        .map((m) => {
          const mark = m.focused ? " (focused)" : ""
          const hz = Math.round(m.refreshRate || 0)
          const modes = (m.availableModes || []).slice(0, 8).join(", ")
          return `${m.name || "?"}${mark}\n  ${m.width}×${m.height} @ ${hz} Hz  scale ${m.scale}\n  modes: ${modes || "preferred / hyprctl"}`
        })
        .join("\n\n")
    } catch {
      return "hyprctl monitors is unavailable."
    }
  })

  return (
    <box
      class="page-body"
      orientation={Gtk.Orientation.VERTICAL}
      spacing={10}
      hexpand
    >
      <label class="page-title" xalign={0} label="Display" />
      <label
        class="control-sub"
        wrap
        xalign={0}
        label="Resolution is applied with Hyprland (hyprctl). wlr-randr is used when present. QEMU guests must start virtio-vga at 1920×1080 or the device stays 640×480."
      />
      <label class="display-info" wrap xalign={0} label={info} />
      <label class="section" xalign={0} label="Apply to focused output" />
      <box spacing={8}>
        <button hexpand onClicked={() => applyDisplay("1920x1080")}>
          <label label="1920×1080" />
        </button>
        <button hexpand onClicked={() => applyDisplay("1280x720")}>
          <label label="1280×720" />
        </button>
        <button hexpand onClicked={() => applyDisplay("preferred")}>
          <label label="Preferred" />
        </button>
      </box>
      <button hexpand onClicked={() => launch("display")}>
        <label label="Monitor details…" />
      </button>
    </box>
  )
}

function AboutPanel() {
  const info = createPoll("Reading system…", 12000, () => {
    try {
      return exec(["coda-settings", "about", "print"]).trim()
    } catch {
      return "System info unavailable."
    }
  })

  return (
    <box
      class="page-body"
      orientation={Gtk.Orientation.VERTICAL}
      spacing={10}
      hexpand
    >
      <label class="page-title" xalign={0} label="About" />
      <label class="display-info" wrap xalign={0} label={info} />
    </box>
  )
}

function PageHeader({ title, body }: { title: string; body: string }) {
  return (
    <box orientation={Gtk.Orientation.VERTICAL} spacing={6}>
      <label class="page-title" xalign={0} label={title} />
      <label class="control-sub" wrap xalign={0} label={body} />
    </box>
  )
}

function LaunchRow({
  label,
  hint,
  tool,
  hide,
}: {
  label: string
  hint: string
  tool: string
  hide: () => void
}) {
  return (
    <button
      hexpand
      class="launch-row"
      onClicked={() => {
        hide()
        launch(tool)
      }}
    >
      <box spacing={10} hexpand>
        <label hexpand xalign={0} label={label} />
        <label class="hint" label={hint} />
      </box>
    </button>
  )
}

export default function ControlCenter() {
  let contentbox: Gtk.Box
  let win: Astal.Window
  const [page, setPage] = createState<Page>("overview")

  const hide = () => {
    if (win) win.visible = false
  }

  function onKey(_e: Gtk.EventControllerKey, keyval: number) {
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

  function NavItem({
    id,
    label,
    icon,
  }: {
    id: Page
    label: string
    icon: string
  }) {
    return (
      <button
        class={page((p) => (p === id ? "nav-item active" : "nav-item"))}
        onClicked={() => setPage(id)}
      >
        <box spacing={8}>
          <image iconName={icon} pixelSize={16} />
          <label hexpand xalign={0} label={label} />
        </box>
      </button>
    )
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
      onNotifyVisible={({ visible }) => {
        if (visible) setPage("overview")
      }}
    >
      <Gtk.EventControllerKey onKeyPressed={onKey} />
      <Gtk.GestureClick onPressed={onClick} />
      <box
        $={(self) => (contentbox = self)}
        class="control-content settings-hub"
        valign={Gtk.Align.CENTER}
        halign={Gtk.Align.CENTER}
        orientation={Gtk.Orientation.HORIZONTAL}
        spacing={0}
      >
        <box class="settings-nav" orientation={Gtk.Orientation.VERTICAL} spacing={4}>
          <label class="control-title" xalign={0} label="Settings" />
          <label
            class="control-sub"
            xalign={0}
            wrap
            label="One place for device settings"
          />
          <NavItem
            id="overview"
            label="Overview"
            icon="preferences-system-symbolic"
          />
          <NavItem
            id="display"
            label="Display"
            icon="video-display-symbolic"
          />
          <NavItem
            id="sound"
            label="Sound"
            icon="audio-headphones-symbolic"
          />
          <NavItem
            id="network"
            label="Network"
            icon="network-wireless-symbolic"
          />
          <NavItem
            id="bluetooth"
            label="Bluetooth"
            icon="bluetooth-symbolic"
          />
          <NavItem
            id="appearance"
            label="Appearance"
            icon="preferences-desktop-theme-symbolic"
          />
          <NavItem
            id="input"
            label="Input"
            icon="input-keyboard-symbolic"
          />
          <NavItem
            id="about"
            label="About"
            icon="dialog-information-symbolic"
          />
        </box>
        <Gtk.Separator orientation={Gtk.Orientation.VERTICAL} />
        <box
          class="settings-page"
          orientation={Gtk.Orientation.VERTICAL}
          spacing={12}
          hexpand
          vexpand
        >
          <box
            visible={page((p) => p === "overview")}
            orientation={Gtk.Orientation.VERTICAL}
            spacing={10}
            hexpand
          >
            <PageHeader
              title="Overview"
              body="Wi-Fi uses iwd (impala). Audio, Bluetooth, webcam, and appearance open official Arch apps."
            />
            <Volume />
            <NetworkStatus />
            <BluetoothStatus />
            <box spacing={8}>
              <button
                hexpand
                onClicked={() => {
                  hide()
                  launch("webcam")
                }}
              >
                <label label="Webcam" />
              </button>
              <button
                hexpand
                onClicked={() => {
                  hide()
                  execAsync("coda-hyprlock").catch(console.error)
                }}
              >
                <label label="Lock screen" />
              </button>
            </box>
          </box>

          <box visible={page((p) => p === "display")} hexpand>
            <DisplayPanel />
          </box>

          <box
            visible={page((p) => p === "sound")}
            orientation={Gtk.Orientation.VERTICAL}
            spacing={10}
            hexpand
          >
            <PageHeader
              title="Sound"
              body="PipeWire volume here; pavucontrol for devices, ports, and applications."
            />
            <Volume />
            <LaunchRow
              label="Open volume control"
              hint="pavucontrol"
              tool="audio"
              hide={hide}
            />
          </box>

          <box
            visible={page((p) => p === "network")}
            orientation={Gtk.Orientation.VERTICAL}
            spacing={10}
            hexpand
          >
            <PageHeader
              title="Network"
              body="systemd-networkd + iwd. There is no NetworkManager. Join Wi-Fi with impala."
            />
            <NetworkStatus />
            <LaunchRow
              label="Wi-Fi"
              hint="impala"
              tool="wifi"
              hide={hide}
            />
            <LaunchRow
              label="Network details"
              hint="networkctl"
              tool="netstatus"
              hide={hide}
            />
          </box>

          <box
            visible={page((p) => p === "bluetooth")}
            orientation={Gtk.Orientation.VERTICAL}
            spacing={10}
            hexpand
          >
            <PageHeader
              title="Bluetooth"
              body="Pair and connect devices with blueman."
            />
            <BluetoothStatus />
            <LaunchRow
              label="Bluetooth devices"
              hint="blueman-manager"
              tool="bluetooth"
              hide={hide}
            />
          </box>

          <box
            visible={page((p) => p === "appearance")}
            orientation={Gtk.Orientation.VERTICAL}
            spacing={10}
            hexpand
          >
            <PageHeader
              title="Appearance"
              body="GTK theme, icons, fonts, and cursors. Wallpaper is the Horos still under /usr/share/backgrounds/codalinux."
            />
            <LaunchRow
              label="GTK theme and cursors"
              hint="nwg-look"
              tool="appearance"
              hide={hide}
            />
          </box>

          <box
            visible={page((p) => p === "input")}
            orientation={Gtk.Orientation.VERTICAL}
            spacing={10}
            hexpand
          >
            <PageHeader
              title="Input"
              body="Keyboard layout is US. Super is the modifier. Touchpad: natural scroll and tap-to-click."
            />
            <label
              class="display-info"
              wrap
              xalign={0}
              label={`Super+Return  Terminal\nSuper+Space   App launcher\nSuper+,       Settings\nSuper+V       Clipboard\nSuper+L       Lock\nSuper+T       Tile ↔ stack\nTitlebar      close / maximize / minimize; scroll to shade`}
            />
            <LaunchRow
              label="Full keyboard and touchpad help"
              hint="shortcuts"
              tool="input"
              hide={hide}
            />
          </box>

          <box visible={page((p) => p === "about")} hexpand>
            <AboutPanel />
          </box>
        </box>
      </box>
    </window>
  )
}
