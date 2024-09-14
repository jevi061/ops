package connector

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"

	"github.com/rs/xid"
)

type LocalConnector struct {
	id      string
	local   bool
	host    string
	user    string
	running bool
	exec    *exec.Cmd
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	stdin   io.WriteCloser
	promet  string
}

var shellCommandArgs = map[string]string{
	"sh":   "-c",
	"bash": "-c",
}

func NewLocalConnector() *LocalConnector {
	return &LocalConnector{id: xid.New().String(), local: true, host: "localhost"}
}
func (r *LocalConnector) Connect() error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	r.user = u.Username
	return nil
}

func (r *LocalConnector) Close() error {
	return nil
}

func (r *LocalConnector) Stdin() io.WriteCloser {
	return r.stdin
}

func (r *LocalConnector) Stdout() io.Reader {
	return r.stdout
}

func (r *LocalConnector) Stderr() io.Reader {
	return r.stderr
}

func (r *LocalConnector) Run(tr Task, options *RunOptions) error {
	if r.running {
		return errors.New("connector is already running")
	}
	if !options.DryRun {
		r.running = true
	}
	flag, ok := shellCommandArgs[tr.Shell()]
	if !ok {
		return fmt.Errorf("shell: [%s] is not supported, please use sh、bash instead", tr.Shell())
	}
	for _, trCmd := range tr.Commands() {
		cmd := exec.Command(tr.Shell(), flag, trCmd)
		jenvs := make([]string, 0)
		for k, v := range tr.Environments() {
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
		if options.Debug || options.DryRun {
			fmt.Printf("%s%s %s %s\n", r.Promet(), tr.Shell(), flag, trCmd)
		}
		if !options.DryRun {
			if err := r.exec.Start(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *LocalConnector) Upload(src, dest string) error {
	return errors.New("upload task is not allowed to run on local")
}

func (r *LocalConnector) Wait() error {
	if !r.running {
		return errors.New("wait on non running cmd is not allowed")
	}
	err := r.exec.Wait()
	r.running = false
	return err
}

func (r *LocalConnector) Promet() string {
	if r.promet != "" {
		return r.promet
	}
	return fmt.Sprintf("%s@localhost | ", r.user)
}
func (r *LocalConnector) SetPromet(promet string) {
	r.promet = promet
}

func (r *LocalConnector) Host() string {
	return r.host
}

func (r *LocalConnector) ID() string {
	return r.id
}
func (r *LocalConnector) Local() bool {
	return r.local
}
func (r *LocalConnector) Signal(sig os.Signal) error {
	if !r.running {
		return nil
	}
	return r.exec.Process.Signal(sig)
}
