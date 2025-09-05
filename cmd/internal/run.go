package internal

import (
	"context"
	"fmt"

	"github.com/KennethanCeyer/ptyx"
)

func RunInPty(ctx context.Context, opts ptyx.SpawnOpts) error {
	c, err := ptyx.NewConsole()
	if err != nil {
		return fmt.Errorf("failed to create console: %w", err)
	}
	defer c.Close()
	c.EnableVT()

	st, err := c.MakeRaw()
	if err == nil {
		defer c.Restore(st)
	}

	w, h := c.Size()
	opts.Cols, opts.Rows = w, h

	s, err := ptyx.Spawn(opts)
	if err != nil {
		return fmt.Errorf("spawn failed: %w", err)
	}
	defer s.Close()

	m := ptyx.NewMux()
	if err := m.Start(c, s); err != nil {
		return fmt.Errorf("mux start failed: %w", err)
	}
	defer m.Stop()

	go func() {
		for range c.OnResize() {
			_ = s.Resize(c.Size())
		}
	}()

	return s.Wait()
}
