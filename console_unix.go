//go:build !windows

package ptyx

import (
	"os"
	"os/signal"
	"syscall"
)

func (c *console) EnableVT() {
}

func (c *console) initResizeWatcher() {
	c.win = &resizeWatcher{C: make(chan struct{}, 1), stop: make(chan struct{})}
	go func() {
		defer close(c.win.C)
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGWINCH)
		defer signal.Stop(sig)

		for {
			select {
			case <-sig:
				select {
				case c.win.C <- struct{}{}:
				default:
				}
			case <-c.win.stop:
				return
			}
		}
	}()
}
