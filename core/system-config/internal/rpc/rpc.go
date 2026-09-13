package rpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codemodify/codalinux/core/system-config/internal/peercred"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

const maxLine = 8 << 20

type Handler func(peerUID int, req protocol.Request) protocol.Response

type Server struct {
	Socket string
	Allow  func(uid int) bool
	Handle Handler

	ln   net.Listener
	once sync.Once
}

func (s *Server) Listen() error {
	if err := sockpath.EnsureDir(s.Socket); err != nil {
		return err
	}
	_ = os.Remove(s.Socket)
	ln, err := net.Listen("unix", s.Socket)
	if err != nil {
		return err
	}
	if err := os.Chmod(s.Socket, 0o600); err != nil {
		_ = ln.Close()
		return err
	}
	s.ln = ln
	return nil
}

func (s *Server) Serve() error {
	if s.ln == nil {
		if err := s.Listen(); err != nil {
			return err
		}
	}
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return err
		}
		go s.serveConn(c)
	}
}

func (s *Server) Close() error {
	var err error
	s.once.Do(func() {
		if s.ln != nil {
			err = s.ln.Close()
		}
		_ = os.Remove(s.Socket)
	})
	return err
}

func (s *Server) serveConn(c net.Conn) {
	defer c.Close()
	uid, err := peercred.UID(c)
	if err != nil {
		return
	}
	if s.Allow != nil && !s.Allow(uid) {
		_ = Write(c, protocol.Response{OK: false, Error: "unauthorized"})
		return
	}
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 0, 64*1024), maxLine)
	for sc.Scan() {
		req, err := protocol.DecodeRequest(sc.Bytes())
		if err != nil {
			_ = Write(c, protocol.Response{OK: false, Error: err.Error()})
			continue
		}
		resp := s.Handle(uid, req)
		resp.ID = req.ID
		if err := Write(c, resp); err != nil {
			return
		}
	}
}

func Write(w io.Writer, v any) error {
	b, err := protocol.Encode(v)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

type Client struct {
	Socket  string
	Timeout time.Duration

	mu sync.Mutex
	c  net.Conn
	id atomic.Uint64
	rd *bufio.Scanner
}

func Dial(socket string) (*Client, error) {
	c := &Client{Socket: socket, Timeout: 12 * time.Second}
	if err := c.ensure(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) ensure() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.c != nil {
		return nil
	}
	conn, err := net.DialTimeout("unix", c.Socket, c.Timeout)
	if err != nil {
		return fmt.Errorf("dial %s: %w", c.Socket, err)
	}
	c.c = conn
	c.rd = bufio.NewScanner(conn)
	c.rd.Buffer(make([]byte, 0, 64*1024), maxLine)
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.c == nil {
		return nil
	}
	err := c.c.Close()
	c.c = nil
	return err
}

func (c *Client) Call(req protocol.Request) (protocol.Response, error) {
	if req.ID == "" {
		req.ID = fmt.Sprintf("%d", c.id.Add(1))
	}
	if req.Version == 0 {
		req.Version = protocol.Version
	}
	if err := c.ensure(); err != nil {
		return protocol.Response{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.c.SetDeadline(time.Now().Add(c.Timeout))
	if err := Write(c.c, req); err != nil {
		return protocol.Response{}, err
	}
	if !c.rd.Scan() {
		if err := c.rd.Err(); err != nil {
			return protocol.Response{}, err
		}
		return protocol.Response{}, io.EOF
	}
	return protocol.DecodeResponse(c.rd.Bytes())
}

func Raw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
