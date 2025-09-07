package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	c, err := ptyx.NewConsole()
	if err != nil {
		if errors.Is(err, ptyx.ErrNotAConsole) {
			return
		}
		log.Fatalf("failed to create console: %v", err)
	}
	defer c.Close()
	c.EnableVT()

	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	runSpinner(appCtx, c)
}

func runSpinner(ctx context.Context, c ptyx.Console) {
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	go func() {
		_, _ = io.Copy(io.Discard, c.In())
		stop()
	}()

	frames := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

	c.Out().Write([]byte(ptyx.CSI("?25l")))
	defer c.Out().Write([]byte(ptyx.CSI("?25h")))

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	i := 0
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			if i >= 20 {
				break loop
			}
			s := fmt.Sprintf("%c  Working... %3d%%", frames[i%len(frames)], i*100/19)
			c.Out().Write([]byte("\r" + ptyx.CSI("2K") + s))
			i++
		}
	}

	c.Out().Write([]byte("\r" + ptyx.CSI("2K")))
	if ctx.Err() != nil {
		c.Out().Write([]byte("Spinner stopped.\n"))
	} else {
		c.Out().Write([]byte("Done.\n"))
	}
}
