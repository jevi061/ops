package runner

import (
	"io"
	"os"
)

// TaskRun executable/runnable task
type TaskRun interface {
	Command() string
	Environments() map[string]string
	Stdin() io.Reader
	Runners() []Runner
	Run() error
}

// Runner local or remote runner for taskrun to run
type Runner interface {
	ID() string
	Connect() error
	Close() error
	Run(tr TaskRun) error
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
