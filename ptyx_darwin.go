//go:build darwin

package ptyx

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	ioctl_TIOCPTYGNAME = 0x80407461
	ioctl_TIOCPTYUNLK = 0x20007462
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

	snameBuf := make([]byte, 128)
	err = ioctl(master.Fd(), ioctl_TIOCPTYGNAME, uintptr(unsafe.Pointer(&snameBuf[0])))
	if err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCPTYGNAME): %w", err)
	}
	sname := "/dev/" + string(snameBuf[:clen(snameBuf)])

	if err = os.Chown(sname, os.Getuid(), os.Getgid()); err != nil {
		return nil, nil, fmt.Errorf("grantpt: chown: %w", err)
	}
	if err = os.Chmod(sname, 0620); err != nil {
		return nil, nil, fmt.Errorf("grantpt: chmod: %w", err)
	}

	if err = ioctl(master.Fd(), ioctl_TIOCPTYUNLK, 0); err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCPTYUNLK): %w", err)
	}

	slave, err := os.OpenFile(sname, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}

	return master, slave, nil
}
