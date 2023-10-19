package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jevi061/ops/internal/ops"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

var (
	ofile string
)

func NewSShCommand() *cobra.Command {
	var sshCmd = &cobra.Command{
		Use:   "ssh",
		Args:  cobra.MatchAll(cobra.MaximumNArgs(1), cobra.MinimumNArgs(1)),
		Short: "Open a shell to target remote computer",
		Long:  `Open a shell through ssh to remote computer,eg: ops ssh www.example.com`,
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here

			host := args[0]
			conf, err := ops.NewOpsfileFromPath(ofile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			c, ok := conf.Computers.Names[host]
			if !ok {
				fmt.Fprintln(os.Stderr, "No computer matched to :", host, "in", ofile)
				os.Exit(1)
			}
			config := &ssh.ClientConfig{
				User: c.User,
				Auth: []ssh.AuthMethod{
					ssh.Password(c.Password),
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         5 * time.Second,
			}
			conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), config)
			if err != nil {
				fmt.Fprintln(os.Stderr, "connect to :", host, "failed:", err)
				os.Exit(1)
			}
			defer conn.Close()
			session, err := conn.NewSession()
			if err != nil {
				fmt.Fprintln(os.Stderr, "open session to :", host, "failed:", err)
				os.Exit(1)
			}
			defer session.Close()
			modes := ssh.TerminalModes{
				ssh.ECHO:          0, // disable echoing
				ssh.TTY_OP_ISPEED: 14400,
				ssh.TTY_OP_OSPEED: 14400,
			}
			w, h, _ := term.GetSize(int(os.Stdout.Fd()))
			// Request pseudo terminal
			if err := session.RequestPty("xterm", h, w, modes); err != nil {
				fmt.Fprintln(os.Stderr, "request pty to :", host, "failed:", err)
				os.Exit(1)
			}
			if outPipe, err := session.StdoutPipe(); err != nil {
				fmt.Fprintln(os.Stderr, "open remote stdout failed:", err)
				os.Exit(1)
			} else {
				go func() {
					io.Copy(os.Stdout, outPipe)
				}()
			}
			inPipe, err := session.StdinPipe()
			if err != nil {
				fmt.Fprintln(os.Stderr, "open remote std in failed:", err)
				os.Exit(1)
			}
			if errPipe, err := session.StderrPipe(); err != nil {
				fmt.Fprintln(os.Stderr, "open remote stderr failed:", err)
				os.Exit(1)
			} else {
				go func() {
					io.Copy(os.Stderr, errPipe)
				}()
			}
			if err := session.Shell(); err != nil {
				fmt.Fprintln(os.Stderr, "open session to :", host, "failed:", err)
				os.Exit(1)
			}
			go func() {
				signals := make(chan os.Signal, 1)
				signal.Notify(signals, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT)
				for {
					sig, ok := <-signals
					if !ok {
						return
					}
					switch sig {
					case syscall.SIGINT:
						inPipe.Write([]byte("\x03"))
						session.Signal(ssh.SIGINT)
					case syscall.SIGQUIT:
						session.Signal(ssh.SIGQUIT)
					case syscall.SIGHUP:
						session.Signal(ssh.SIGHUP)
					}
				}
			}()
			input := bufio.NewReader(os.Stdin)
			go func() {
				for {
					str, _ := input.ReadString('\n')
					if _, err := fmt.Fprint(inPipe, str); err != nil {
						os.Exit(1)
					}
				}
			}()
			if err := session.Wait(); err != nil {
				fmt.Println("failed:", err)
			}

		},
	}
	sshCmd.PersistentFlags().StringVarP(&ofile, "opsfile", "f", "./Opsfile.yml", "opsfile")
	return sshCmd
}
