package connector

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jevi061/ops/internal/termsize"
	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type SSHConnector struct {
	id            string
	local         bool
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
type SSHTaskRunnerOption func(*SSHConnector)

func WithPort(port uint) SSHTaskRunnerOption {
	return func(s *SSHConnector) {
		if port <= 0 {
			s.port = 22
		} else {
			s.port = port
		}
	}
}
func WithUser(user string) SSHTaskRunnerOption {
	return func(s *SSHConnector) {
		s.user = user
	}
}
func WithPassword(password string) SSHTaskRunnerOption {
	return func(s *SSHConnector) {
		s.password = password
	}
}
func NewSSHConnector(host string, options ...SSHTaskRunnerOption) *SSHConnector {
	r := &SSHConnector{id: xid.New().String(), local: false, host: host, port: 22}
	for _, option := range options {
		option(r)
	}
	return r
}

func (r *SSHConnector) Connect() error {
	// prefer private key based auth method if home dir exists
	authMethods := make([]ssh.AuthMethod, 0)
	if homeDir, err := os.UserHomeDir(); err == nil {
		var signers []ssh.Signer
		for _, name := range []string{"id_rsa", "id_ecdsa", "id_ecdsa_sk", "id_ed25519", "id_ed25519_sk", "id_dsa"} {
			path := filepath.Join(homeDir, ".ssh", name)
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				if privateKey, err := os.ReadFile(path); err == nil {
					if signer, err := ssh.ParsePrivateKey(privateKey); err == nil {
						signers = append(signers, signer)
					}
				}
			}
		}
		if len(signers) > 0 {
			authMethods = append(authMethods, ssh.PublicKeys(signers...))
		}
	}
	if r.password != "" {
		authMethods = append(authMethods, ssh.Password(r.password))
	}
	if len(authMethods) <= 0 {
		fmt.Printf("%s@%s's password: ", r.user, r.host)
		if pass, err := term.ReadPassword(int(os.Stdin.Fd())); err != nil {
			return errors.New("read password failed")
		} else {
			fmt.Println("")
			r.password = string(pass)
			authMethods = append(authMethods, ssh.Password(r.password))
		}
	}
	config := &ssh.ClientConfig{
		User:            r.user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", r.host, r.port), config)
	if err != nil {
		if strings.Contains(err.Error(), "unable to authenticate") && !strings.Contains(err.Error(), "password") {
			fmt.Printf("%s@%s's password: ", r.user, r.host)
			if pass, err := term.ReadPassword(int(os.Stdin.Fd())); err != nil {
				return errors.New("read password failed")
			} else {
				fmt.Println("")
				r.password = string(pass)
				config.Auth = append(authMethods, ssh.Password(r.password))
				conn, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", r.host, r.port), config)
				if err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}
	r.conn = conn
	return nil
}
func (r *SSHConnector) Run(tr Task) error {

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
	stdout, err := r.session.StdoutPipe()
	if err != nil {
		return err
	}
	r.stderr, err = r.session.StderrPipe()
	if err != nil {
		return err
	}
	// setup sshpass
	sudoPrompt := fmt.Sprintf(`[sudo via ops, id=%s] password:`, r.ID())
	r.stdout = &passReader{host: r.host, user: r.user, password: r.password, expect: sudoPrompt, reader: bufio.NewReader(stdout), stdin: r.stdin}
	for k, v := range tr.Environments() {
		r.session.Setenv(k, v)
	}
	jenvs := make([]string, 0)
	for k, v := range tr.Environments() {
		jenvs = append(jenvs, fmt.Sprintf("%s=%s", k, v))
	}
	envStr := strings.Join(jenvs, " ")
	flag, ok := shellCommandArgs[tr.Shell()]
	if !ok {
		return fmt.Errorf("shell: [%s] is not supported,please use sh and bash instead", tr.Shell())
	}
	cmd := fmt.Sprintf("%s %s '%s'", tr.Shell(), flag, tr.Command())
	if strings.Contains(cmd, "sudo") {
		cmd = strings.ReplaceAll(cmd, "sudo", fmt.Sprintf(`sudo -E -p "%s"`, sudoPrompt))
	}
	cmd = envStr + " " + cmd
	if r.debug {
		fmt.Printf("%s%s\n", r.Promet(), cmd)
	}
	if tr.Stdin() == nil {
		// request pty
		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO:          0, // enable echoing
			ssh.ECHOCTL:       0,
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
			ssh.VSTATUS:       1,
		}
		w, h := termsize.DefaultSize(800, 600)
		// Request pseudo terminal
		if err := r.session.RequestPty("xterm", h, w, modes); err != nil {
			return err
		}

	}

	if err := r.session.Start(cmd); err != nil {
		return err
	}
	r.sessionOpened = true
	return nil

}

func (r *SSHConnector) Wait() error {
	if !r.sessionOpened {
		return errors.New("wait on closed ssh session is not allowed")
	}
	err := r.session.Wait()
	r.session.Close()
	r.sessionOpened = false
	return err
}
func (r *SSHConnector) Close() error {

	if r.sessionOpened {
		if err := r.session.Close(); err != nil {
			return err
		}
	}
	return r.conn.Close()
}

func (r *SSHConnector) Promet() string {
	if r.promet != "" {
		return r.promet
	}
	return fmt.Sprintf("%s@%s | ", r.user, r.host)
}
func (r *SSHConnector) SetPromet(promet string) {
	r.promet = promet
}
func (r *SSHConnector) Stdin() io.WriteCloser {
	return r.stdin
}
func (r *SSHConnector) Stdout() io.Reader {
	return r.stdout
}
func (r *SSHConnector) Stderr() io.Reader {
	return r.stderr
}

func (r *SSHConnector) Host() string {
	return r.host
}

func (r *SSHConnector) Debug() bool {
	return r.debug
}

func (r *SSHConnector) SetDebug(debug bool) {
	r.debug = debug
}
func (r *SSHConnector) ID() string {
	return r.id
}
func (r *SSHConnector) Local() bool {
	return r.local
}
func (r *SSHConnector) Signal(sig os.Signal) error {
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

type passReader struct {
	host     string
	user     string
	password string
	expect   string
	content  []byte //already readed content
	reader   *bufio.Reader
	stdin    io.Writer
}

func (pr *passReader) Read(data []byte) (int, error) {
	b, err := pr.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	if pr.content == nil {
		pr.content = make([]byte, 0)
	}
	pr.content = append(pr.content, b)
	if strings.Contains(string(pr.content), pr.expect) {
		pr.content = nil
		if pr.password == "" {
			fmt.Printf("%s@%s's password: ", pr.user, pr.host)
			if pass, err := term.ReadPassword(int(os.Stdin.Fd())); err != nil {
				return 0, errors.New("read password failed")
			} else {
				fmt.Println("")
				pr.password = string(pass)
			}
		}
		if _, err := io.Copy(pr.stdin, bytes.NewBuffer([]byte(pr.password+"\n"))); err != nil {
			return 0, err
		}
	}
	return copy(data, []byte{b}), nil
}
