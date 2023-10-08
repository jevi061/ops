package runner

import (
	"errors"
	"fmt"
	"github.com/rs/xid"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
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
	promet  string
	debug   bool // running in debug mode or not
}

func NewLocalRunner() *LocalRunner {
	return &LocalRunner{id: xid.New().String(), host: "localhost"}
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

func (r *LocalRunner) Run(c *Job, input io.Reader) error {
	if r.running {
		return errors.New("runner is already running")
	}
	r.running = true
	sh := "powershell"
	if runtime.GOOS != "windows" {
		sh = "bash -c"
	}
	cmd := exec.Command(sh, c.Cmd)
	jenvs := make([]string, 0)
	for k, v := range c.Envs {
		jenvs = append(jenvs, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = append(os.Environ(), jenvs...)
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
		fmt.Printf("%s%s %s\n", r.Promet(), sh, c.Cmd)
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
