package reportd

import (
	"os"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/reportprobe"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

type Server struct {
	Socket   string
	AllowUID int
	Probe    *reportprobe.Probe
	rpc      *rpc.Server
}

func New(socket string) *Server {
	if socket == "" {
		socket = sockpath.Report()
	}
	s := &Server{Socket: socket, AllowUID: sockpath.UID(), Probe: reportprobe.New()}
	s.rpc = &rpc.Server{
		Socket: socket,
		Allow: func(uid int) bool {
			return uid == s.AllowUID || uid == 0 || uid == os.Getuid()
		},
		Handle: s.handle,
	}
	return s
}

func (s *Server) Listen() error { return s.rpc.Listen() }
func (s *Server) Serve() error  { return s.rpc.Serve() }
func (s *Server) Close() error  { return s.rpc.Close() }

func (s *Server) handle(_ int, req protocol.Request) protocol.Response {
	if req.Op != protocol.OpScan {
		return protocol.Response{OK: false, Error: "report accepts scan only"}
	}
	path := protocol.NormalizePath(req.Path)
	if path == "" {
		path = protocol.PathDisplay
	}
	raw, err := s.Probe.Collect(path)
	if err != nil {
		return protocol.Response{OK: false, Error: err.Error()}
	}
	return protocol.Response{OK: true, Path: path, Observed: raw}
}

// PushOnce collects path and put-observed into D.
func PushOnce(dSocket, path string, probe *reportprobe.Probe) error {
	if probe == nil {
		probe = reportprobe.New()
	}
	if path == "" {
		path = protocol.PathDisplay
	}
	raw, err := probe.Collect(path)
	if err != nil {
		return err
	}
	c, err := rpc.Dial(dSocket)
	if err != nil {
		return err
	}
	defer c.Close()
	resp, err := c.Call(protocol.Request{Op: protocol.OpPutObserved, Path: path, Data: raw})
	if err != nil {
		return err
	}
	if !resp.OK {
		return errStr(resp.Error)
	}
	return nil
}

type errStr string

func (e errStr) Error() string { return string(e) }
