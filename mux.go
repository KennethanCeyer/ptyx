package ptyx

import (
	"io"
	"os"
	"runtime"
	"sync"
)

const (
	muxInit = iota
	muxRunning
	muxStopped
)

type mux struct {
	cancel func()
	wg     sync.WaitGroup

	mu    sync.Mutex
	state int

	c Console
	s Session
}

func NewMux() Mux { return &mux{} }

func (m *mux) Start(c Console, s Session) error {
	m.mu.Lock()
	if m.state != muxInit {
		m.mu.Unlock()
		return ErrMuxAlreadyStarted
	}
	m.state = muxRunning
	m.c, m.s = c, s
	m.mu.Unlock()

	m.cancel = func() {}

	m.wg.Add(2)

	go func() {
		defer m.wg.Done()
		_, _ = io.Copy(s.PtyWriter(), c.In())
	}()

	go func() {
		defer m.wg.Done()
		_, _ = io.Copy(c.Out(), s.PtyReader())
		if runtime.GOOS != "windows" {
			if closer, ok := c.In().(*os.File); ok {
				_ = closer.Close()
			}
		}
	}()
	return nil
}

func (m *mux) Stop() error {
	m.mu.Lock()
	if m.state == muxRunning {
		m.state = muxStopped
		if m.cancel != nil {
			m.cancel()
		}
		if m.s != nil {
			_ = m.s.CloseStdin()
		}
		if runtime.GOOS != "windows" && m.c != nil {
			if f, ok := m.c.In().(*os.File); ok {
				_ = f.Close()
			}
		}
	}
	m.mu.Unlock()

	m.wg.Wait()
	return nil
}
