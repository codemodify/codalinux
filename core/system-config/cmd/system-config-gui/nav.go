package main

import "github.com/codemodify/codalinux/core/system-config/internal/protocol"

// navItem is one Settings section. PrefsPage / NavRail do not exist on
// uitoolkit@dev (current pin), so the GUI composes a source-list rail from
// these rows: ListView + group caption on the page, per-section Apply.
type navItem struct {
	Label string
	Path  string
	Group string
	Blurb string
}

// nav is the Settings rail. Order is prefs-like (hardware → links →
// session → system), not the Mail TreeView dump of KnownPaths.
var nav = []navItem{
	{
		Label: "Display", Path: protocol.PathDisplay, Group: "Hardware",
		Blurb: "Stage scale / mode / position, then Apply. D runs hyprctl eval hl.monitor (not keyword).",
	},
	{
		Label: "Input", Path: protocol.PathInput, Group: "Hardware",
		Blurb: "XKB / vconsole via localectl + Hyprland hl.input eval.",
	},
	{
		Label: "Devices", Path: protocol.PathDevicesSummary, Group: "Hardware",
		Blurb: "PCI / USB / DMI inventory. Observe only — Refresh reloads the probe.",
	},
	{
		Label: "Network", Path: protocol.PathNetwork, Group: "Connectivity",
		Blurb: "systemd-networkd + iwd. Static IP writes a drop-in or 20-coda-*.network (survives reboot). No NetworkManager.",
	},
	{
		Label: "Audio", Path: protocol.PathAudio, Group: "Connectivity",
		Blurb: "PipeWire via pw-dump / wpctl. Default sink/source persist to ~/.config/wireplumber/wireplumber.conf.d/51-coda-defaults.conf.",
	},
	{
		Label: "Bluetooth", Path: protocol.PathBluetooth, Group: "Connectivity",
		Blurb: "BlueZ D-Bus pairing agent. Pair opens a PIN dialog; Apply writes $XDG_RUNTIME_DIR/coda/bluetooth-pin and the agent reads it (or waits up to 12s).",
	},
	{
		Label: "Date & time", Path: protocol.PathDateTime, Group: "Session",
		Blurb: "timedatectl: timezone, NTP, optional manual time.",
	},
	{
		Label: "Locale", Path: protocol.PathLocale, Group: "Session",
		Blurb: "localectl / locale.conf. Apply writes LANG and keymap.",
	},
	{
		Label: "Session", Path: protocol.PathSession, Group: "Session",
		Blurb: "logind sessions + seats. Apply = lock only (loginctl lock-sessions, then coda-hyprlock). Reboot/poweroff are not allowlisted.",
	},
	{
		Label: "Power", Path: protocol.PathPower, Group: "Session",
		Blurb: "Brightness when a backlight exists. Lid HandleLidSwitch via logind drop-in. Suspend/hibernate are gated; reboot/poweroff are not allowlisted.",
	},
	{
		Label: "Printers", Path: protocol.PathPrinters, Group: "System",
		Blurb: "CUPS via lpstat / lpadmin. present=false when cups is missing. Apply sets default and enable.",
	},
	{
		Label: "Users", Path: protocol.PathUsers, Group: "System",
		Blurb: "Local accounts from /etc/passwd. Apply changes login shell only (usermod -s). No add/delete/password.",
	},
	{
		Label: "Storage", Path: protocol.PathStorage, Group: "System",
		Blurb: "lsblk / udisks. Apply mount/unmount via udisksctl. System mounts (/, /boot, /usr, /home) are refused.",
	},
}

func pageMeta(path string) navItem {
	for _, n := range nav {
		if n.Path == path {
			return n
		}
	}
	return navItem{Label: path, Path: path, Blurb: path}
}

func navIndex(path string) int {
	for i, n := range nav {
		if n.Path == path {
			return i
		}
	}
	return -1
}

func navGroups() []string {
	var out []string
	seen := map[string]bool{}
	for _, n := range nav {
		if n.Group == "" || seen[n.Group] {
			continue
		}
		seen[n.Group] = true
		out = append(out, n.Group)
	}
	return out
}
