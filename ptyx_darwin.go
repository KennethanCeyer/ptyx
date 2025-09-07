//go:build darwin && !ios

package ptyx

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openPTY() (pty, tty *os.File, err error) {
	pty, err = os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			_ = pty.Close()
		}
	}()

	if err = ioctl(pty.Fd(), unix.TIOCPTYGRANT, 0); err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCPTYGRANT): %w", err)
	}

	if err = ioctl(pty.Fd(), unix.TIOCPTYUNLK, 0); err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCPTYUNLK): %w", err)
	}

	snameBuf := make([]byte, 128)
	if err = ioctl(pty.Fd(), unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&snameBuf[0]))); err != nil {
		return nil, nil, fmt.Errorf("ioctl(TIOCPTYGNAME): %w", err)
	}
	sname := string(snameBuf[:clen(snameBuf)])

	tty, err = os.OpenFile(sname, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}

	return pty, tty, nil
}
