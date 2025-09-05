//go:build netbsd

package ptyx

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	ioctl_TIOCGRANTPT = 0x20007406
	ioctl_TIOCPTSNAME = 0x80287409
)

type ptmget struct {
	Cfd int32
	Sfd int32
	Cn  [16]byte
	Sn  [16]byte
}

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

	if err = ioctl(master.Fd(), ioctl_TIOCGRANTPT, 0); err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCGRANTPT): %w", err)
	}

	var pm ptmget
	if err = ioctl(master.Fd(), ioctl_TIOCPTSNAME, uintptr(unsafe.Pointer(&pm))); err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCPTSNAME): %w", err)
	}
	sname := string(pm.Sn[:clen(pm.Sn[:])])

	slave, err := os.OpenFile(sname, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}

	return master, slave, nil
}
