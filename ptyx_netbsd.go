//go:build netbsd || dragonfly

package ptyx

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func openPTY() (*os.File, *os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			_ = master.Close()
		}
	}()

	sname, err := unix.Ptsname(int(master.Fd()))
	if err != nil {
		return nil, nil, fmt.Errorf("ptsname: %w", err)
	}
	slave, err := os.OpenFile(sname, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}

	return master, slave, nil
}
