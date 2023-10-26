package cmd

import (
	"fmt"
	"io"
	"os"
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
				ssh.ECHO:          1,     // enable echoing
				ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
				ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
			}

			fd := int(os.Stdin.Fd())

			originalState, err := term.MakeRaw(fd)
			if err != nil {
				os.Exit(1)
			}
			defer term.Restore(fd, originalState)
			termWidth, termHeight, _ := term.GetSize(fd)
			if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
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
			} else {
				go func() {
					io.Copy(inPipe, os.Stdin)
				}()
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
			if err := session.Wait(); err != nil {
				fmt.Println("failed:", err)
			}
		},
	}
	sshCmd.PersistentFlags().StringVarP(&ofile, "opsfile", "f", "./Opsfile.yml", "opsfile")
	return sshCmd
}
