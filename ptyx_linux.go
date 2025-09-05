//go:build linux && !android

package ptyx

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openPTY() (*os.File, *os.File, error) {
	masterFd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}

	ptmxN, err := unix.IoctlGetInt(masterFd, unix.TIOCGPTN)
	if err != nil {
		_ = unix.Close(masterFd)
		return nil, nil, fmt.Errorf("ioctl(TIOCGPTN): %w", err)
	}

	val := 0
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(masterFd), unix.TIOCSPTLCK, uintptr(unsafe.Pointer(&val)))
	if errno != 0 {
		err = errno
		_ = unix.Close(masterFd)
		return nil, nil, fmt.Errorf("ioctl(TIOCSPTLCK): %w", err)
	}

	slavePath := fmt.Sprintf("/dev/pts/%d", ptmxN)
	slaveFd, err := unix.Open(slavePath, unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		_ = unix.Close(masterFd)
		return nil, nil, err
	}

	return os.NewFile(uintptr(masterFd), "pty-master"), os.NewFile(uintptr(slaveFd), "pty-slave"), nil
}
