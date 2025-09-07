//go:build windows

package ptyx

import (
	"time"

	"golang.org/x/sys/windows"
)

func (c *console) EnableVT() {
	const ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	const DISABLE_NEWLINE_AUTO_RETURN = 0x0008
	h := windows.Handle(c.out.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) == nil {
		mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING | DISABLE_NEWLINE_AUTO_RETURN
		_ = windows.SetConsoleMode(h, mode)
	}
	h2 := windows.Handle(c.err.Fd())
	if windows.GetConsoleMode(h2, &mode) == nil {
		mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING | DISABLE_NEWLINE_AUTO_RETURN
		_ = windows.SetConsoleMode(h2, mode)
	}
}

func (c *console) initResizeWatcher() {
	c.win = &resizeWatcher{C: make(chan struct{}, 1), stop: make(chan struct{}), ready: make(chan struct{})}
	go func() {
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		defer close(c.win.C)
		close(c.win.ready)
		for {
			select {
			case <-t.C:
				select { case c.win.C <- struct{}{}: default: }
			case <-c.win.stop: return
			}
		}
	}()
}
