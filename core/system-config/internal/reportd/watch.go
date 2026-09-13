package reportd

import (
	"context"
	"log"
	"time"

	"github.com/codemodify/codalinux/core/system-config/internal/reportprobe"
	"github.com/codemodify/codalinux/core/system-config/internal/udevwatch"
)

// Watch pushes observed into D on udev netlink events, with a slow poll
// fallback for L2 stacks (Hyprland, PipeWire, timedatectl) that do not
// emit kobject uevents.
func Watch(ctx context.Context, dSock string, paths []string, poll time.Duration, probe *reportprobe.Probe) {
	if poll <= 0 {
		poll = 30 * time.Second
	}
	want := map[string]bool{}
	for _, p := range paths {
		want[p] = true
	}
	push := func(list []string) {
		for _, p := range list {
			if !want[p] {
				continue
			}
			if err := PushOnce(dSock, p, probe); err != nil {
				log.Printf("push %s: %v", p, err)
			}
		}
	}
	push(paths)

	ev, err := udevwatch.Listen(ctx)
	if err != nil {
		log.Printf("udev netlink unavailable (%v); polling every %s", err, poll)
	} else {
		log.Printf("udev netlink watch on; L2 poll every %s", poll)
		go func() {
			pending := map[string]bool{}
			timer := time.NewTimer(250 * time.Millisecond)
			if !timer.Stop() {
				<-timer.C
			}
			flush := func() {
				var list []string
				for p := range pending {
					list = append(list, p)
				}
				pending = map[string]bool{}
				if len(list) > 0 {
					push(list)
				}
			}
			for {
				select {
				case <-ctx.Done():
					return
				case e, ok := <-ev:
					if !ok {
						return
					}
					for _, p := range udevwatch.PathsFor(e) {
						if want[p] {
							pending[p] = true
						}
					}
					_ = timer.Reset(250 * time.Millisecond)
				case <-timer.C:
					flush()
				}
			}
		}()
	}

	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			push(paths)
		}
	}
}
