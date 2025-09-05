package ptyx

import (
	"context"
	"io"
	"sync"
)

type mux struct {
	cancel func()
	wg     sync.WaitGroup
	mu     sync.Mutex
	state  int
}

func NewMux() Mux { return &mux{} }

func (m *mux) Start(c Console, s Session) error {
	m.mu.Lock()
	if m.state != 0 {
		m.mu.Unlock()
		return ErrMuxAlreadyStarted
	}
	m.state = 1
	m.mu.Unlock()

	_, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	m.wg.Add(2)

	go func() {
		defer m.wg.Done()
		io.Copy(s.PtyWriter(), c.In())
	}()

	go func() {
		defer m.wg.Done()
		io.Copy(c.Out(), s.PtyReader())
		if closer, ok := c.In().(io.Closer); ok {
			_ = closer.Close()
		}
		m.cancel()
	}()
	return nil
}

func (m *mux) Stop() error {
	m.mu.Lock()
	if m.state == 1 {
		m.state = 2
		if m.cancel != nil {
			m.cancel()
		}
	}
	m.mu.Unlock()

	m.wg.Wait()
	return nil
}
