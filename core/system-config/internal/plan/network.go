package plan

import (
	"encoding/json"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func FromNetwork(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.NetworkModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	haveBy := map[string]protocol.NetLink{}
	for _, l := range have.Links {
		haveBy[l.Name] = l
	}
	var ops []protocol.PlanOp
	for i, w := range want.Links {
		if w.Name == "" {
			continue
		}
		h := haveBy[w.Name]
		if linkJSONHas(desired, i, "enabled") && w.Enabled != h.Enabled {
			ops = append(ops, protocol.PlanOp{
				Type: protocol.OpNetIfaceEnable, Device: w.Name, Enabled: boolPtr(w.Enabled),
			})
		}
		if w.Method != "" && (w.Method != h.Method || staticChanged(w, h)) {
			ops = append(ops, protocol.PlanOp{
				Type: protocol.OpNetIfaceMethod, Device: w.Name, Method: w.Method,
				Address: firstAddr(w.Addresses), Gateway: w.Gateway, DNS: w.DNS, Search: w.Search,
			})
		}
	}
	if jsonHas(desired, "airplane") && want.Airplane != have.Airplane {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpNetAirplane, Enabled: boolPtr(want.Airplane)})
	}
	dev := want.WiFi.Device
	if dev == "" {
		dev = have.WiFi.Device
	}
	if want.WiFi.Disconnect && have.WiFi.Connected != "" {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpNetWiFiDisconnect, Device: dev})
	}
	if want.WiFi.Connect != "" && want.WiFi.Connect != have.WiFi.Connected {
		ops = append(ops, protocol.PlanOp{
			Type: protocol.OpNetWiFiConnect, Device: dev, SSID: want.WiFi.Connect, PSK: want.WiFi.PSK,
			Hidden: want.WiFi.Hidden,
		})
	}
	return protocol.Plan{Path: protocol.PathNetwork, Ops: ops}, nil
}

func linkJSONHas(desired []byte, i int, key string) bool {
	var wrap struct {
		Links []json.RawMessage `json:"links"`
	}
	if json.Unmarshal(desired, &wrap) != nil {
		return false
	}
	if i < 0 || i >= len(wrap.Links) {
		return false
	}
	return jsonHas(wrap.Links[i], key)
}

func firstAddr(a []string) string {
	if len(a) == 0 {
		return ""
	}
	return a[0]
}

func staticChanged(w, h protocol.NetLink) bool {
	if w.Method != "static" {
		return false
	}
	if w.Gateway != "" && w.Gateway != h.Gateway {
		return true
	}
	if a := firstAddr(w.Addresses); a != "" && (len(h.Addresses) == 0 || a != h.Addresses[0]) {
		return true
	}
	if !sameStrings(w.DNS, h.DNS) {
		return true
	}
	if !sameStrings(w.Search, h.Search) {
		return true
	}
	return false
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return len(a) == 0 && len(b) == 0
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
