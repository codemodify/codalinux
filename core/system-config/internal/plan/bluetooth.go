package plan

import "github.com/codemodify/codalinux/core/system-config/internal/protocol"

func FromBluetooth(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.BluetoothModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	if jsonHas(desired, "powered") && want.Powered != have.Powered {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpBTPower, Enabled: boolPtr(want.Powered)})
	}
	if jsonHas(desired, "scanning") && want.Scanning != have.Scanning {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpBTScan, Enabled: boolPtr(want.Scanning)})
	}
	for _, addr := range want.Pair {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpBTPair, Device: addr})
	}
	for _, addr := range want.Connect {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpBTConnect, Device: addr})
	}
	for _, addr := range want.Disconnect {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpBTDisconnect, Device: addr})
	}
	for _, addr := range want.Trust {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpBTTrust, Device: addr})
	}
	return protocol.Plan{Path: protocol.PathBluetooth, Ops: ops}, nil
}
