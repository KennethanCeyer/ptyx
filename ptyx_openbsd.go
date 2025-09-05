//go:build openbsd

package ptyx

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func openPTY() (*os.File, *os.File, error) {
	for i := 0; i < 256; i++ {
		masterPath := fmt.Sprintf("/dev/pty%c%x", 'p'+i/16, i%16)
		master, err := os.OpenFile(masterPath, os.O_RDWR, 0)
		if err != nil {
			continue
		}

		slavePath := fmt.Sprintf("/dev/tty%c%x", 'p'+i/16, i%16)
		slave, err := os.OpenFile(slavePath, os.O_RDWR|unix.O_NOCTTY, 0)
		if err != nil {
			_ = master.Close()
			continue
		}
		return master, slave, nil
	}
	return nil, nil, fmt.Errorf("out of PTY devices")
}
