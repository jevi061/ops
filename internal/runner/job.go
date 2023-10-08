package runner

import (
	"io"
)

// Job is unit of work(local or remote command) for runner to run
type Job struct {
	Cmd   string
	Envs  map[string]string
	Input io.Reader
}
