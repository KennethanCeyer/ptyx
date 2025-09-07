package ptyx

import "context"

func Run(ctx context.Context, opts SpawnOpts) error {
	s, err := Spawn(ctx, opts)
	if err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- s.Wait()
		close(waitCh)
	}()

	select {
	case <-ctx.Done():
		_ = s.Close()
		<-waitCh
		return ctx.Err()
	case err := <-waitCh:
		_ = s.Close()
		return err
	}
}
