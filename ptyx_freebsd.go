//go:build freebsd || dragonfly

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

	sname, err := os.Readlink(fmt.Sprintf("/dev/fd/%d", master.Fd()))
	if err != nil {
		_ = master.Close()
		return nil, nil, fmt.Errorf("readlink slave pty: %w", err)
	}

	slave, err := os.OpenFile(sname, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		_ = master.Close()
		return nil, nil, err
	}

	return master, slave, nil
}
