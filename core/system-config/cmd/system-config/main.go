// Command system-config is the CLI client. It talks to system-configd only.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "system-config:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Printf(`Usage: system-config <get|set|refresh|apply|watch> [path] [json]

Talks to system-configd only (never report or apply).

Paths:
  %s

Set/apply: display network audio bluetooth input datetime locale session power
Observe-only: devices.summary devices.pci devices.usb hardware.dmi

  system-config get network
  system-config set network '{"wifi":{"device":"wlan0","connect":"SSID","psk":"secret"}}'
  system-config apply network
  system-config set audio '{"volume":0.5,"mute":false}'
  system-config set bluetooth '{"powered":true,"connect":["AA:BB:CC:DD:EE:FF"]}'
  system-config set input '{"kb_layout":"us","natural_scroll":true}'
  system-config set datetime '{"timezone":"America/Denver","ntp":true}'
  system-config set locale '{"lang":"en_US.UTF-8","keymap":"us"}'
  system-config refresh display
  system-config watch display
  system-config watch display --follow

watch (default): one snapshot; the connection stays request/response.
watch --follow: snapshot plus further JSON lines when report pushes observed
(udev/netlink + slow poll) or another client sets desired. Identical
re-pushes are not emitted. Empty path watches every KnownPath.

Socket: $CODA_SYSTEM_CONFIG_SOCKET or $XDG_RUNTIME_DIR/coda/system-configd.sock
`, strings.Join(protocol.KnownPaths, " "))
		return nil
	}
	sock := sockpath.Daemon()
	c, err := client.Dial(sock)
	if err != nil {
		return err
	}
	defer c.Close()

	op := args[0]
	path, follow := parsePathFollow(args[1:])
	var resp any
	var callErr error
	switch op {
	case "get":
		resp, callErr = c.Get(path)
	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: system-config set <path> <json>")
		}
		payload := args[2]
		if !json.Valid([]byte(payload)) {
			return fmt.Errorf("set data is not JSON")
		}
		resp, callErr = c.Set(args[1], json.RawMessage(payload))
	case "refresh":
		resp, callErr = c.Refresh(path)
	case "apply":
		resp, callErr = c.Apply(path)
	case "watch":
		if follow {
			return watchFollow(c, path)
		}
		resp, callErr = c.Watch(path)
	default:
		return fmt.Errorf("unknown command %s (try help)", op)
	}
	if callErr != nil {
		return callErr
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func parsePathFollow(args []string) (path string, follow bool) {
	for _, a := range args {
		if a == "-f" || a == "--follow" || a == "follow" {
			follow = true
			continue
		}
		if path == "" {
			path = a
		}
	}
	return path, follow
}

func watchFollow(c *client.Client, path string) error {
	resp, err := c.WatchFollow(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(resp); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	for {
		ev, err := c.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if err := enc.Encode(ev); err != nil {
			return err
		}
	}
}
