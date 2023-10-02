package runner

import (
	"io"
	"os"
)

type InputFunc func() error
type Runner interface {
	ID() string
	Connect() error
	Close() error
	Run(*Job, io.Reader) error
	Wait() error
	Stdin() io.WriteCloser
	Stderr() io.Reader
	Stdout() io.Reader
	Promet() string
	SetPromet(string)
	Host() string
	Debug() bool
	SetDebug(bool)
	Signal(os.Signal) error
}
