// Package client talks to system-configd only.
package client

import (
	"encoding/json"
	"fmt"
	"time"

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

// WatchFollow sends watch with follow=true and returns the first snapshot.
// Call Next to read later observed/desired events on this connection.
func (c *Client) WatchFollow(path string) (protocol.Response, error) {
	data, _ := json.Marshal(protocol.WatchOpts{Follow: true})
	return c.Call(protocol.Request{Op: protocol.OpWatch, Path: path, Data: data})
}

// WatchFollowTimeout is WatchFollow with a server-side stream deadline.
func (c *Client) WatchFollowTimeout(path string, timeout time.Duration) (protocol.Response, error) {
	ms := int(timeout / time.Millisecond)
	if ms < 1 {
		ms = 1
	}
	data, _ := json.Marshal(protocol.WatchOpts{Follow: true, TimeoutMS: ms})
	return c.Call(protocol.Request{Op: protocol.OpWatch, Path: path, Data: data})
}
