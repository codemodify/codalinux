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
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

func main() {
	sock := flag.String("socket", sockpath.Report(), "listen socket")
	dSock := flag.String("daemon-socket", sockpath.Daemon(), "system-configd socket (for --once/--watch push)")
	once := flag.Bool("once", false, "scan paths, push observed into D, exit")
	watch := flag.Bool("watch", false, "push observed on udev netlink + slow L2 poll")
	interval := flag.Duration("interval", 30*time.Second, "L2 poll interval while watching (udev is event-driven)")
	path := flag.String("path", "", "single path for --once/--watch (default: all starter paths)")
	flag.Parse()

	if *once || *watch {
		paths := protocol.StarterPaths
		if *path != "" {
			paths = []string{*path}
		}
		if *once && !*watch {
			for _, p := range paths {
				if err := reportd.PushOnce(*dSock, p, nil); err != nil {
					log.Printf("push %s: %v", p, err)
				}
			}
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-ch
			cancel()
		}()
		reportd.Watch(ctx, *dSock, paths, *interval, nil)
		return
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
