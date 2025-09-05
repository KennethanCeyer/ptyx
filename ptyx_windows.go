//go:build windows

package ptyx

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type winSession struct {
	con     *conPty
	pid     int
	process windows.Handle
}

func Spawn(opts SpawnOpts) (sess Session, err error) {
	con, err := newConPty(opts.Cols, opts.Rows)
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

	var cmdlineBuilder strings.Builder
	cmdlineBuilder.WriteString(windows.EscapeArg(progPath))
	for _, arg := range opts.Args {
		cmdlineBuilder.WriteByte(' ')
		cmdlineBuilder.WriteString(windows.EscapeArg(arg))
	}

	siEx := new(windows.StartupInfoEx)
	siEx.StartupInfo.Cb = uint32(unsafe.Sizeof(*siEx))
	siEx.ProcThreadAttributeList = con.attrList.List()

	pi := new(windows.ProcessInformation)

	creationFlags := uint32(windows.CREATE_UNICODE_ENVIRONMENT) | windows.EXTENDED_STARTUPINFO_PRESENT

	pProgPath, err := windows.UTF16PtrFromString(progPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert program path to UTF16: %w", err)
	}
	pCmdline, err := windows.UTF16PtrFromString(cmdlineBuilder.String())
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
	env := append(os.Environ(), opts.Env...)

	err = windows.CreateProcess(
		pProgPath,
		pCmdline,
		nil,
		nil,
		false,
		creationFlags,
		createEnvBlock(env),
		pDir,
		&siEx.StartupInfo,
		pi,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	_ = windows.CloseHandle(pi.Thread)

	return &winSession{
		con:     con,
		pid:     int(pi.ProcessId),
		process: pi.Process,
	}, nil
}

func createEnvBlock(env []string) *uint16 {
	if len(env) == 0 {
		return nil
	}
	cleanEnv := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.ContainsRune(e, 0) {
			cleanEnv = append(cleanEnv, e)
		}
	}

	var block []uint16
	for _, e := range cleanEnv {
		u16, err := windows.UTF16FromString(e)
		if err != nil {
			continue
		}
		block = append(block, u16...)
	}
	block = append(block, 0)
	return &block[0]
}

func (s *winSession) PtyReader() io.Reader { return s.con.conout }
func (s *winSession) PtyWriter() io.Writer { return s.con.conin }
func (s *winSession) Resize(cols, rows int) error { return s.con.resize(cols, rows) }
func (s *winSession) Wait() error {
	status, err := windows.WaitForSingleObject(s.process, windows.INFINITE)
	if err != nil {
		return err
	}
	if status != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("unexpected wait status: %d", status)
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(s.process, &exitCode); err != nil {
		return err
	}
	if exitCode == 0 {
		return nil
	}
	return &ExitError{ExitCode: int(exitCode)}
}
func (s *winSession) Kill() error {
	return windows.TerminateProcess(s.process, 1)
}
func (s *winSession) Close() error {
	_ = s.Kill()
	_ = windows.CloseHandle(s.process)
	if s.con != nil {
		return s.con.Close()
	}
	return nil
}
func (s *winSession) Pid() int {
	return s.pid
}
