package testptyx

import (
	"bytes"
	"io"
	"os"

	"github.com/KennethanCeyer/ptyx"
)

type MockConsole struct {
	InReader        io.Reader
	OutBuffer       *bytes.Buffer
	MakeRawError    error
	WaitError       error
	ForceWriteError error
}

func NewMockConsole(input string) *MockConsole {
	return &MockConsole{
		InReader:  bytes.NewBufferString(input),
		OutBuffer: &bytes.Buffer{},
	}
}

func (m *MockConsole) In() io.Reader             { return m.InReader }
func (m *MockConsole) Out() io.Writer {
	if m.ForceWriteError != nil {
		return &errorWriter{err: m.ForceWriteError}
	}
	return m.OutBuffer
}
func (m *MockConsole) Err() *os.File {
	return os.Stderr
}
func (m *MockConsole) IsATTYOut() bool           { return true }
func (m *MockConsole) Size() (int, int)          { return 80, 24 }
func (m *MockConsole) MakeRaw() (ptyx.RawState, error) {
	return nil, m.MakeRawError
}
func (m *MockConsole) Restore(state ptyx.RawState) error   { return nil }
func (m *MockConsole) EnableVT()                 {}
func (m *MockConsole) OnResize() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (m *MockConsole) Close() error              { return nil }

type MockSession struct {
	PtyInBuffer     *bytes.Buffer
	PtyOutReader    io.Reader
	WaitError       error
	ForceWriteError error
}

func NewMockSession(output string) *MockSession {
	return &MockSession{
		PtyInBuffer:  &bytes.Buffer{},
		PtyOutReader: bytes.NewBufferString(output),
	}
}

func (m *MockSession) PtyReader() io.Reader      { return m.PtyOutReader }
func (m *MockSession) PtyWriter() io.Writer {
	if m.ForceWriteError != nil {
		return &errorWriter{err: m.ForceWriteError}
	}
	return m.PtyInBuffer
}
func (m *MockSession) Resize(cols, rows int) error { return nil }
func (m *MockSession) Wait() error {
	return m.WaitError
}
func (m *MockSession) Kill() error                 { return nil }
func (m *MockSession) Close() error                { return nil }
func (m *MockSession) Pid() int                    { return 1234 }
func (m *MockSession) CloseStdin() error           { return nil }

type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}
