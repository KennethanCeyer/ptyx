//go:build darwin

package ptyx

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openPtyFallback() (*os.File, *os.File, error) {
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

func openPTY() (pty, tty *os.File, err error) {
	pty, err = os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return openPtyFallback()
		}
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			_ = pty.Close()
		}
	}()

	snameBuf := make([]byte, 128)
	err = ioctl(pty.Fd(), unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&snameBuf[0])))
	if err != nil {
		if errors.Is(err, unix.ENOTTY) {
			_ = pty.Close()
			return openPtyFallback()
		}
		return nil, nil, fmt.Errorf("ioctl(TIOCPTYGNAME): %w", err)
	}
	sname := string(snameBuf[:clen(snameBuf)])

	if err = ioctl(pty.Fd(), unix.TIOCPTYGRANT, 0); err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCPTYGRANT): %w", err)
	}

	if err = ioctl(pty.Fd(), unix.TIOCPTYUNLK, 0); err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCPTYUNLK): %w", err)
	}

	tty, err = os.OpenFile(sname, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}

	return pty, tty, nil
}
