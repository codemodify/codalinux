// Command system-config-report collects observed state. It never applies config.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/reportd"
	"github.com/codemodify/codalinux/core/system-config/internal/rootguard"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

func main() {
	rootguard.RefuseRoot("system-config-report")
	sock := flag.String("socket", sockpath.Report(), "listen socket")
	dSock := flag.String("daemon-socket", sockpath.Daemon(), "system-configd socket (for --once/--watch push)")
	once := flag.Bool("once", false, "scan paths, push observed into D, exit")
	watchOnly := flag.Bool("watch", false, "udev watch only (no scan socket); default server also watches")
	noWatch := flag.Bool("no-watch", false, "RPC scan only; do not start udev netlink push")
	interval := flag.Duration("interval", 30*time.Second, "L2 poll interval while watching (udev is event-driven)")
	path := flag.String("path", "", "single path for --once/--watch (default: all starter paths)")
	flag.Parse()

	paths := protocol.StarterPaths
	if *path != "" {
		paths = []string{*path}
	}

	if *once {
		for _, p := range paths {
			if err := reportd.PushOnce(*dSock, p, nil); err != nil {
				log.Printf("push %s: %v", p, err)
			}
		}
		if !*watchOnly {
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()

	startWatch := func() {
		if *noWatch {
			return
		}
		go reportd.Watch(ctx, *dSock, paths, *interval, nil)
	}

	if *watchOnly {
		startWatch()
		<-ctx.Done()
		return
	}

	s := reportd.New(*sock)
	if err := s.Listen(); err != nil {
		log.Fatal(err)
	}
	log.Printf("system-config-report listening on %s", *sock)
	startWatch()
	go func() {
		if err := s.Serve(); err != nil {
			log.Printf("serve: %v", err)
		}
	}()
	<-ctx.Done()
	_ = s.Close()
}
