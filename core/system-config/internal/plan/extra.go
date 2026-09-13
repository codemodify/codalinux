package plan

import "github.com/codemodify/codalinux/core/system-config/internal/protocol"

func FromPrinters(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.PrintersModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	def := want.Default
	if def == "" {
		for _, p := range want.Printers {
			if p.Default {
				def = p.Name
				break
			}
		}
	}
	if def != "" && def != have.Default {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpPrinterDefault, Name: def})
	}
	haveBy := map[string]protocol.Printer{}
	for _, p := range have.Printers {
		haveBy[p.Name] = p
	}
	for _, w := range want.Printers {
		if w.Name == "" || w.Enabled == nil {
			continue
		}
		h := haveBy[w.Name]
		if h.Enabled == nil || *h.Enabled != *w.Enabled {
			ops = append(ops, protocol.PlanOp{Type: protocol.OpPrinterEnable, Name: w.Name, Enabled: w.Enabled})
		}
	}
	return protocol.Plan{Path: protocol.PathPrinters, Ops: ops}, nil
}

func FromUsers(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.UsersModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	haveBy := map[string]protocol.LocalUser{}
	for _, u := range have.Users {
		haveBy[u.Name] = u
	}
	var ops []protocol.PlanOp
	for _, w := range want.Users {
		if w.Name == "" || w.Shell == "" {
			continue
		}
		h := haveBy[w.Name]
		if w.Shell != h.Shell {
			ops = append(ops, protocol.PlanOp{Type: protocol.OpUserShell, Name: w.Name, Value: w.Shell})
		}
	}
	return protocol.Plan{Path: protocol.PathUsers, Ops: ops}, nil
}

func FromStorage(desired, observed []byte) (protocol.Plan, error) {
	var want protocol.StorageModel
	if err := unmarshal(desired, observed, &want, &protocol.StorageModel{}); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	for _, w := range want.Block {
		if w.Name == "" {
			continue
		}
		switch w.Action {
		case "mount":
			ops = append(ops, protocol.PlanOp{Type: protocol.OpStorageMount, Device: w.Name})
		case "unmount":
			ops = append(ops, protocol.PlanOp{Type: protocol.OpStorageUnmount, Device: w.Name, Value: w.Mount})
		}
	}
	return protocol.Plan{Path: protocol.PathStorage, Ops: ops}, nil
}
