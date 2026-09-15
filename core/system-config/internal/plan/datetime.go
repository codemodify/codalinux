package plan

import "github.com/codemodify/codalinux/core/system-config/internal/protocol"

func FromDateTime(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.DateTimeModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	if want.Timezone != "" && want.Timezone != have.Timezone {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpDateTimeTimezone, Value: want.Timezone})
	}
	if jsonHas(desired, "ntp") && want.NTP != have.NTP {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpDateTimeNTP, Enabled: boolPtr(want.NTP)})
	}
	if want.Time != "" && want.Time != have.Time {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpDateTimeTime, Value: want.Time})
	}
	return protocol.Plan{Path: protocol.PathDateTime, Ops: ops}, nil
}

func FromLocale(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.Locale
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	if want.Lang != "" && want.Lang != have.Lang {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpLocaleLang, Value: want.Lang})
	}
	if want.Keymap != "" && want.Keymap != have.Keymap {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpLocaleKeymap, Value: want.Keymap})
	}
	if want.Timezone != "" && want.Timezone != have.Timezone {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpDateTimeTimezone, Value: want.Timezone})
	}
	return protocol.Plan{Path: protocol.PathLocale, Ops: ops}, nil
}

func FromSession(desired, observed []byte) (protocol.Plan, error) {
	var want protocol.SessionModel
	if err := unmarshal(desired, observed, &want, &protocol.SessionModel{}); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	if want.Action == "lock" {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpSessionLock, Action: "lock"})
	}
	return protocol.Plan{Path: protocol.PathSession, Ops: ops}, nil
}

func FromPower(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.PowerModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	switch want.Action {
	case "suspend":
		ops = append(ops, protocol.PlanOp{Type: protocol.OpPowerSuspend, Action: "suspend"})
	case "hibernate":
		ops = append(ops, protocol.PlanOp{Type: protocol.OpPowerHibernate, Action: "hibernate"})
	}
	if jsonHas(desired, "brightness") && want.Brightness != have.Brightness && want.Brightness >= 0 {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpPowerBrightness, ID: want.Backlight, Scale: float64(want.Brightness)})
	}
	if want.Lid != "" && want.Lid != have.Lid {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpPowerLid, Value: want.Lid})
	}
	return protocol.Plan{Path: protocol.PathPower, Ops: ops}, nil
}
