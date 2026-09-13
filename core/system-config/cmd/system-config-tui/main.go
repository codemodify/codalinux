// Command system-config-tui is a minimal text client. Talks to D only.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

func main() {
	sock := sockpath.Daemon()
	c, err := client.Dial(sock)
	if err != nil {
		fmt.Fprintf(os.Stderr, "system-config-tui: %v\n(start system-configd first)\n", err)
		os.Exit(1)
	}
	defer c.Close()

	fmt.Println("Coda system-config (TUI stub) — D only")
	fmt.Println("Submodels:")
	for _, p := range protocol.StarterPaths {
		resp, err := c.Get(p)
		if err != nil {
			fmt.Printf("  %s  error: %v\n", p, err)
			continue
		}
		if !resp.OK {
			fmt.Printf("  %s  %s\n", p, resp.Error)
			continue
		}
		st := ""
		if resp.Status != nil {
			st = fmt.Sprintf(" present=%v configured=%v changed=%v", resp.Status.Present, resp.Status.Configured, resp.Status.Changed)
		}
		fmt.Printf("  %s%s\n", p, st)
		if len(resp.Observed) > 0 {
			var pretty any
			_ = json.Unmarshal(resp.Observed, &pretty)
			b, _ := json.MarshalIndent(pretty, "    ", "  ")
			fmt.Printf("    observed %s\n", b)
		}
	}
	fmt.Println("Full TUI pages are not implemented; use system-config CLI or system-config-gui.")
}
