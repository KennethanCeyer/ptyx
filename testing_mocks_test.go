package ptyx

import (
	"bytes"
	"io"
	"os"
)

type mockConsole struct {
	in     io.Reader
	outBuf *bytes.Buffer
}

func newMockConsole(input string) *mockConsole {
	return &mockConsole{
		in:     bytes.NewBufferString(input),
		outBuf: &bytes.Buffer{},
	}
}

func (m *mockConsole) In() io.Reader               { return m.in }
func (m *mockConsole) Out() io.Writer              { return m.outBuf }
func (m *mockConsole) Err() *os.File               { panic("not implemented") }
func (m *mockConsole) IsATTYOut() bool             { return true }
func (m *mockConsole) IsATTYErr() bool             { return true }
func (m *mockConsole) Size() (int, int)            { return 80, 24 }
func (m *mockConsole) MakeRaw() (RawState, error)  { return nil, nil }
func (m *mockConsole) Restore(RawState) error      { return nil }
func (m *mockConsole) EnableVT()                   {}
func (m *mockConsole) OnResize() <-chan struct{}   { return make(chan struct{}) }
func (m *mockConsole) Close() error                { return nil }

type mockSession struct {
	ptyIn  *bytes.Buffer
	ptyOut io.Reader
}

func newMockSession(output string) *mockSession {
	return &mockSession{
		ptyIn:  &bytes.Buffer{},
		ptyOut: bytes.NewBufferString(output),
	}
}

func (m *mockSession) PtyReader() io.Reader      { return m.ptyOut }
func (m *mockSession) PtyWriter() io.Writer      { return m.ptyIn }
func (m *mockSession) Resize(cols, rows int) error { return nil }
func (m *mockSession) Wait() error                 { return nil }
func (m *mockSession) Kill() error                 { return nil }
func (m *mockSession) Close() error                { return nil }
func (m *mockSession) Pid() int                    { return 1234 }
func (m *mockSession) CloseStdin() error           { return nil }
