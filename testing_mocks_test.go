package ptyx

import (
	"bytes"
	"io"
	"os"
)

type mockConsole struct {
	in     io.ReadCloser
	outBuf *bytes.Buffer
}

func newMockConsole(input string) *mockConsole {
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		if input != "" {
			_, _ = io.WriteString(w, input)
		}
	}()
	return &mockConsole{in: r, outBuf: &bytes.Buffer{}}
}

func (m *mockConsole) In() io.Reader             { return m.in }
func (m *mockConsole) Out() io.Writer            { return m.outBuf }
func (m *mockConsole) Err() *os.File             { panic("not implemented") }
func (m *mockConsole) IsATTYOut() bool           { return true }
func (m *mockConsole) Size() (int, int)          { return 80, 24 }
func (m *mockConsole) MakeRaw() (RawState, error)  { return nil, nil }
func (m *mockConsole) Restore(RawState) error      { return nil }
func (m *mockConsole) EnableVT()                 {}
func (m *mockConsole) OnResize() <-chan struct{}   { return make(chan struct{}) }
func (m *mockConsole) Close() error              { return m.in.Close() }

type mockSession struct {
	ptyIn  *bytes.Buffer
	ptyOut io.Reader
	closeStdinFunc func() error
	waitFunc       func() error
	closeFunc      func() error
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
func (m *mockSession) Wait() error {
	if m.waitFunc != nil {
		return m.waitFunc()
	}
	return nil
}
func (m *mockSession) Kill() error                 { return nil }
func (m *mockSession) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}
func (m *mockSession) Pid() int                    { return 1234 }
func (m *mockSession) CloseStdin() error {
	if m.closeStdinFunc != nil {
		return m.closeStdinFunc()
	}
	return nil
}

type mockMux struct {
	startErr error
	stopErr  error
}

func (m *mockMux) Start(c Console, s Session) error { return m.startErr }
func (m *mockMux) Stop() error                      { return m.stopErr }
