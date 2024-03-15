package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/jevi061/ops/internal/ops"
	"github.com/spf13/cobra"
)

var (
	tag     string
	opsfile string
	quiet   bool
)

func NewRunCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run TASK...",
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1)),
		Short: "Run tasks",
		Long:  `Run tasks defined in Opsfile,eg: ops run task1 task2 ...`,
		Run: func(cmd *cobra.Command, args []string) {
			conf, err := ops.NewOpsfileFromPath(opsfile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			o := ops.NewOps(conf, ops.WithDebug(!quiet))
			selectedServers := make([]*ops.Server, 0)
			if tag == "" {
				for _, v := range conf.Servers.Names {
					selectedServers = append(selectedServers, v)
				}
			} else {
				for _, v := range conf.Servers.Names {
					for _, t := range v.Tags {
						if t == tag {
							selectedServers = append(selectedServers, v)
						}
					}
				}
			}
			taskRuns, err := o.PrepareTaskRuns(selectedServers, args)
			if err != nil {
				var pe *ops.ParseError
				if errors.As(err, &pe) {
					fmt.Fprintln(os.Stderr, "PARSE ERROR:", err)
				} else {
					fmt.Fprintln(os.Stderr, err)
				}
				os.Exit(1)
			}
			runners := o.CollectRunners(taskRuns)
			if err := o.ConnectRunners(runners); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("CONNECT ERROR(%s):", err.Host), err)
				os.Exit(1)
			}
			defer o.CloseRunners(runners)
			o.SetRunnersRunningMode(runners, !quiet)
			if !quiet {
				o.AlignAndColorRunnersPromets(runners)
			}
			// relay signals to runners
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt)
			go o.RelaySignals(runners, signals)
			defer func() {
				signal.Stop(signals)
				close(signals)
			}()
			if err := o.Execute(taskRuns); err != nil {
				fmt.Fprintln(os.Stderr, "TASK ERROR:", err)
				os.Exit(1)
			}
		},
	}
	runCmd.PersistentFlags().StringVarP(&tag, "tag", "t", "", "computers tag")
	runCmd.PersistentFlags().StringVarP(&opsfile, "opsfile", "f", "./Opsfile.yml", "opsfile")
	runCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "run tasks in quiet mode")
	return runCmd
}
