package ptyx

import (
	"io"
	"os"
)

type Console interface {
	In() io.Reader
	Out() io.Writer
	Err() *os.File
	IsATTYOut() bool
	IsATTYErr() bool
	Size() (int, int)
	MakeRaw() (RawState, error)
	Restore(RawState) error
	EnableVT()
	OnResize() <-chan struct{}
	Close() error
}

type RawState interface{}

type Session interface {
	PtyReader() io.Reader
	PtyWriter() io.Writer
	Resize(cols, rows int) error
	Wait() error
	Kill() error
	Close() error
	Pid() int
	CloseStdin() error
}

type SpawnOpts struct {
	Prog string
	Args []string
	Env  []string
	Dir  string
	Cols int
	Rows int
}

type Mux interface {
	Start(c Console, s Session) error
	Stop() error
}
