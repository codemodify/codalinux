// Package client talks to system-configd only.
package client

import (
	"encoding/json"
	"fmt"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

type Client struct {
	*rpc.Client
}

func Dial(socket string) (*Client, error) {
	if socket == "" {
		socket = sockpath.Daemon()
	}
	c, err := rpc.Dial(socket)
	if err != nil {
		return nil, fmt.Errorf("system-configd: %w", err)
	}
	return &Client{Client: c}, nil
}

func (c *Client) Get(path string) (protocol.Response, error) {
	return c.Call(protocol.Request{Op: protocol.OpGet, Path: path})
}

func (c *Client) Set(path string, data json.RawMessage) (protocol.Response, error) {
	return c.Call(protocol.Request{Op: protocol.OpSet, Path: path, Data: data})
}

func (c *Client) Refresh(path string) (protocol.Response, error) {
	return c.Call(protocol.Request{Op: protocol.OpRefresh, Path: path})
}

func (c *Client) Apply(path string) (protocol.Response, error) {
	return c.Call(protocol.Request{Op: protocol.OpApply, Path: path})
}

func (c *Client) Watch(path string) (protocol.Response, error) {
	return c.Call(protocol.Request{Op: protocol.OpWatch, Path: path})
}
