package ptyx

import (
	"io"
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
	closeStdinOnce sync.Once

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
		m.closeStdinOnce.Do(func() { _ = s.CloseStdin() })
	}()

	go func() {
		defer m.wg.Done()
		_, _ = io.Copy(c.Out(), s.PtyReader())

		m.closeStdinOnce.Do(func() { _ = s.CloseStdin() })
	}()
	return nil
}

func (m *mux) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == muxRunning {
		m.state = muxStopped
		if m.c != nil {
			if closer, ok := m.c.In().(io.Closer); ok {
				_ = closer.Close()
			}
		}
	}

	m.wg.Wait()
	return nil
}
