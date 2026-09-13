// Command system-configd is the unprivileged control plane.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/codemodify/codalinux/core/system-config/internal/daemon"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

func main() {
	sock := flag.String("socket", sockpath.Daemon(), "listen socket")
	apply := flag.String("apply-socket", sockpath.Apply(), "system-config-apply socket")
	report := flag.String("report-socket", sockpath.Report(), "system-config-report socket")
	flag.Parse()

	s := daemon.New(daemon.Options{Socket: *sock, ApplySocket: *apply, ReportSocket: *report})
	if err := s.Listen(); err != nil {
		log.Fatal(err)
	}
	log.Printf("system-configd listening on %s (pid %d uid %d)", *sock, os.Getpid(), os.Getuid())
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
