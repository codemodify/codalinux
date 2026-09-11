import { createBinding, createComputed, createState } from "ags"
import app from "ags/gtk4/app"
import { exec, execAsync } from "ags/process"
import { createPoll } from "ags/time"
import { Astal, Gtk, Gdk } from "ags/gtk4"
import Graphene from "gi://Graphene"
import AstalBattery from "gi://AstalBattery"
import AstalBluetooth from "gi://AstalBluetooth"
import AstalNotifd from "gi://AstalNotifd"
import AstalWp from "gi://AstalWp"

const { TOP, BOTTOM, LEFT, RIGHT } = Astal.WindowAnchor

type Page =
  | "overview"
  | "appearance"
  | "wallpaper"
  | "desktops"
  | "windows"
  | "effects"
  | "lock"
  | "shortcuts"
  | "startup"
  | "search"
  | "notifications"
  | "regional"
  | "users"
  | "accessibility"
  | "applications"
  | "network"
  | "display"
  | "sound"
  | "bluetooth"
  | "input"
  | "power"
  | "storage"
  | "about"

function launch(tool: string, ...args: string[]) {
  execAsync(["coda-settings", tool, ...args]).catch(console.error)
}

function applyDisplay(mode: string) {
  launch("display", "apply", mode)
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
  args = [],
  hide,
}: {
  label: string
  hint: string
  tool: string
  args?: string[]
  hide: () => void
}) {
  return (
    <button
      hexpand
      class="launch-row"
      onClicked={() => {
        hide()
        launch(tool, ...args)
      }}
    >
      <box spacing={10} hexpand>
        <label hexpand xalign={0} label={label} />
        <label class="hint" label={hint} />
      </box>
    </button>
  )
}

function ActionRow({
  label,
  hint,
  onClicked,
}: {
  label: string
  hint?: string
  onClicked: () => void
}) {
  return (
    <button hexpand class="launch-row" onClicked={onClicked}>
      <box spacing={10} hexpand>
        <label hexpand xalign={0} label={label} />
        <label class="hint" label={hint || ""} />
      </box>
    </button>
  )
}

function SkipNote({ title, reason }: { title: string; reason: string }) {
  return (
    <box class="skip-row" spacing={8} hexpand>
      <label class="skip-title" xalign={0} label={title} />
      <label class="hint" hexpand wrap xalign={1} label={reason} />
    </box>
  )
}

function PollInfo({
  cmd,
  interval = 3000,
}: {
  cmd: string[]
  interval?: number
}) {
  const info = createPoll("…", interval, () => {
    try {
      return exec(cmd).trim() || "No data."
    } catch {
      return "Unavailable."
    }
  })
  return <label class="display-info" wrap xalign={0} label={info} />
}

function DisplayPanel({ hide }: { hide: () => void }) {
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
          return `${m.name || "?"}${mark}\n  ${m.width}×${m.height} @ ${hz} Hz  scale ${m.scale}\n  modes: ${modes || "1920x1080@60"}`
        })
        .join("\n\n")
    } catch {
      return "hyprctl monitors is unavailable."
    }
  })

  return (
    <box class="page-body" orientation={Gtk.Orientation.VERTICAL} spacing={10} hexpand>
      <PageHeader
        title="Display and Monitor"
        body="Hyprland preferred on QEMU virtio EDID picks 640×480@120 (listed first). Session default is 1920×1080@60. Night Color / KWin compositor are not on this stack."
      />
      <label class="display-info" wrap xalign={0} label={info} />
      <label class="section" xalign={0} label="Resolution (focused output)" />
      <box spacing={8}>
        <button hexpand onClicked={() => applyDisplay("1920x1080@60")}>
          <label label="1920×1080@60" />
        </button>
        <button hexpand onClicked={() => applyDisplay("1280x720@60")}>
          <label label="1280×720@60" />
        </button>
      </box>
      <button
        hexpand
        tooltipText="On QEMU virtio this often becomes 640×480@120"
        onClicked={() => applyDisplay("preferred")}
      >
        <label label="Preferred (avoid on QEMU)" />
      </button>
      <label class="section" xalign={0} label="Scale" />
      <box spacing={8}>
        <button hexpand onClicked={() => launch("display", "scale", "1")}>
          <label label="100%" />
        </button>
        <button hexpand onClicked={() => launch("display", "scale", "1.25")}>
          <label label="125%" />
        </button>
      </box>
      <SkipNote
        title="Night Color"
        reason="No hyprsunset / wlsunset / Plasma Night Color"
      />
      <LaunchRow
        label="Monitor details…"
        hint="hyprctl"
        tool="display"
        hide={hide}
      />
    </box>
  )
}

function AboutPanel() {
  return (
    <box class="page-body" orientation={Gtk.Orientation.VERTICAL} spacing={10} hexpand>
      <PageHeader
        title="System Information"
        body="About this CodaLinux session. Plasma System Information extras are not shipped."
      />
      <PollInfo cmd={["coda-settings", "about", "print"]} interval={12000} />
    </box>
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

  function Jump(id: Page, label: string) {
    return (
      <button hexpand onClicked={() => setPage(id)}>
        <label label={label} />
      </button>
    )
  }

  const notifd = AstalNotifd.get_default()
  const dnd = createBinding(notifd, "dontDisturb")

  const battery = AstalBattery.get_default()
  const battPresent = battery ? createBinding(battery, "isPresent") : null
  const battPct = battery
    ? createBinding(battery, "percentage")((p) => `${Math.floor((p ?? 0) * 100)}%`)
    : null

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
        class="control-content settings-hub"
        valign={Gtk.Align.CENTER}
        halign={Gtk.Align.CENTER}
        orientation={Gtk.Orientation.HORIZONTAL}
        spacing={0}
      >
        <Gtk.ScrolledWindow
          class="settings-nav-scroll"
          vexpand
          hexpand={false}
          minContentWidth={214}
        >
          <box class="settings-nav" orientation={Gtk.Orientation.VERTICAL} spacing={3}>
            <label class="control-title" xalign={0} label="Settings" />
            <label
              class="control-sub"
              xalign={0}
              wrap
              label="System Settings — Hyprland + AGS"
            />
            <NavItem id="overview" label="Overview" icon="preferences-system-symbolic" />

            <label class="nav-group" xalign={0} label="Appearance" />
            <NavItem id="appearance" label="Theme and Style" icon="preferences-desktop-theme-symbolic" />
            <NavItem id="wallpaper" label="Wallpaper" icon="folder-pictures-symbolic" />

            <label class="nav-group" xalign={0} label="Workspace" />
            <NavItem id="desktops" label="Virtual Desktops" icon="view-grid-symbolic" />
            <NavItem id="windows" label="Window Management" icon="window-new-symbolic" />
            <NavItem id="effects" label="Effects" icon="applications-graphics-symbolic" />
            <NavItem id="lock" label="Screen Locking" icon="system-lock-screen-symbolic" />
            <NavItem id="shortcuts" label="Shortcuts" icon="input-keyboard-symbolic" />
            <NavItem id="startup" label="Startup and Shutdown" icon="system-run-symbolic" />
            <NavItem id="search" label="Search" icon="system-search-symbolic" />

            <label class="nav-group" xalign={0} label="Personalization" />
            <NavItem id="notifications" label="Notifications" icon="preferences-system-notifications-symbolic" />
            <NavItem id="regional" label="Regional Settings" icon="preferences-desktop-locale-symbolic" />
            <NavItem id="users" label="Users" icon="system-users-symbolic" />
            <NavItem id="accessibility" label="Accessibility" icon="preferences-desktop-accessibility-symbolic" />
            <NavItem id="applications" label="Applications" icon="application-x-executable-symbolic" />

            <label class="nav-group" xalign={0} label="Network" />
            <NavItem id="network" label="Connections" icon="network-wireless-symbolic" />

            <label class="nav-group" xalign={0} label="Hardware" />
            <NavItem id="display" label="Display and Monitor" icon="video-display-symbolic" />
            <NavItem id="sound" label="Audio" icon="audio-headphones-symbolic" />
            <NavItem id="bluetooth" label="Bluetooth" icon="bluetooth-symbolic" />
            <NavItem id="input" label="Input Devices" icon="input-mouse-symbolic" />
            <NavItem id="power" label="Power Management" icon="battery-symbolic" />
            <NavItem id="storage" label="Removable Storage" icon="drive-harddisk-symbolic" />

            <label class="nav-group" xalign={0} label="System Administration" />
            <NavItem id="about" label="About" icon="dialog-information-symbolic" />
          </box>
        </Gtk.ScrolledWindow>
        <Gtk.Separator orientation={Gtk.Orientation.VERTICAL} />
        <Gtk.ScrolledWindow class="settings-page-scroll" hexpand vexpand>
          <box
            class="settings-page"
            orientation={Gtk.Orientation.VERTICAL}
            spacing={12}
            hexpand
          >
            <box
              visible={page((p) => p === "overview")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Overview"
                body="Quick status. Categories on the left match Plasma System Settings where this stack has an equivalent."
              />
              <Volume />
              <NetworkStatus />
              <BluetoothStatus />
              <label class="section" xalign={0} label="Open a module" />
              <box spacing={8}>
                {Jump("display", "Display")}
                {Jump("appearance", "Appearance")}
                {Jump("shortcuts", "Shortcuts")}
              </box>
              <box spacing={8}>
                {Jump("lock", "Lock screen")}
                {Jump("about", "About")}
                <button
                  hexpand
                  onClicked={() => {
                    hide()
                    launch("webcam")
                  }}
                >
                  <label label="Webcam" />
                </button>
              </box>
            </box>

            <box
              visible={page((p) => p === "appearance")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Theme and Style"
                body="GTK application style, colors, fonts, icons, and cursors. Floating titlebars are hyprbars, not KWin decorations."
              />
              <LaunchRow
                label="GTK theme, icons, fonts, cursors"
                hint="nwg-look"
                tool="appearance"
                hide={hide}
              />
              <label class="section" xalign={0} label="Plasma modules not on CodaLinux" />
              <SkipNote title="Global Theme" reason="No Plasma look-and-feel packages" />
              <SkipNote title="Plasma Style" reason="No Plasma desktop theme" />
              <SkipNote title="Window Decorations" reason="hyprbars titlebars; edit hyprland.lua" />
              <SkipNote title="Colors KCM" reason="Use nwg-look + Hyprland border colors" />
            </box>

            <box
              visible={page((p) => p === "wallpaper")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Wallpaper"
                body="Live wallpaper is the Plasma Horos still. coda-wallpaper tries hyprpaper, then swaybg. Do not overwrite default.png with gen-wallpaper.py."
              />
              <PollInfo cmd={["coda-settings", "wallpaper", "print"]} interval={8000} />
              <ActionRow
                label="Restore Horos wallpaper"
                hint="coda-wallpaper"
                onClicked={() => launch("wallpaper", "reset")}
              />
              <LaunchRow
                label="Open wallpaper folder"
                hint="thunar"
                tool="wallpaper"
                args={["open"]}
                hide={hide}
              />
            </box>

            <box
              visible={page((p) => p === "desktops")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Virtual Desktops"
                body="Hyprland workspaces. Super+N / Super+= add an empty workspace; Super+- removes an empty one. No Plasma Activities."
              />
              <PollInfo cmd={["coda-settings", "workspace", "print"]} />
              <box spacing={8}>
                <button hexpand onClicked={() => launch("workspace", "add")}>
                  <label label="Add workspace" />
                </button>
                <button hexpand onClicked={() => launch("workspace", "remove")}>
                  <label label="Remove empty" />
                </button>
              </box>
              <SkipNote title="Activities" reason="KDE Activities are not used" />
            </box>

            <box
              visible={page((p) => p === "windows")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Window Management"
                body="Tile vs overlapping float (Stack) is workspace-wide via coda-hypr-ws. Task switcher is Super+Tab. Window rules live in hyprland.lua."
              />
              <PollInfo cmd={["coda-settings", "workspace", "print"]} />
              <ActionRow
                label="Tile ↔ stack on this workspace"
                hint="coda-hypr-ws"
                onClicked={() => launch("workspace", "toggle-stack")}
              />
              <label
                class="display-info"
                wrap
                xalign={0}
                label={"Resize: grab borders/corners, Super+RMB, or Alt+RMB.\nTitlebar (float): close / maximize / minimize; scroll to shade."}
              />
              <SkipNote title="KWin Scripts / Rules KCM" reason="Edit desktop/hypr/hyprland.lua" />
            </box>

            <box
              visible={page((p) => p === "effects")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Desktop Effects"
                body="Hyprland animations and gaps. Blur stays off on the live VM pixman path. No KWin effect pack."
              />
              <PollInfo cmd={["coda-settings", "hypr", "print"]} />
              <box spacing={8}>
                <button hexpand onClicked={() => launch("hypr", "animations", "on")}>
                  <label label="Animations on" />
                </button>
                <button hexpand onClicked={() => launch("hypr", "animations", "off")}>
                  <label label="Animations off" />
                </button>
              </box>
              <label class="section" xalign={0} label="Gaps and borders" />
              <box spacing={8}>
                <button hexpand onClicked={() => launch("hypr", "gaps-in", "2")}>
                  <label label="Inner +" />
                </button>
                <button hexpand onClicked={() => launch("hypr", "gaps-in", "-2")}>
                  <label label="Inner −" />
                </button>
                <button hexpand onClicked={() => launch("hypr", "gaps-out", "2")}>
                  <label label="Outer +" />
                </button>
                <button hexpand onClicked={() => launch("hypr", "gaps-out", "-2")}>
                  <label label="Outer −" />
                </button>
              </box>
              <box spacing={8}>
                <button hexpand onClicked={() => launch("hypr", "border", "1")}>
                  <label label="Border +" />
                </button>
                <button hexpand onClicked={() => launch("hypr", "border", "-1")}>
                  <label label="Border −" />
                </button>
              </box>
              <SkipNote title="Screen edges" reason="No Plasma screen-edge KCM" />
            </box>

            <box
              visible={page((p) => p === "lock")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Screen Locking"
                body="Super+L runs coda-hyprlock. Live hypridle does not lock or DPMS-off on idle — hyprlock dies under VirtualBox/pixman."
              />
              <LaunchRow
                label="Lock now"
                hint="coda-hyprlock"
                tool="lock"
                hide={hide}
              />
              <label
                class="display-info"
                wrap
                xalign={0}
                label={"Recover a crashed lockscreen:\nhyprctl --instance 0 eval 'hl.clear_crashed_lockscreen()'\nkillall -9 hyprlock"}
              />
            </box>

            <box
              visible={page((p) => p === "shortcuts")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Shortcuts"
                body="Bindings are in hyprland.lua. Super is the modifier. This is not the Plasma Shortcuts KCM."
              />
              <label
                class="display-info"
                wrap
                xalign={0}
                label={`Super+Return  Terminal\nSuper+Space   App launcher\nSuper+,       Settings\nSuper+V       Clipboard\nSuper+L       Lock\nSuper+T       Tile ↔ stack\nSuper+N / −   Add / remove workspace\nSuper+Tab     Next window\nTitlebar      close / maximize / minimize; scroll to shade`}
              />
              <LaunchRow
                label="Full keyboard and touchpad help"
                hint="input-help"
                tool="input"
                hide={hide}
              />
            </box>

            <box
              visible={page((p) => p === "startup")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Startup and Shutdown"
                body="greetd starts the session. Not SDDM. No Plasma Autostart / session restore KCM."
              />
              <PollInfo cmd={["coda-settings", "startup", "print"]} interval={15000} />
              <SkipNote title="SDDM" reason="Login is greetd + coda-hyprland" />
              <SkipNote title="Autostart" reason="Add hl.exec_cmd in hyprland.lua" />
            </box>

            <box
              visible={page((p) => p === "search")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Search"
                body="Application search is the AGS launcher (Super+Space / Super+D). No KRunner, Baloo, or Plasma Search KCM."
              />
              <button
                hexpand
                onClicked={() => {
                  hide()
                  const launcher = app.get_window("launcher")
                  if (launcher) launcher.visible = true
                }}
              >
                <label label="Open application launcher" />
              </button>
            </box>

            <box
              visible={page((p) => p === "notifications")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Notifications"
                body="AGS Astal notifd draws the popups. This is not Plasma Notifications."
              />
              <box class="status-row" spacing={10} hexpand>
                <label
                  hexpand
                  xalign={0}
                  label={dnd((d) => (d ? "Do not disturb is on" : "Do not disturb is off"))}
                />
                <button onClicked={() => notifd.set_dont_disturb(!notifd.dontDisturb)}>
                  <label label={dnd((d) => (d ? "Allow" : "Mute"))} />
                </button>
              </box>
              <ActionRow
                label="Send a test notification"
                hint="notify-send"
                onClicked={() => launch("notify", "test")}
              />
            </box>

            <box
              visible={page((p) => p === "regional")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Regional Settings"
                body="Bozeman, Montana defaults. Locale, timezone, and keymap are not asked at install and are not changed here."
              />
              <PollInfo cmd={["coda-settings", "region", "print"]} interval={15000} />
            </box>

            <box
              visible={page((p) => p === "users")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Users"
                body="Live ISO autologins user live. There is no Plasma Users KCM."
              />
              <PollInfo cmd={["coda-settings", "users", "print"]} interval={15000} />
              <SkipNote
                title="KDE Wallet / Online Accounts / Feedback"
                reason="No wallet, accounts, or telemetry modules"
              />
            </box>

            <box
              visible={page((p) => p === "accessibility")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Accessibility"
                body="Touchpad defaults are natural scroll and tap-to-click. Screen reader and magnifier are not on the live image."
              />
              <label
                class="display-info"
                wrap
                xalign={0}
                label={"Keyboard layout: us\nTouchpad: natural scroll on, tap to click on\nModifier: Super"}
              />
              <SkipNote title="Orca / KMag" reason="Not packaged on the v1 live ISO" />
            </box>

            <box
              visible={page((p) => p === "applications")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Default Applications"
                body="xdg-mime / Thunar file associations. No Plasma Applications KCM."
              />
              <PollInfo cmd={["coda-settings", "apps", "print"]} interval={10000} />
              <LaunchRow
                label="Open home folder"
                hint="thunar"
                tool="apps"
                args={["files"]}
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
                title="Connections"
                body="systemd-networkd + iwd. There is no NetworkManager, plasma-nm, or proxy KCM. Join Wi-Fi with impala."
              />
              <NetworkStatus />
              <LaunchRow label="Wi-Fi" hint="impala" tool="wifi" hide={hide} />
              <LaunchRow
                label="Network details"
                hint="networkctl"
                tool="netstatus"
                hide={hide}
              />
            </box>

            <box visible={page((p) => p === "display")} hexpand>
              <DisplayPanel hide={hide} />
            </box>

            <box
              visible={page((p) => p === "sound")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Audio"
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
              visible={page((p) => p === "input")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Input Devices"
                body="Keyboard layout is us. Super is the modifier. Touchpad: natural scroll and tap-to-click. Change compositor input in hyprland.lua."
              />
              <label
                class="display-info"
                wrap
                xalign={0}
                label={"Keyboard: us (Bozeman default)\nMouse: follow_mouse, border resize\nTouchpad: natural_scroll, tap_to_click"}
              />
              <LaunchRow
                label="Full keyboard and touchpad help"
                hint="shortcuts"
                tool="input"
                hide={hide}
              />
              <LaunchRow
                label="GTK cursors (nwg-look)"
                hint="appearance"
                tool="appearance"
                hide={hide}
              />
            </box>

            <box
              visible={page((p) => p === "power")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Power Management"
                body="Brightness via brightnessctl (XF86 keys). Battery via UPower when a battery is present. No TLP or power-profiles-daemon."
              />
              <box
                class="status-row"
                spacing={10}
                hexpand
                visible={battPresent ? battPresent : false}
              >
                <image iconName="battery-symbolic" pixelSize={18} />
                <label hexpand xalign={0} label={battPct ? battPct : "No battery"} />
              </box>
              <PollInfo cmd={["coda-settings", "power", "print"]} interval={8000} />
              <box spacing={8}>
                <button hexpand onClicked={() => launch("power", "brightness", "down")}>
                  <label label="Brightness −" />
                </button>
                <button hexpand onClicked={() => launch("power", "brightness", "up")}>
                  <label label="Brightness +" />
                </button>
              </box>
            </box>

            <box
              visible={page((p) => p === "storage")}
              orientation={Gtk.Orientation.VERTICAL}
              spacing={10}
              hexpand
            >
              <PageHeader
                title="Removable Storage"
                body="udisks + Thunar. No Plasma Device Notifier."
              />
              <PollInfo cmd={["coda-settings", "storage", "print"]} interval={8000} />
              <LaunchRow
                label="Open media folder"
                hint="thunar"
                tool="storage"
                args={["open"]}
                hide={hide}
              />
              <LaunchRow
                label="Webcam"
                hint="snapshot"
                tool="webcam"
                hide={hide}
              />
            </box>

            <box visible={page((p) => p === "about")} hexpand>
              <AboutPanel />
            </box>
          </box>
        </Gtk.ScrolledWindow>
      </box>
    </window>
  )
}
