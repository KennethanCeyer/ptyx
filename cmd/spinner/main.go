package main

import (
	"fmt"
	"log"
	"time"

	"github.com/KennethanCeyer/ptyx"
)

func main() {
	c, err := ptyx.NewConsole()
	if err != nil {
		log.Fatalf("failed to create console: %v", err)
	}
	defer c.Close()
	c.EnableVT()

	frames := []rune{'⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏'}

	c.Out().Write([]byte(ptyx.CSI("?25l")))
	defer c.Out().Write([]byte(ptyx.CSI("?25h")))

	for i := 0; i < 60; i++ {
		pc := i * 100 / 59
		s := fmt.Sprintf("%c  Progress: %3d%%", frames[i%len(frames)], pc)
		c.Out().Write([]byte("\r" + ptyx.CSI("2K") + s))
		time.Sleep(80 * time.Millisecond)
	}
	c.Out().Write([]byte("\r" + ptyx.CSI("2K") + "Done.\n"))
}
