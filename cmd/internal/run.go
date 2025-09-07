package internal

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/KennethanCeyer/ptyx"
)

func RunInPty(ctx context.Context, opts ptyx.SpawnOpts) error {
	c, err := ptyx.NewConsole()
	if err != nil {
		if !ptyx.IsErrNotAConsole(err) {
			return fmt.Errorf("failed to create console: %w", err)
		}

		s, spawnErr := ptyx.Spawn(ctx, opts)
		if spawnErr != nil {
			return fmt.Errorf("spawn failed: %w", spawnErr)
		}
		defer s.Close()

		go io.Copy(s.PtyWriter(), os.Stdin)
		go io.Copy(os.Stdout, s.PtyReader())

		return s.Wait()
	}
	defer c.Close()
	c.EnableVT()

	st, err := c.MakeRaw()
	if err == nil {
		defer c.Restore(st)
	}

	w, h := c.Size()
	opts.Cols, opts.Rows = w, h

	s, err := ptyx.Spawn(ctx, opts)
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
