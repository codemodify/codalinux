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

// Starter submodels (architecture.md). Clients query these, not the whole tree.
const (
	PathDisplay        = "display"
	PathDevicesSummary = "devices.summary"
	PathDevicesPCI     = "devices.pci"
	PathLocale         = "locale"
	PathSubmodels      = "submodels"
)

var StarterPaths = []string{
	PathDisplay,
	PathDevicesSummary,
	PathDevicesPCI,
	PathLocale,
}

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

type PlanOp struct {
	Type   string  `json:"type"`
	Output string  `json:"output,omitempty"`
	Scale  float64 `json:"scale,omitempty"`
	Mode   string  `json:"mode,omitempty"`
}

const (
	OpDisplayScale = "display.scale"
	OpDisplayMode  = "display.mode"
)

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
	for _, k := range StarterPaths {
		if p == k {
			return true
		}
	}
	return false
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
