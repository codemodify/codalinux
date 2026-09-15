// Command system-config is the CLI client. It talks to system-configd only.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	path := ""
	if len(args) > 1 {
		path = args[1]
	}
	var resp any
	var callErr error
	switch op {
	case "get":
		resp, callErr = c.Get(path)
	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: system-config set <path> <json>")
		}
		if !json.Valid([]byte(args[2])) {
			return fmt.Errorf("set data is not JSON")
		}
		resp, callErr = c.Set(path, json.RawMessage(args[2]))
	case "refresh":
		resp, callErr = c.Refresh(path)
	case "apply":
		resp, callErr = c.Apply(path)
	case "watch":
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
