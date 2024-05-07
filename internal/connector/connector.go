package connector

import (
	"io"
	"os"
)

// Connector build a tunnel to run commands between host and local/remote servers.
type Connector interface {
	ID() string
	Local() bool
	Connect() error
	Close() error
	Run(Task) error
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
