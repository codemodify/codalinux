package main

import (
	"slices"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

// staged is the editable UI state we persist across sidebar navigation.
// Each settable section compares only its own fields against base.
type staged struct {
	scale   float64
	outName string
	outMode string
	outPos  string

	airplane   bool
	wifiDev    string
	wifiSSID   string
	wifiPSK    string
	wifiHidden bool
	wifiDisc   bool
	iface      string
	netMethod  string
	netAddr    string
	netGW      string
	netDNS     string
	netSearch  string

	vol    float64
	mute   bool
	sink   string
	source string

	btPower      bool
	btScan       bool
	btAddr       string
	btPIN        string
	btPair       []string
	btConnect    []string
	btDisconnect []string
	btTrust      []string

	input protocol.InputModel
	dt    protocol.DateTimeModel
	loc   protocol.Locale

	sessionAct string

	bright   float64
	lid      string
	powerAct string

	printerName string
	printerOn   bool
	userName    string
	userShell   string
	storageName string
	storageAct  string
}

func (s *session) capture() staged {
	return staged{
		scale: s.scale, outName: s.outName, outMode: s.outMode, outPos: s.outPos,
		airplane: s.airplane, wifiDev: s.wifiDev, wifiSSID: s.wifiSSID, wifiPSK: s.wifiPSK,
		wifiHidden: s.wifiHidden, wifiDisc: s.net.WiFi.Disconnect,
		iface: s.iface, netMethod: s.netMethod, netAddr: s.netAddr, netGW: s.netGW,
		netDNS: s.netDNS, netSearch: s.netSearch,
		vol: s.vol, mute: s.mute, sink: s.audio.DefaultSink, source: s.audio.DefaultSource,
		btPower: s.btPower, btScan: s.btScan, btAddr: s.btAddr, btPIN: s.btPIN,
		btPair: slices.Clone(s.btPair), btConnect: slices.Clone(s.btConnect),
		btDisconnect: slices.Clone(s.btDisconnect), btTrust: slices.Clone(s.btTrust),
		input: s.input, dt: s.dt, loc: s.loc,
		sessionAct: s.sessionAct,
		bright:     s.bright, lid: s.power.Lid, powerAct: s.power.Action,
		printerName: s.printerName, printerOn: s.printerOn,
		userName: s.userName, userShell: s.userShell,
		storageName: s.storageName, storageAct: s.storageAct,
	}
}

func copyPath(dst *staged, src staged, path string) {
	switch path {
	case protocol.PathDisplay:
		dst.scale, dst.outName, dst.outMode, dst.outPos = src.scale, src.outName, src.outMode, src.outPos
	case protocol.PathNetwork:
		dst.airplane, dst.wifiDev, dst.wifiSSID, dst.wifiPSK = src.airplane, src.wifiDev, src.wifiSSID, src.wifiPSK
		dst.wifiHidden, dst.wifiDisc = src.wifiHidden, src.wifiDisc
		dst.iface, dst.netMethod, dst.netAddr, dst.netGW = src.iface, src.netMethod, src.netAddr, src.netGW
		dst.netDNS, dst.netSearch = src.netDNS, src.netSearch
	case protocol.PathAudio:
		dst.vol, dst.mute, dst.sink, dst.source = src.vol, src.mute, src.sink, src.source
	case protocol.PathBluetooth:
		dst.btPower, dst.btScan, dst.btAddr, dst.btPIN = src.btPower, src.btScan, src.btAddr, src.btPIN
		dst.btPair = slices.Clone(src.btPair)
		dst.btConnect = slices.Clone(src.btConnect)
		dst.btDisconnect = slices.Clone(src.btDisconnect)
		dst.btTrust = slices.Clone(src.btTrust)
	case protocol.PathInput:
		dst.input = src.input
	case protocol.PathDateTime:
		dst.dt = src.dt
	case protocol.PathLocale:
		dst.loc = src.loc
	case protocol.PathSession:
		dst.sessionAct = src.sessionAct
	case protocol.PathPower:
		dst.bright, dst.lid, dst.powerAct = src.bright, src.lid, src.powerAct
	case protocol.PathPrinters:
		dst.printerName, dst.printerOn = src.printerName, src.printerOn
	case protocol.PathUsers:
		dst.userName, dst.userShell = src.userName, src.userShell
	case protocol.PathStorage:
		dst.storageName, dst.storageAct = src.storageName, src.storageAct
	}
}

func (s *session) snapshot(path string) {
	copyPath(&s.base, s.capture(), path)
}

func (s *session) snapshotAll() {
	s.base = s.capture()
}

func near(a, b float64) bool { return abs(a-b) <= 0.01 }

func (s *session) dirtyPath(path string) bool {
	if s.cli == nil || !protocol.Settable(path) {
		return false
	}
	cur, base := s.capture(), s.base
	switch path {
	case protocol.PathDisplay:
		return !near(cur.scale, base.scale) || cur.outName != base.outName ||
			cur.outMode != base.outMode || cur.outPos != base.outPos
	case protocol.PathNetwork:
		return cur.airplane != base.airplane || cur.wifiDev != base.wifiDev ||
			cur.wifiSSID != base.wifiSSID || cur.wifiPSK != base.wifiPSK ||
			cur.wifiHidden != base.wifiHidden || cur.wifiDisc != base.wifiDisc ||
			cur.iface != base.iface || cur.netMethod != base.netMethod ||
			cur.netAddr != base.netAddr || cur.netGW != base.netGW ||
			cur.netDNS != base.netDNS || cur.netSearch != base.netSearch
	case protocol.PathAudio:
		return !near(cur.vol, base.vol) || cur.mute != base.mute ||
			cur.sink != base.sink || cur.source != base.source
	case protocol.PathBluetooth:
		return cur.btPower != base.btPower || cur.btScan != base.btScan ||
			cur.btPIN != base.btPIN ||
			!slices.Equal(cur.btPair, base.btPair) ||
			!slices.Equal(cur.btConnect, base.btConnect) ||
			!slices.Equal(cur.btDisconnect, base.btDisconnect) ||
			!slices.Equal(cur.btTrust, base.btTrust)
	case protocol.PathInput:
		return cur.input.KBLayout != base.input.KBLayout ||
			cur.input.Keymap != base.input.Keymap ||
			!near(cur.input.PointerSpeed, base.input.PointerSpeed) ||
			cur.input.NaturalScroll != base.input.NaturalScroll ||
			cur.input.TapToClick != base.input.TapToClick
	case protocol.PathDateTime:
		return cur.dt.Timezone != base.dt.Timezone || cur.dt.NTP != base.dt.NTP ||
			cur.dt.Time != base.dt.Time
	case protocol.PathLocale:
		return cur.loc.Lang != base.loc.Lang || cur.loc.Keymap != base.loc.Keymap ||
			cur.loc.Timezone != base.loc.Timezone
	case protocol.PathSession:
		return strings.TrimSpace(cur.sessionAct) != "" && cur.sessionAct != base.sessionAct
	case protocol.PathPower:
		return !near(cur.bright, base.bright) || cur.lid != base.lid ||
			cur.powerAct != base.powerAct
	case protocol.PathPrinters:
		return cur.printerName != base.printerName || cur.printerOn != base.printerOn
	case protocol.PathUsers:
		return cur.userName != base.userName || cur.userShell != base.userShell
	case protocol.PathStorage:
		return cur.storageAct != "" && (cur.storageAct != base.storageAct ||
			cur.storageName != base.storageName)
	default:
		return false
	}
}

func (s *session) dirty() bool { return s.dirtyPath(s.path()) }

func (s *session) clearActions(path string) {
	switch path {
	case protocol.PathNetwork:
		s.net.WiFi.Disconnect = false
	case protocol.PathBluetooth:
		s.btPair, s.btConnect, s.btDisconnect, s.btTrust = nil, nil, nil, nil
		s.btPIN = ""
	case protocol.PathSession:
		s.sessionAct = ""
	case protocol.PathPower:
		s.power.Action = ""
	case protocol.PathStorage:
		s.storageAct = ""
	}
}
