package ptyx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

func Run(ctx context.Context, opts SpawnOpts) error {
	spawnCtx, spawnCancel := context.WithCancel(context.Background())
	defer spawnCancel()

	go func() { <-ctx.Done(); spawnCancel() }()

	s, err := Spawn(spawnCtx, opts)
	if err != nil {
		return err
	}
	defer s.Close()

	waitCh := make(chan error, 1)
	go func() { waitCh <- s.Wait() }()

	select {
	case <-ctx.Done():
		_ = s.Close()
		<-waitCh
		return ctx.Err()
	case err := <-waitCh:
		return err
	}
}

func RunInteractive(ctx context.Context, opts SpawnOpts) error {
	c, err := NewConsole()
	if err != nil {
		if !IsErrNotAConsole(err) {
			return fmt.Errorf("failed to create console: %w", err)
		}

		s, spawnErr := Spawn(ctx, opts)
		if spawnErr != nil {
			return fmt.Errorf("spawn failed: %w", spawnErr)
		}
		defer s.Close()

		inDone := make(chan struct{})
		outDone := make(chan struct{})

		go func() {
			if n, _ := io.Copy(s.PtyWriter(), os.Stdin); n > 0 {
				_ = s.CloseStdin()
			}
			close(inDone)
		}()
		go func() { _, _ = io.Copy(os.Stdout, s.PtyReader()); close(outDone) }()

		waitCh := make(chan error, 1)
		go func() { waitCh <- s.Wait() }()

		select {
		case <-ctx.Done():
			_ = s.Close()
			<-waitCh
			<-inDone
			<-outDone
			return ctx.Err()
		case err := <-waitCh:
			<-inDone
			<-outDone

			var exitErr *ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode == -1 { return nil }
			return err
		}
	}

	defer c.Close()
	c.EnableVT()

	if st, err := c.MakeRaw(); err == nil {
		defer c.Restore(st)
	}

	w, h := c.Size()
	opts.Cols, opts.Rows = w, h

	s, err := Spawn(ctx, opts)
	if err != nil {
		return fmt.Errorf("spawn failed: %w", err)
	}
	defer s.Close()

	m := NewMux()
	if err := m.Start(c, s); err != nil {
		return fmt.Errorf("mux start failed: %w", err)
	}
	defer m.Stop()

	if ch := c.OnResize(); ch != nil {
		go func(ch <-chan struct{}) {
			for {
				select {
				case _, ok := <-ch:
					if !ok {
						return
					}
					_ = s.Resize(c.Size())
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- s.Wait() }()

	select {
	case <-ctx.Done():
		_ = s.Close()
		<-waitCh
		return ctx.Err()
	case err := <-waitCh:
		return err
	}
}
