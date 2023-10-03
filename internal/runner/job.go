package runner

import (
	"errors"
	"io"
	"strings"
)

// Job is unit of work(local or remote command) for runner to run
type Job struct {
	Cmd   string
	Args  []string
	Envs  map[string]string
	Input io.Reader
}

type JobOption func(*Job)

func WithCmdline(cmdline string) JobOption {
	return func(j *Job) {
		if cmdline != "" {
			parts := strings.Split(cmdline, " ")
			if len(parts) == 1 {
				j.Cmd = cmdline
			} else {
				j.Cmd = parts[0]
				j.Args = parts[1:]
			}
		}

	}
}

func NewJob(name string, options ...JobOption) *Job {
	job := &Job{Cmd: name}
	for _, option := range options {
		option(job)
	}
	return job
}
func Parse(cmdline string) (*Job, error) {
	if cmdline == "" {
		return nil, errors.New("empty command is not allowed")
	}
	parts := strings.Split(cmdline, " ")
	if len(parts) == 1 {
		return &Job{Cmd: cmdline, Args: []string{}}, nil
	} else {
		return &Job{Cmd: parts[0], Args: parts[1:]}, nil
	}
}
