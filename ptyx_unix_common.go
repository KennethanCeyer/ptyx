//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package ptyx

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

type unixSession struct {
	cmd    *exec.Cmd
	master *os.File
}

func Spawn(ctx context.Context, opts SpawnOpts) (sess Session, err error) {
	if opts.Prog == "" {
		return nil, errors.New("ptyx: empty program")
	}
	m, s, err := openPTY()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = m.Close()
			_ = s.Close()
		}
	}()

	cmd := exec.CommandContext(ctx, opts.Prog, opts.Args...)
	cmd.Env = opts.Env
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = s, s, s
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}

	if opts.Cols > 0 && opts.Rows > 0 {
		_ = setWinsize(int(m.Fd()), opts.Cols, opts.Rows)
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}
	_ = s.Close()

	return &unixSession{cmd: cmd, master: m}, nil
}

func (s *unixSession) PtyReader() io.Reader { return s.master }
func (s *unixSession) PtyWriter() io.Writer { return s.master }
func (s *unixSession) Resize(cols, rows int) error { return setWinsize(int(s.master.Fd()), cols, rows) }
func (s *unixSession) Wait() error {
	err := s.cmd.Wait()
	if exitErr, ok := err.(*exec.ExitError); ok {
		return &ExitError{
			ExitCode:   exitErr.ExitCode(),
			waitStatus: exitErr.Sys(),
		}
	}
	return err
}
func (s *unixSession) Kill() error { return s.cmd.Process.Kill() }
func (s *unixSession) Close() error { return s.master.Close() }
func (s *unixSession) Pid() int { return s.cmd.Process.Pid }

func (s *unixSession) CloseStdin() error {
	return s.master.Close()
}

func setWinsize(fd int, cols, rows int) error {
	ws := &unix.Winsize{Col: uint16(cols), Row: uint16(rows)}
	return unix.IoctlSetWinsize(fd, unix.TIOCSWINSZ, ws)
}

func clen(b []byte) int {
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			return i
		}
	}
	return len(b)
}

func ioctl(fd, op, arg uintptr) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, op, arg)
	if errno != 0 {
		return errno
	}
	return nil
}
