package plan

import (
	"encoding/json"
	"fmt"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

// Build diffs desired vs observed and emits allowlisted ops for path.
func Build(path string, desired, observed json.RawMessage) (protocol.Plan, error) {
	path = protocol.NormalizePath(path)
	var (
		p   protocol.Plan
		err error
	)
	switch path {
	case protocol.PathDisplay:
		p, err = FromDisplay(desired, observed)
	case protocol.PathNetwork:
		p, err = FromNetwork(desired, observed)
	case protocol.PathAudio:
		p, err = FromAudio(desired, observed)
	case protocol.PathBluetooth:
		p, err = FromBluetooth(desired, observed)
	case protocol.PathInput:
		p, err = FromInput(desired, observed)
	case protocol.PathDateTime:
		p, err = FromDateTime(desired, observed)
	case protocol.PathLocale:
		p, err = FromLocale(desired, observed)
	case protocol.PathSession:
		p, err = FromSession(desired, observed)
	case protocol.PathPower:
		p, err = FromPower(desired, observed)
	default:
		return protocol.Plan{}, fmt.Errorf("no apply plan for %s", path)
	}
	if err != nil {
		return p, err
	}
	p.Path = path
	return p, nil
}

func unmarshal[T any](desired, observed json.RawMessage, want, have *T) error {
	if len(desired) > 0 {
		if err := json.Unmarshal(desired, want); err != nil {
			return fmt.Errorf("desired: %w", err)
		}
	}
	if len(observed) > 0 {
		if err := json.Unmarshal(observed, have); err != nil {
			return fmt.Errorf("observed: %w", err)
		}
	}
	return nil
}

func jsonHas(raw json.RawMessage, key string) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

func boolPtr(v bool) *bool { return &v }

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
