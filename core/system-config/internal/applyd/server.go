package applyd

import (
	"os"

	"github.com/codemodify/codalinux/core/system-config/internal/applyexec"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

type Server struct {
	Socket   string
	AllowUID int
	Runner   *applyexec.Runner
	rpc      *rpc.Server
}

func New(socket string) *Server {
	if socket == "" {
		socket = sockpath.Apply()
	}
	s := &Server{Socket: socket, AllowUID: sockpath.UID(), Runner: applyexec.New()}
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
	if req.Op != protocol.OpExec {
		return protocol.Response{OK: false, Error: "apply accepts exec only"}
	}
	if req.Plan == nil || len(req.Plan.Ops) == 0 {
		return protocol.Response{OK: false, Error: "exec needs a plan"}
	}
	if err := s.Runner.Exec(*req.Plan); err != nil {
		return protocol.Response{OK: false, Error: err.Error()}
	}
	return protocol.Response{OK: true}
}
