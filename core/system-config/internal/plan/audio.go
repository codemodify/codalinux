package plan

import "github.com/codemodify/codalinux/core/system-config/internal/protocol"

func FromAudio(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.AudioModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	// Always re-apply an explicit default so persistAudioDefault writes
	// WirePlumber conf even when routing already matches (Dummy Output).
	if jsonHas(desired, "default_sink") && want.DefaultSink != "" {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioDefaultSink, ID: want.DefaultSink, Name: want.DefaultSink})
	}
	if jsonHas(desired, "default_source") && want.DefaultSource != "" {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioDefaultSource, ID: want.DefaultSource, Name: want.DefaultSource})
	}
	if jsonHas(desired, "volume") && abs(want.Volume-haveDefaultVol(have)) > 0.005 {
		id := want.DefaultSink
		if id == "" {
			id = have.DefaultSink
		}
		ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioVolume, ID: id, Volume: want.Volume})
	}
	if want.Mute != nil && (defaultMute(have) != *want.Mute) {
		id := want.DefaultSink
		if id == "" {
			id = have.DefaultSink
		}
		ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioMute, ID: id, Mute: want.Mute})
	}
	for _, w := range want.Sinks {
		h := findNode(have.Sinks, w.ID, w.Name)
		if w.Default && !h.Default {
			ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioDefaultSink, ID: w.ID, Name: w.Name})
		}
		if w.Volume > 0 && abs(w.Volume-h.Volume) > 0.005 {
			ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioVolume, ID: w.ID, Name: w.Name, Volume: w.Volume})
		}
		if w.Mute != h.Mute {
			ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioMute, ID: w.ID, Name: w.Name, Mute: boolPtr(w.Mute)})
		}
	}
	for _, w := range want.Sources {
		h := findNode(have.Sources, w.ID, w.Name)
		if w.Default && !h.Default {
			ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioDefaultSource, ID: w.ID, Name: w.Name})
		}
		if w.Volume > 0 && abs(w.Volume-h.Volume) > 0.005 {
			ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioVolume, ID: w.ID, Name: w.Name, Volume: w.Volume})
		}
		if w.Mute != h.Mute {
			ops = append(ops, protocol.PlanOp{Type: protocol.OpAudioMute, ID: w.ID, Name: w.Name, Mute: boolPtr(w.Mute)})
		}
	}
	return protocol.Plan{Path: protocol.PathAudio, Ops: ops}, nil
}

func findNode(nodes []protocol.AudioNode, id, name string) protocol.AudioNode {
	for _, n := range nodes {
		if (id != "" && n.ID == id) || (name != "" && n.Name == name) {
			return n
		}
	}
	return protocol.AudioNode{}
}

func haveDefaultVol(have protocol.AudioModel) float64 {
	for _, n := range have.Sinks {
		if n.Default || n.ID == have.DefaultSink || n.Name == have.DefaultSink {
			return n.Volume
		}
	}
	return 0
}

func defaultMute(have protocol.AudioModel) bool {
	for _, n := range have.Sinks {
		if n.Default || n.ID == have.DefaultSink || n.Name == have.DefaultSink {
			return n.Mute
		}
	}
	return false
}
