package model

import (
	"encoding/json"
	"sync"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

// Store holds desired + observed per submodel path.
type Store struct {
	mu       sync.Mutex
	desired  map[string]json.RawMessage
	observed map[string]json.RawMessage
	status   map[string]protocol.Status
}

func New() *Store {
	return &Store{
		desired:  map[string]json.RawMessage{},
		observed: map[string]json.RawMessage{},
		status:   map[string]protocol.Status{},
	}
}

func (s *Store) Get(path string) (desired, observed json.RawMessage, st protocol.Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path = protocol.NormalizePath(path)
	desired = append(json.RawMessage(nil), s.desired[path]...)
	observed = append(json.RawMessage(nil), s.observed[path]...)
	st = s.status[path]
	if observed != nil {
		st.Present = presentFor(path, observed)
	}
	if desired != nil {
		st.Configured = true
	}
	st.Changed = changed(desired, observed)
	return desired, observed, st
}

func (s *Store) SetDesired(path string, data json.RawMessage) error {
	if !json.Valid(data) {
		return errInvalidJSON
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path = protocol.NormalizePath(path)
	s.desired[path] = append(json.RawMessage(nil), data...)
	st := s.status[path]
	st.Configured = true
	st.Changed = changed(s.desired[path], s.observed[path])
	st.ApplyError = ""
	s.status[path] = st
	return nil
}

func (s *Store) PutObserved(path string, data json.RawMessage) error {
	if !json.Valid(data) {
		return errInvalidJSON
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path = protocol.NormalizePath(path)
	s.observed[path] = append(json.RawMessage(nil), data...)
	st := s.status[path]
	st.Present = presentFor(path, data)
	st.Changed = changed(s.desired[path], s.observed[path])
	s.status[path] = st
	return nil
}

func presentFor(path string, raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	switch path {
	case protocol.PathBluetooth:
		var m protocol.BluetoothModel
		if json.Unmarshal(raw, &m) != nil {
			return false
		}
		return m.Adapter != "" || len(m.Devices) > 0
	case protocol.PathPrinters:
		var m protocol.PrintersModel
		if json.Unmarshal(raw, &m) != nil {
			return false
		}
		return len(m.Printers) > 0
	case protocol.PathStorage:
		var m protocol.StorageModel
		if json.Unmarshal(raw, &m) != nil {
			return false
		}
		return len(m.Block) > 0
	default:
		return true
	}
}

func (s *Store) SetApplyError(path, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path = protocol.NormalizePath(path)
	st := s.status[path]
	st.ApplyError = msg
	s.status[path] = st
}

func (s *Store) ClearApplyError(path string) {
	s.SetApplyError(path, "")
}

func changed(desired, observed json.RawMessage) bool {
	if len(desired) == 0 || len(observed) == 0 {
		return len(desired) > 0
	}
	return string(desired) != string(observed)
}

var errInvalidJSON = errStr("invalid JSON")

type errStr string

func (e errStr) Error() string { return string(e) }

func SubmodelsPayload() json.RawMessage {
	return rpc.Raw(map[string]any{"paths": protocol.StarterPaths})
}
