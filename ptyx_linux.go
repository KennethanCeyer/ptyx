//go:build linux

package ptyx

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	unixOpen    = unix.Open
	unixSyscall = unix.Syscall
	unixClose   = unix.Close
)

func openPTY() (*os.File, *os.File, error) {
	masterFd, err := unixOpen("/dev/ptmx", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}

	var ptn uint32
	_, _, errno := unixSyscall(unix.SYS_IOCTL, uintptr(masterFd), unix.TIOCGPTN, uintptr(unsafe.Pointer(&ptn)))
	if errno != 0 {
		err = errno
		_ = unixClose(masterFd)
		return nil, nil, fmt.Errorf("ioctl(TIOCGPTN): %w", err)
	}
	slaveName := fmt.Sprintf("/dev/pts/%d", ptn)

	var p int
	_, _, errno = unixSyscall(unix.SYS_IOCTL, uintptr(masterFd), unix.TIOCSPTLCK, uintptr(unsafe.Pointer(&p)))
	if errno != 0 {
		err = errno
		_ = unixClose(masterFd)
		return nil, nil, fmt.Errorf("ioctl(TIOCSPTLCK): %w", err)
	}

	slaveFd, err := unixOpen(slaveName, unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		_ = unixClose(masterFd)
		return nil, nil, err
	}

	return os.NewFile(uintptr(masterFd), "pty-master"), os.NewFile(uintptr(slaveFd), "pty-slave"), nil
}
