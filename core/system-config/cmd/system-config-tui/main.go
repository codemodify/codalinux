// Command system-config-tui is the terminal Settings client. Talks to D only.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	dump := flag.Bool("dump", false, "print every KnownPath (non-interactive)")
	path := flag.String("path", "", "start on this KnownPath")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: system-config-tui [-dump] [-path PATH]

Terminal Settings. Talks to system-configd only (never report or apply).
Every KnownPath has a page: refresh / get / edit / per-section Apply.

  -dump          print all paths (scripts / non-TTY)
  -path PATH     start on PATH (display, network, …)

TTY keys:
  ↑↓ j k     move          ←→ h l   sidebar ↔ fields
  Enter      edit/stage    Space    toggle
  r refresh  a apply       u revert section
  q quit     ? help
`)
	}
	flag.Parse()

	s, err := newSession(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "system-config-tui: %v\n(start system-configd first)\n", err)
		os.Exit(1)
	}
	defer s.close()

	if *dump || !isTTY(int(os.Stdin.Fd())) {
		if err := s.dump(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "system-config-tui: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := s.runUI(); err != nil {
		fmt.Fprintf(os.Stderr, "system-config-tui: %v\n", err)
		os.Exit(1)
	}
}
