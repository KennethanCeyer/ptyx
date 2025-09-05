//go:build windows

package ptyx

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type conPty struct {
	handle   windows.Handle
	conin    *os.File // The pipe for writing to the PTY (our stdin)
	conout   *os.File // The pipe for reading from the PTY (our stdout)
	attrList *windows.ProcThreadAttributeListContainer
	closeMu  sync.Mutex
	closed   bool
}

func newConPty(w, h int) (c *conPty, err error) {
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 25
	}

	var ptyIn, ptyOut windows.Handle
	var coninPipe, conoutPipe *os.File

	var ptyInRead, ptyInWrite windows.Handle
	if err = windows.CreatePipe(&ptyInRead, &ptyInWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("failed to create input pipe for pseudo console: %w", err)
	}
	defer func() {
		if err != nil {
			_ = windows.CloseHandle(ptyInRead)
			_ = windows.CloseHandle(ptyInWrite)
		}
	}()

	var ptyOutRead, ptyOutWrite windows.Handle
	if err = windows.CreatePipe(&ptyOutRead, &ptyOutWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("failed to create output pipe for pseudo console: %w", err)
	}
	defer func() {
		if err != nil {
			_ = windows.CloseHandle(ptyOutRead)
			_ = windows.CloseHandle(ptyOutWrite)
		}
	}()

	ptyIn = ptyInRead
	ptyOut = ptyOutWrite
	coninPipe = os.NewFile(uintptr(ptyInWrite), "pty-input-writer")
	conoutPipe = os.NewFile(uintptr(ptyOutRead), "pty-output-reader")

	size := windows.Coord{X: int16(w), Y: int16(h)}
	var hpc windows.Handle

	err = windows.CreatePseudoConsole(size, ptyIn, ptyOut, 0, &hpc)
	if err != nil {
		return nil, fmt.Errorf("failed to create pseudo console: %w", err)
	}

	_ = windows.CloseHandle(ptyIn)
	_ = windows.CloseHandle(ptyOut)

	attrList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, fmt.Errorf("failed to create proc thread attribute list: %w", err)
	}

	err = attrList.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, unsafe.Pointer(hpc), unsafe.Sizeof(hpc))
	if err != nil {
		attrList.Delete()
		return nil, fmt.Errorf("failed to update proc thread attributes: %w", err)
	}

	return &conPty{
		handle:   hpc,
		conin:    coninPipe,
		conout:   conoutPipe,
		attrList: attrList,
	}, nil
}

func (c *conPty) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true

	if c.attrList != nil {
		c.attrList.Delete()
	}

	windows.ClosePseudoConsole(c.handle)

	err1 := c.conin.Close()
	err2 := c.conout.Close()

	if err1 != nil {
		return err1
	}
	return err2
}

func (c *conPty) resize(w, h int) error {
	if c == nil {
		return nil
	}
	size := windows.Coord{X: int16(w), Y: int16(h)}
	return windows.ResizePseudoConsole(c.handle, size)
}
