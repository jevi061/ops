package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/rs/xid"
)

type LocalRunner struct {
	id      string
	host    string
	user    string
	running bool
	exec    *exec.Cmd
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	stdin   io.WriteCloser
	envs    []string //  Each entry is of the form "key=value".
	promet  string
	debug   bool // running in debug mode or not
}

func NewLocalRunner(envs map[string]string) *LocalRunner {
	envSlice := make([]string, len(envs))
	i := 0
	for k, v := range envs {
		envSlice[i] = k + "=" + v
	}
	return &LocalRunner{id: xid.New().String(), envs: envSlice, host: "localhost"}
}
func (r *LocalRunner) Connect() error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	r.user = u.Username
	return nil
}

func (r *LocalRunner) Close() error {
	return nil
}

func (r *LocalRunner) Stdin() io.WriteCloser {
	return r.stdin
}

func (r *LocalRunner) Stdout() io.Reader {
	return r.stdout
}

func (r *LocalRunner) Stderr() io.Reader {
	return r.stderr
}

func (r *LocalRunner) Run(c *Job, input InputFunc) error {
	if r.running {
		return errors.New("runner is already running")
	}
	r.running = true
	cmd := exec.Command(c.Cmd, c.Args...)
	cmd.Env = append(os.Environ(), r.envs...)
	r.exec = cmd
	var err error
	r.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}

	r.stderr, err = cmd.StderrPipe()
	if err != nil {
		return err
	}

	r.stdin, err = cmd.StdinPipe()
	if err != nil {
		return err
	}
	if r.debug {
		args := strings.Join(c.Args, " ")
		fmt.Printf("%s%s %s\n", r.Promet(), c.Cmd, args)
	}
	if err := r.exec.Start(); err != nil {
		return err
	}
	return nil
}

func (r *LocalRunner) Wait() error {
	if !r.running {
		return errors.New("wait on non running cmd is not allowed")
	}
	err := r.exec.Wait()
	r.running = false
	return err
}

func (r *LocalRunner) Promet() string {
	if r.promet != "" {
		return r.promet
	}
	return fmt.Sprintf("%s@localhost | ", r.user)
}
func (r *LocalRunner) SetPromet(promet string) {
	r.promet = promet
}

func (r *LocalRunner) Host() string {
	return r.host
}
func (r *LocalRunner) Debug() bool {
	return r.debug
}

func (r *LocalRunner) SetDebug(debug bool) {
	r.debug = debug
}
func (r *LocalRunner) ID() string {
	return r.id
}
func (r *LocalRunner) Signal(sig os.Signal) error {
	if !r.running {
		return errors.New("runner is not running")
	}
	return r.exec.Process.Signal(sig)
}
