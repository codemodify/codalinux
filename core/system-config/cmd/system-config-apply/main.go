// Command system-config-apply executes typed plans from D only.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/codemodify/codalinux/core/system-config/internal/applyd"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

func main() {
	sock := flag.String("socket", sockpath.Apply(), "listen socket")
	flag.Parse()

	s := applyd.New(*sock)
	if err := s.Listen(); err != nil {
		log.Fatal(err)
	}
	log.Printf("system-config-apply listening on %s (euid %d seat %d); allowlist (%d): %s",
		*sock, os.Getuid(), sockpath.UID(), len(protocol.ApplyOps), protocol.ApplyOpsLog())
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
