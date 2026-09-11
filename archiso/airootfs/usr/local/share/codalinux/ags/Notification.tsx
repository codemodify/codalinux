import Gtk from "gi://Gtk?version=4.0"
import Gdk from "gi://Gdk?version=4.0"
import GLib from "gi://GLib"
import AstalNotifd from "gi://AstalNotifd"
import Pango from "gi://Pango"

function isIcon(icon?: string | null) {
  const display = Gdk.Display.get_default()
  if (!display || !icon) return false
  return Gtk.IconTheme.get_for_display(display).has_icon(icon)
}

function fileExists(path: string) {
  return GLib.file_test(path, GLib.FileTest.EXISTS)
}

function time(unix: number, format = "%H:%M") {
  return GLib.DateTime.new_from_unix_local(unix).format(format)!
}

function urgency(n: AstalNotifd.Notification) {
  const { LOW, NORMAL, CRITICAL } = AstalNotifd.Urgency
  switch (n.urgency) {
    case LOW:
      return "low"
    case CRITICAL:
      return "critical"
    case NORMAL:
    default:
      return "normal"
  }
}

export default function Notification({
  notification: n,
}: {
  notification: AstalNotifd.Notification
}) {
  return (
    <box
      class={`Notification ${urgency(n)}`}
      orientation={Gtk.Orientation.VERTICAL}
      widthRequest={360}
    >
      <box class="header" spacing={8}>
        {(n.appIcon || isIcon(n.desktopEntry)) && (
          <image
            class="app-icon"
            iconName={n.appIcon || n.desktopEntry}
            pixelSize={16}
          />
        )}
        <label
          class="app-name"
          hexpand
          xalign={0}
          ellipsize={Pango.EllipsizeMode.END}
          label={n.appName || "Notification"}
        />
        <label class="time" label={time(n.time)} />
        <button class="close" onClicked={() => n.dismiss()}>
          <image iconName="window-close-symbolic" pixelSize={12} />
        </button>
      </box>
      <Gtk.Separator visible />
      <box class="content" spacing={10}>
        {n.image && fileExists(n.image) && (
          <image
            class="image"
            file={n.image}
            pixelSize={64}
          />
        )}
        {n.image && !fileExists(n.image) && isIcon(n.image) && (
          <image class="image" iconName={n.image} pixelSize={48} />
        )}
        <box orientation={Gtk.Orientation.VERTICAL} spacing={4} hexpand>
          <label
            class="summary"
            xalign={0}
            wrap
            label={n.summary}
          />
          {n.body && (
            <label
              class="body"
              xalign={0}
              wrap
              label={n.body}
            />
          )}
        </box>
      </box>
      {n.actions.length > 0 && (
        <box class="actions" spacing={6}>
          {n.actions.map(({ label, id }) => (
            <button hexpand onClicked={() => n.invoke(id)}>
              <label label={label} />
            </button>
          ))}
        </box>
      )}
    </box>
  )
}
