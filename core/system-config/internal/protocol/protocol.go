// Package protocol is the v1 JSON-lines API between clients, D, report, and apply.
package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const Version = 1

// Ops clients may send to system-configd.
const (
	OpGet         = "get"
	OpSet         = "set"
	OpWatch       = "watch"
	OpRefresh     = "refresh"
	OpApply       = "apply"
	OpPutObserved = "put-observed" // report → D
	OpScan        = "scan"         // D → report
	OpExec        = "exec"         // D → apply
)

// Submodel paths (architecture.md). Clients query these, not the whole tree.
const (
	PathDisplay        = "display"
	PathNetwork        = "network"
	PathAudio          = "audio"
	PathBluetooth      = "bluetooth"
	PathInput          = "input"
	PathDateTime       = "datetime"
	PathLocale         = "locale"
	PathDevicesSummary = "devices.summary"
	PathDevicesPCI     = "devices.pci"
	PathDevicesUSB     = "devices.usb"
	PathHardwareDMI    = "hardware.dmi"
	PathSession        = "session"
	PathPower          = "power"
	PathSubmodels      = "submodels"
)

// KnownPaths is every submodel D/report understand.
var KnownPaths = []string{
	PathDisplay,
	PathNetwork,
	PathAudio,
	PathBluetooth,
	PathInput,
	PathDateTime,
	PathLocale,
	PathDevicesSummary,
	PathDevicesPCI,
	PathDevicesUSB,
	PathHardwareDMI,
	PathSession,
	PathPower,
}

// StarterPaths is the list advertised on get submodels (alias of KnownPaths).
var StarterPaths = KnownPaths

type Request struct {
	ID      string          `json:"id"`
	Op      string          `json:"op"`
	Path    string          `json:"path,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Plan    *Plan           `json:"plan,omitempty"`
	Version int             `json:"v,omitempty"`
}

type Response struct {
	ID       string          `json:"id"`
	OK       bool            `json:"ok"`
	Error    string          `json:"error,omitempty"`
	Path     string          `json:"path,omitempty"`
	Desired  json.RawMessage `json:"desired,omitempty"`
	Observed json.RawMessage `json:"observed,omitempty"`
	Status   *Status         `json:"status,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
	Note     string          `json:"note,omitempty"`
}

type Status struct {
	Present    bool   `json:"present"`
	Configured bool   `json:"configured"`
	Changed    bool   `json:"changed"`
	ApplyError string `json:"apply_error,omitempty"`
}

type Plan struct {
	ID   string   `json:"id"`
	Path string   `json:"path"`
	Ops  []PlanOp `json:"ops"`
}

// PlanOp is a closed allowlisted action. Extra fields are ignored by unused ops.
type PlanOp struct {
	Type      string   `json:"type"`
	Output    string   `json:"output,omitempty"`
	Scale     float64  `json:"scale,omitempty"`
	Mode      string   `json:"mode,omitempty"`
	Device    string   `json:"device,omitempty"`
	Name      string   `json:"name,omitempty"`
	Value     string   `json:"value,omitempty"`
	SSID      string   `json:"ssid,omitempty"`
	PSK       string   `json:"psk,omitempty"`
	Method    string   `json:"method,omitempty"`
	Address   string   `json:"address,omitempty"`
	Gateway   string   `json:"gateway,omitempty"`
	DNS       []string `json:"dns,omitempty"`
	Volume    float64  `json:"volume,omitempty"`
	Mute      *bool    `json:"mute,omitempty"`
	Enabled   *bool    `json:"enabled,omitempty"`
	Action    string   `json:"action,omitempty"`
	ID        string   `json:"id,omitempty"`
	Speed     float64  `json:"speed,omitempty"`
	AddressBT string   `json:"bt_address,omitempty"`
}

const (
	OpDisplayScale = "display.scale"
	OpDisplayMode  = "display.mode"

	OpNetIfaceEnable    = "network.iface.enable"
	OpNetIfaceMethod    = "network.iface.method"
	OpNetWiFiConnect    = "network.wifi.connect"
	OpNetWiFiDisconnect = "network.wifi.disconnect"

	OpAudioDefaultSink   = "audio.default.sink"
	OpAudioDefaultSource = "audio.default.source"
	OpAudioVolume        = "audio.volume"
	OpAudioMute          = "audio.mute"

	OpBTPower      = "bluetooth.power"
	OpBTScan       = "bluetooth.scan"
	OpBTPair       = "bluetooth.pair"
	OpBTConnect    = "bluetooth.connect"
	OpBTDisconnect = "bluetooth.disconnect"
	OpBTTrust      = "bluetooth.trust"

	OpInputKeymap        = "input.keymap"
	OpInputKBLayout      = "input.kb_layout"
	OpInputPointerSpeed  = "input.pointer.speed"
	OpInputNaturalScroll = "input.pointer.natural_scroll"
	OpInputTapToClick    = "input.touchpad.tap"

	OpDateTimeTimezone = "datetime.timezone"
	OpDateTimeNTP      = "datetime.ntp"
	OpDateTimeTime     = "datetime.time"

	OpLocaleLang   = "locale.lang"
	OpLocaleKeymap = "locale.keymap"

	OpSessionLock = "session.lock"

	OpPowerSuspend    = "power.suspend"
	OpPowerHibernate  = "power.hibernate"
	OpPowerBrightness = "power.brightness"
	OpPowerLid        = "power.lid"
)

// ApplyOps is the closed allowlist executed by system-config-apply.
var ApplyOps = []string{
	OpDisplayScale, OpDisplayMode,
	OpNetIfaceEnable, OpNetIfaceMethod, OpNetWiFiConnect, OpNetWiFiDisconnect,
	OpAudioDefaultSink, OpAudioDefaultSource, OpAudioVolume, OpAudioMute,
	OpBTPower, OpBTScan, OpBTPair, OpBTConnect, OpBTDisconnect, OpBTTrust,
	OpInputKeymap, OpInputKBLayout, OpInputPointerSpeed, OpInputNaturalScroll, OpInputTapToClick,
	OpDateTimeTimezone, OpDateTimeNTP, OpDateTimeTime,
	OpLocaleLang, OpLocaleKeymap,
	OpSessionLock,
	OpPowerSuspend, OpPowerHibernate, OpPowerBrightness, OpPowerLid,
}

func ApplyOpsLog() string {
	return strings.Join(ApplyOps, " ")
}

func AllowedOp(t string) bool {
	for _, a := range ApplyOps {
		if a == t {
			return true
		}
	}
	return false
}

func NormalizePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "/")
	return strings.ToLower(p)
}

func KnownPath(p string) bool {
	p = NormalizePath(p)
	if p == "" || p == PathSubmodels {
		return true
	}
	for _, k := range KnownPaths {
		if p == k {
			return true
		}
	}
	return false
}

func Settable(p string) bool {
	switch NormalizePath(p) {
	case PathDisplay, PathNetwork, PathAudio, PathBluetooth, PathInput,
		PathDateTime, PathLocale, PathSession, PathPower:
		return true
	default:
		return false
	}
}

func Encode(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func DecodeRequest(line []byte) (Request, error) {
	var r Request
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return r, fmt.Errorf("empty request")
	}
	if err := json.Unmarshal(line, &r); err != nil {
		return r, err
	}
	r.Path = NormalizePath(r.Path)
	if r.Version == 0 {
		r.Version = Version
	}
	return r, nil
}

func DecodeResponse(line []byte) (Response, error) {
	var r Response
	line = bytes.TrimSpace(line)
	if err := json.Unmarshal(line, &r); err != nil {
		return r, err
	}
	return r, nil
}
