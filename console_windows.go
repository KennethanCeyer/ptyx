//go:build windows

package ptyx

import (
	"os"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

type rawState struct{ st *term.State; fd int }
type winWatcher struct{ C chan struct{}; stop chan struct{} }

func NewConsole() (Console, error) {
	c := &console{ in: os.Stdin, out: os.Stdout, err: os.Stderr }
	c.outTTY = term.IsTerminal(int(c.out.Fd()))
	c.errTTY = term.IsTerminal(int(c.err.Fd()))
	c.win = &winWatcher{C: make(chan struct{}, 1), stop: make(chan struct{})}
	go func() {
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		defer close(c.win.C)
		for {
			select {
			case <-t.C:
				select { case c.win.C <- struct{}{}: default: }
			case <-c.win.stop:
				return
			}
		}
	}()
	c.EnableVT()
	return c, nil
}

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

func (c *console) Size() (int, int) {
	w, h, err := term.GetSize(int(c.out.Fd()))
	if err != nil { return 0, 0 }
	return w, h
}

func (c *console) MakeRaw() (RawState, error) {
	fd := int(c.in.Fd())
	st, err := term.MakeRaw(fd)
	if err != nil { return nil, err }
	r := &rawState{st: st, fd: fd}
	c.raw = r
	return r, nil
}

func (c *console) Restore(s RawState) error {
	r, ok := s.(*rawState)
	if !ok || r == nil || r.st == nil { return nil }
	err := term.Restore(r.fd, r.st)
	if err == nil { c.raw = nil }
	return err
}

func (c *console) Close() error {
	c.closeOnce.Do(func() {
		if c.win != nil && c.win.stop != nil {
			close(c.win.stop)
		}
	})
	return nil
}
