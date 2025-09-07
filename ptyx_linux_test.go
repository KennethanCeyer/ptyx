//go:build linux

package ptyx

import (
	"errors"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestOpenPTY_ErrorPaths(t *testing.T) {
	t.Run("OpenPtmxError", func(t *testing.T) {
		originalUnixOpen := unixOpen
		unixOpen = func(path string, mode int, perm uint32) (int, error) {
			return -1, errors.New("mock open error")
		}
		t.Cleanup(func() { unixOpen = originalUnixOpen })

		_, _, err := openPTY()
		if err == nil {
			t.Fatal("openPTY should have failed but did not")
		}
		if !strings.Contains(err.Error(), "mock open error") {
			t.Errorf("Expected 'mock open error', got %v", err)
		}
	})

	t.Run("IoctlTiocgptnError", func(t *testing.T) {
		originalUnixSyscall := unixSyscall
		unixSyscall = func(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno) {
			if a2 == unix.TIOCGPTN {
				return 0, 0, syscall.EIO
			}
			return originalUnixSyscall(trap, a1, a2, a3)
		}
		t.Cleanup(func() { unixSyscall = originalUnixSyscall })

		_, _, err := openPTY()
		if err == nil {
			t.Fatal("openPTY should have failed but did not")
		}
		if !strings.Contains(err.Error(), "ioctl(TIOCGPTN)") {
			t.Errorf("Expected 'ioctl(TIOCGPTN)' error, got %v", err)
		}
	})

	t.Run("IoctlTiocsptlckError", func(t *testing.T) {
		originalUnixSyscall := unixSyscall
		unixSyscall = func(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno) {
			if a2 == unix.TIOCSPTLCK {
				return 0, 0, syscall.EIO
			}
			return originalUnixSyscall(trap, a1, a2, a3)
		}
		t.Cleanup(func() { unixSyscall = originalUnixSyscall })

		_, _, err := openPTY()
		if err == nil {
			t.Fatal("openPTY should have failed but did not")
		}
		if !strings.Contains(err.Error(), "ioctl(TIOCSPTLCK)") {
			t.Errorf("Expected 'ioctl(TIOCSPTLCK)' error, got %v", err)
		}
	})

	t.Run("OpenSlaveError", func(t *testing.T) {
		originalUnixOpen := unixOpen
		unixOpen = func(path string, mode int, perm uint32) (int, error) {
			if strings.HasPrefix(path, "/dev/pts/") {
				return -1, errors.New("mock open slave error")
			}
			return originalUnixOpen(path, mode, perm)
		}
		t.Cleanup(func() { unixOpen = originalUnixOpen })

		_, _, err := openPTY()
		if err == nil {
			t.Fatal("openPTY should have failed but did not")
		}
		if !strings.Contains(err.Error(), "mock open slave error") {
			t.Errorf("Expected 'mock open slave error', got %v", err)
		}
	})
}
