// Package daemon is system-configd: model + client API. Clients talk here only.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codemodify/codalinux/core/system-config/internal/model"
	"github.com/codemodify/codalinux/core/system-config/internal/plan"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

type Options struct {
	Socket       string
	ApplySocket  string
	ReportSocket string
	AllowUID     int
	// Optional in-process hooks (tests). Production dials ApplySocket / ReportSocket.
	Scan  func(path string) (json.RawMessage, error)
	Apply func(protocol.Plan) error
}

type Server struct {
	opt Options
	st  *model.Store
	rpc *rpc.Server
}

func New(opt Options) *Server {
	if opt.Socket == "" {
		opt.Socket = sockpath.Daemon()
	}
	if opt.ApplySocket == "" {
		opt.ApplySocket = sockpath.Apply()
	}
	if opt.ReportSocket == "" {
		opt.ReportSocket = sockpath.Report()
	}
	if opt.AllowUID == 0 {
		opt.AllowUID = sockpath.UID()
	}
	s := &Server{opt: opt, st: model.New()}
	s.rpc = &rpc.Server{
		Socket: opt.Socket,
		Allow: func(uid int) bool {
			return uid == opt.AllowUID || uid == 0 || uid == os.Getuid()
		},
		Handle: s.handle,
	}
	return s
}

func (s *Server) Listen() error  { return s.rpc.Listen() }
func (s *Server) Serve() error   { return s.rpc.Serve() }
func (s *Server) Close() error   { return s.rpc.Close() }
func (s *Server) Socket() string { return s.opt.Socket }

func (s *Server) handle(_ int, req protocol.Request) protocol.Response {
	switch req.Op {
	case protocol.OpGet:
		return s.get(req)
	case protocol.OpSet:
		return s.set(req)
	case protocol.OpWatch:
		resp := s.get(req)
		resp.Note = "watch stub: single snapshot (no event stream yet)"
		return resp
	case protocol.OpRefresh:
		return s.refresh(req)
	case protocol.OpApply:
		return s.apply(req)
	case protocol.OpPutObserved:
		return s.putObserved(req)
	default:
		return protocol.Response{OK: false, Error: "unknown op " + req.Op + " (get|set|watch|refresh|apply)"}
	}
}

func (s *Server) get(req protocol.Request) protocol.Response {
	path := protocol.NormalizePath(req.Path)
	if path == "" || path == protocol.PathSubmodels {
		return protocol.Response{OK: true, Path: protocol.PathSubmodels, Data: model.SubmodelsPayload()}
	}
	if !protocol.KnownPath(path) {
		return protocol.Response{OK: false, Error: "unknown submodel " + path}
	}
	d, o, st := s.st.Get(path)
	return protocol.Response{OK: true, Path: path, Desired: d, Observed: o, Status: &st}
}

func (s *Server) set(req protocol.Request) protocol.Response {
	path := protocol.NormalizePath(req.Path)
	if path != protocol.PathDisplay && path != protocol.PathLocale {
		return protocol.Response{OK: false, Error: "set not allowed on " + path}
	}
	if len(req.Data) == 0 {
		return protocol.Response{OK: false, Error: "set needs data"}
	}
	if err := s.st.SetDesired(path, req.Data); err != nil {
		return protocol.Response{OK: false, Error: err.Error()}
	}
	d, o, st := s.st.Get(path)
	return protocol.Response{OK: true, Path: path, Desired: d, Observed: o, Status: &st}
}

func (s *Server) putObserved(req protocol.Request) protocol.Response {
	path := protocol.NormalizePath(req.Path)
	if !protocol.KnownPath(path) || path == "" || path == protocol.PathSubmodels {
		return protocol.Response{OK: false, Error: "put-observed needs a submodel path"}
	}
	data := req.Data
	if len(data) == 0 {
		return protocol.Response{OK: false, Error: "put-observed needs data"}
	}
	if err := s.st.PutObserved(path, data); err != nil {
		return protocol.Response{OK: false, Error: err.Error()}
	}
	return protocol.Response{OK: true, Path: path}
}

func (s *Server) refresh(req protocol.Request) protocol.Response {
	path := protocol.NormalizePath(req.Path)
	if path == "" {
		path = protocol.PathDisplay
	}
	raw, err := s.scan(path)
	if err != nil {
		return protocol.Response{OK: false, Error: "refresh: " + err.Error()}
	}
	if err := s.st.PutObserved(path, raw); err != nil {
		return protocol.Response{OK: false, Error: err.Error()}
	}
	d, o, st := s.st.Get(path)
	return protocol.Response{OK: true, Path: path, Desired: d, Observed: o, Status: &st}
}

func (s *Server) scan(path string) (json.RawMessage, error) {
	if s.opt.Scan != nil {
		return s.opt.Scan(path)
	}
	c, err := rpc.Dial(s.opt.ReportSocket)
	if err != nil {
		return nil, fmt.Errorf("report not running (%s): %w", s.opt.ReportSocket, err)
	}
	defer c.Close()
	resp, err := c.Call(protocol.Request{Op: protocol.OpScan, Path: path})
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	if len(resp.Observed) > 0 {
		return resp.Observed, nil
	}
	return resp.Data, nil
}

func (s *Server) apply(req protocol.Request) protocol.Response {
	path := protocol.NormalizePath(req.Path)
	if path == "" {
		path = protocol.PathDisplay
	}
	if path != protocol.PathDisplay {
		return protocol.Response{OK: false, Error: "apply v1 allowlist is display only"}
	}
	d, o, _ := s.st.Get(path)
	pl, err := plan.FromDisplay(d, o)
	if err != nil {
		return protocol.Response{OK: false, Error: err.Error()}
	}
	pl.Path = path
	if len(pl.Ops) == 0 {
		s.st.ClearApplyError(path)
		_, o, st := s.st.Get(path)
		return protocol.Response{OK: true, Path: path, Observed: o, Status: &st, Note: "nothing to apply"}
	}
	if err := s.execPlan(pl); err != nil {
		s.st.SetApplyError(path, err.Error())
		_, _, st := s.st.Get(path)
		return protocol.Response{OK: false, Path: path, Error: err.Error(), Status: &st}
	}
	s.st.ClearApplyError(path)
	if raw, err := s.scan(path); err == nil {
		_ = s.st.PutObserved(path, raw)
	}
	_, o, st := s.st.Get(path)
	return protocol.Response{OK: true, Path: path, Observed: o, Status: &st}
}

func (s *Server) execPlan(pl protocol.Plan) error {
	if s.opt.Apply != nil {
		return s.opt.Apply(pl)
	}
	c, err := rpc.Dial(s.opt.ApplySocket)
	if err != nil {
		return fmt.Errorf("apply not running (%s): %w", s.opt.ApplySocket, err)
	}
	defer c.Close()
	resp, err := c.Call(protocol.Request{Op: protocol.OpExec, Plan: &pl})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}
