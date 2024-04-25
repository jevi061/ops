package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/jevi061/ops/internal/ops"
	"github.com/jevi061/ops/internal/runner"
	"github.com/spf13/cobra"
)

var (
	tag     string
	opsfile string
	debug   bool
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
			o := ops.NewOps(conf, ops.WithDebug(debug))
			remoteRunners := make([]runner.Runner, 0)
			if conf.Servers != nil && len(conf.Servers.Names) > 0 {
				remoteRunners = o.PrepareRunners(conf.Servers.Names, tag)
			}
			taskRuns := make([]runner.TaskRun, 0)
			for _, taskName := range args {
				if runs, err := o.PrepareTaskRuns(taskName, remoteRunners); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				} else {
					taskRuns = append(taskRuns, runs...)
				}
			}
			runners := o.CollectRunners(taskRuns)
			if err := o.ConnectRunners(runners); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("CONNECT ERROR(%s):", err.Host), err)
				os.Exit(1)
			}
			defer o.CloseRunners(runners)
			o.SetRunnersRunningMode(runners, debug)
			if debug {
				o.AlignAndColorRunnersPromets(runners)
			}
			// relay signals to runners
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt)
			go func() {
				if err := o.RelaySignals(runners, signals); err != nil {
					fmt.Fprintln(os.Stderr, "RUN ERROR:", err)
					os.Exit(1)
				}
			}()
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
	runCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "run tasks in debug mode")
	return runCmd
}
