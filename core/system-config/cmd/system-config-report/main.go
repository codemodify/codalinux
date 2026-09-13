// Command system-config-report collects observed state. It never applies config.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/reportd"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

func main() {
	sock := flag.String("socket", sockpath.Report(), "listen socket")
	dSock := flag.String("daemon-socket", sockpath.Daemon(), "system-configd socket (for --once/--watch push)")
	once := flag.Bool("once", false, "scan paths, push observed into D, exit")
	watch := flag.Bool("watch", false, "poll and push observed (udev netlink not wired yet)")
	interval := flag.Duration("interval", 3*time.Second, "watch poll interval")
	path := flag.String("path", "", "single path for --once (default: all starter paths)")
	flag.Parse()

	if *once || *watch {
		paths := protocol.StarterPaths
		if *path != "" {
			paths = []string{*path}
		}
		push := func() {
			for _, p := range paths {
				if err := reportd.PushOnce(*dSock, p, nil); err != nil {
					log.Printf("push %s: %v", p, err)
				}
			}
		}
		push()
		if !*watch {
			return
		}
		log.Printf("report watch stub: polling every %s (not udev netlink)", *interval)
		t := time.NewTicker(*interval)
		defer t.Stop()
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		for {
			select {
			case <-t.C:
				push()
			case <-ch:
				return
			}
		}
	}

	s := reportd.New(*sock)
	if err := s.Listen(); err != nil {
		log.Fatal(err)
	}
	log.Printf("system-config-report listening on %s", *sock)
	go func() {
		if err := s.Serve(); err != nil {
			log.Printf("serve: %v", err)
		}
	}()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	_ = s.Close()
}
