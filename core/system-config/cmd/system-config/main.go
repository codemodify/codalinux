// Command system-config is the CLI client. It talks to system-configd only.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
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
		fmt.Print(`Usage: system-config <get|set|refresh|apply|watch> [path] [json]

Talks to system-configd only (never report or apply).

  system-config get display
  system-config get devices.summary
  system-config get devices.pci
  system-config get locale
  system-config set display '{"outputs":[{"name":"Virtual-1","scale":2}]}'
  system-config refresh display
  system-config apply display
  system-config watch display

Socket: $CODA_SYSTEM_CONFIG_SOCKET or $XDG_RUNTIME_DIR/coda/system-configd.sock
`)
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
