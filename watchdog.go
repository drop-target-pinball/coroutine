package coroutine

import (
	"log"
	"os"
	"runtime/pprof"
	"time"
)

type Watchdog struct {
	reset chan struct{}
	done  chan struct{}
}

func NewWatchdog(timeout time.Duration) *Watchdog {
	w := &Watchdog{
		reset: make(chan struct{}, 1),
		done:  make(chan struct{}, 1),
	}
	go func() {
		for {
			select {
			case <-w.reset:
			case <-w.done:
				return
			case <-time.After(timeout):
				println("watchdog timer expired")
				profile := pprof.Lookup("goroutine")
				profile.WriteTo(os.Stdout, 1)
				log.Panicf("watchdog panic")
			}
		}
	}()
	return w
}

func (w *Watchdog) Reset() {
	w.reset <- struct{}{}
}

func (w *Watchdog) Stop() {
	w.done <- struct{}{}
}
