package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type SSHRunner struct {
	id            string
	host          string
	port          uint
	user          string
	password      string
	conn          *ssh.Client
	session       *ssh.Session
	stdin         io.WriteCloser
	stdout        io.Reader
	stderr        io.Reader
	sessionOpened bool
	promet        string // output prefix
	debug         bool   // run job in debug mode or not
}
type SSHRunnerOption func(*SSHRunner)

func WithPort(port uint) SSHRunnerOption {
	return func(s *SSHRunner) {
		s.port = port
	}
}
func WithUser(user string) SSHRunnerOption {
	return func(s *SSHRunner) {
		s.user = user
	}
}
func WithPassword(password string) SSHRunnerOption {
	return func(s *SSHRunner) {
		s.password = password
	}
}
func NewSSHRunner(host string, options ...SSHRunnerOption) *SSHRunner {
	r := &SSHRunner{id: xid.New().String(), host: host, port: 22}
	for _, option := range options {
		option(r)
	}
	return r
}

func (r *SSHRunner) Connect() error {
	config := &ssh.ClientConfig{
		User: r.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(r.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", r.host, r.port), config)
	if err != nil {
		return err
	}
	r.conn = conn
	return nil
}
func (r *SSHRunner) Run(job *Job, input io.Reader) error {

	if r.sessionOpened {
		return errors.New("another seesion is using")
	}
	session, err := r.conn.NewSession()
	if err != nil {
		return err
	}
	r.session = session
	r.sessionOpened = true
	r.stdin, err = r.session.StdinPipe()
	if err != nil {
		return err
	}
	r.stdout, err = r.session.StdoutPipe()
	if err != nil {
		return err
	}
	r.stderr, err = r.session.StderrPipe()
	if err != nil {
		return err
	}
	for k, v := range job.Envs {
		r.session.Setenv(k, v)
	}
	args := strings.Join(job.Args, " ")
	if r.debug {
		fmt.Printf("%s%s %s\n", r.Promet(), job.Cmd, args)
	}
	if input == nil {
		// request pty
		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO: 0, // disable echoing
		}
		w, h, _ := term.GetSize(int(os.Stdout.Fd()))
		// Request pseudo terminal
		if err := r.session.RequestPty("xterm", h, w, modes); err != nil {
			return err
		}
	}

	if err := r.session.Start(fmt.Sprintf("%s %s", job.Cmd, args)); err != nil {
		return err
	}
	r.sessionOpened = true
	return nil

}
func (r *SSHRunner) Wait() error {
	if !r.sessionOpened {
		return errors.New("wait on closed ssh session is not allowed")
	}
	err := r.session.Wait()
	r.session.Close()
	r.sessionOpened = false
	return err
}
func (r *SSHRunner) Close() error {

	if r.sessionOpened {
		if err := r.session.Close(); err != nil {
			return err
		}
	}
	return r.conn.Close()
}

func (r *SSHRunner) Promet() string {
	if r.promet != "" {
		return r.promet
	}
	return fmt.Sprintf("%s@%s | ", r.user, r.host)
}
func (r *SSHRunner) SetPromet(promet string) {
	r.promet = promet
}
func (r *SSHRunner) Stdin() io.WriteCloser {
	return r.stdin
}
func (r *SSHRunner) Stdout() io.Reader {
	return r.stdout
}
func (r *SSHRunner) Stderr() io.Reader {
	return r.stderr
}

func (r *SSHRunner) Host() string {
	return r.host
}

func (r *SSHRunner) Debug() bool {
	return r.debug
}

func (r *SSHRunner) SetDebug(debug bool) {
	r.debug = debug
}
func (r *SSHRunner) ID() string {
	return r.id
}
func (r *SSHRunner) Signal(sig os.Signal) error {
	if !r.sessionOpened {
		return fmt.Errorf("session is not open")
	}

	switch sig {
	case os.Interrupt:
		// https://github.com/golang/go/issues/4115#issuecomment-66070418
		r.stdin.Write([]byte("\x03"))
		return r.session.Signal(ssh.SIGINT)
	default:
		return fmt.Errorf("siginal:%v not supported", sig)
	}
}
