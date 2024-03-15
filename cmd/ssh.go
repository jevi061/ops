package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/containerd/console"
	"github.com/jevi061/ops/internal/ops"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var (
	ofile string
)

func NewSShCommand() *cobra.Command {
	var sshCmd = &cobra.Command{
		Use:   "ssh SERVER_NAME",
		Args:  cobra.MatchAll(cobra.MaximumNArgs(1), cobra.MinimumNArgs(1)),
		Short: "Open a shell to target remote server",
		Long:  `Open a shell through ssh to remote server,eg: ops ssh www.example.com`,
		Run: func(cmd *cobra.Command, args []string) {
			serverName := args[0]
			conf, err := ops.NewOpsfileFromPath(ofile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			c, ok := conf.Servers.Names[serverName]
			if !ok {
				fmt.Fprintln(os.Stderr, "No server name matched to :", serverName, "in", ofile)
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
				fmt.Fprintln(os.Stderr, "connect to :", serverName, "failed:", err)
				os.Exit(1)
			}
			defer conn.Close()
			session, err := conn.NewSession()
			if err != nil {
				fmt.Fprintln(os.Stderr, "open session to :", serverName, "failed:", err)
				os.Exit(1)
			}
			defer session.Close()
			session.Stdout = os.Stdout
			session.Stderr = os.Stderr
			session.Stdin = os.Stdin
			modes := ssh.TerminalModes{
				ssh.ECHO:          1, // enable echoing
				ssh.ECHOCTL:       0,
				ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
				ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
				ssh.VSTATUS:       1,
			}
			current := console.Current()
			defer current.Reset()

			if err := current.SetRaw(); err != nil {
				fmt.Fprintln(os.Stderr, "make current console in raw mode failed:", err)
				os.Exit(1)
			}
			ws, err := current.Size()
			if err != nil {
				fmt.Fprintln(os.Stderr, "get current console size failed:", err)
				os.Exit(1)
			}
			current.Resize(ws)

			term := os.Getenv("TERM")
			if term == "" {
				term = "xterm-256color"
			}
			if err := session.RequestPty(term, int(ws.Height), int(ws.Width), modes); err != nil {
				fmt.Fprintln(os.Stderr, "request pty to :", serverName, "failed:", err)
				os.Exit(1)
			}

			if err := session.Shell(); err != nil {
				fmt.Fprintln(os.Stderr, "open session to :", serverName, "failed:", err)
				os.Exit(1)
			}
			if err := session.Wait(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		},
	}
	sshCmd.PersistentFlags().StringVarP(&ofile, "opsfile", "f", "./Opsfile.yml", "opsfile")
	return sshCmd
}
