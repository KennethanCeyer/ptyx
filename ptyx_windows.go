//go:build windows

package ptyx

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type winSession struct {
	con     *ConPty
	pid     int
	process windows.Handle
	thread  windows.Handle
	closeOnce sync.Once
}

func buildCommandLine(prog string, args []string) string {
	allArgs := make([]string, 0, 1+len(args))
	allArgs = append(allArgs, prog)
	allArgs = append(allArgs, args...)
	var b strings.Builder
	for i, v := range allArgs {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(windows.EscapeArg(v))
	}
	return b.String()
}

func buildEnvBlock(env []string) []uint16 {
	if len(env) == 0 {
		return nil
	}
	cleanEnv := make([]string, 0, len(env))
	for _, s := range env {
		if !strings.ContainsRune(s, 0) {
			cleanEnv = append(cleanEnv, s)
		}
	}
	blockStr := strings.Join(cleanEnv, "\x00") + "\x00\x00"
	return utf16.Encode([]rune(blockStr))
}

func Spawn(ctx context.Context, opts SpawnOpts) (Session, error) {
	con, err := NewConPty(opts.Cols, opts.Rows, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = con.Close()
		}
	}()

	progPath, err := exec.LookPath(opts.Prog)
	if err != nil {
		return nil, err
	}

	cmdline := buildCommandLine(progPath, opts.Args)
	siEx := new(windows.StartupInfoEx)
	siEx.Flags = windows.STARTF_USESTDHANDLES
	siEx.Cb = uint32(unsafe.Sizeof(*siEx))
	siEx.ProcThreadAttributeList = con.attrList.List()

	pi := new(windows.ProcessInformation)
	flags := uint32(windows.CREATE_UNICODE_ENVIRONMENT | windows.EXTENDED_STARTUPINFO_PRESENT)

	pCmdline, err := windows.UTF16PtrFromString(cmdline)
	if err != nil {
		return nil, fmt.Errorf("failed to convert command line to UTF16: %w", err)
	}

	var pDir *uint16
	if opts.Dir != "" {
		pDir, err = windows.UTF16PtrFromString(opts.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to convert directory to UTF16: %w", err)
		}
	}

	var env []string
	if opts.Env != nil {
		env = opts.Env
	} else {
		env = os.Environ()
	}
	envBlock := buildEnvBlock(env)
	var pEnv *uint16
	if len(envBlock) > 0 {
		pEnv = &envBlock[0]
	}

	err = windows.CreateProcess(
		nil,
		pCmdline,
		nil,
		nil,
		false,
		flags,
		pEnv,
		pDir,
		&siEx.StartupInfo,
		pi,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	go func() {
		<-ctx.Done()
		_ = windows.TerminateProcess(pi.Process, 1)
	}()

	return &winSession{
		con:     con,
		pid:     int(pi.ProcessId),
		process: pi.Process,
		thread:  pi.Thread,
	}, nil
}

func (s *winSession) PtyReader() io.Reader        { return s.con.outFile }
func (s *winSession) PtyWriter() io.Writer        { return s.con.inFile }
func (s *winSession) Resize(cols, rows int) error { return s.con.resize(cols, rows) }
func (s *winSession) Pid() int                    { return s.pid }

func (s *winSession) Wait() error {
	st, err := windows.WaitForSingleObject(s.process, windows.INFINITE)
	if err != nil {
		return err
	}
	if st != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("unexpected wait status: %d", st)
	}
	var code uint32
	if err := windows.GetExitCodeProcess(s.process, &code); err != nil {
		return err
	}
	if code == 0 {
		return nil
	}
	return &ExitError{ExitCode: int(code), waitStatus: nil}
}

func (s *winSession) Kill() error {
	return windows.TerminateProcess(s.process, 1)
}

func (s *winSession) Close() error {
	var err error
	s.closeOnce.Do(func() {
		if s.con != nil {
			err = s.con.Close()
		}
		if s.process != 0 {
			_ = windows.CloseHandle(s.process)
		}
		if s.thread != 0 {
			_ = windows.CloseHandle(s.thread)
		}
	})
	return err
}

func (s *winSession) CloseStdin() error {
	if s == nil || s.con == nil || s.con.inFile == nil {
		return nil
	}
	return s.con.inFile.Close()
}
