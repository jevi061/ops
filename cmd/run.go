package cmd

import (
	"errors"
	"fmt"
	"os"

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
		Use:   "run [task|pipeline...]",
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1)),
		Short: "Run tasks or pipelines",
		Long:  `Run tasks or pipelines defined in Opsfile,eg: ops run task1 task2 pipeline1`,
		Run: func(cmd *cobra.Command, args []string) {
			_, err := os.Stat(opsfile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			data, err := os.ReadFile(opsfile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			conf, err := ops.NewOpsfile(data)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			o := ops.NewOps(conf, ops.WithDebug(!quiet))
			selected := make([]*ops.Computer, 0)
			if tag == "" {
				for _, v := range conf.Computers.Names {
					selected = append(selected, v)
				}
			} else {
				for _, v := range conf.Computers.Names {
					for _, t := range v.Tags {
						if t == tag {
							selected = append(selected, v)
						}
					}
				}
			}
			taskRuns, err := o.PrepareOpsRuns(selected, args)
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
			if err := o.Execute(taskRuns); err != nil {
				var (
					te *ops.RunError
				)
				if errors.As(err, &te) {
					fmt.Fprintln(os.Stderr, "TASK ERROR:", err)
				} else {
					fmt.Fprintln(os.Stderr, err)
				}
				os.Exit(1)
			}
		},
	}
	runCmd.PersistentFlags().StringVarP(&tag, "tag", "t", "", "computers tag")
	runCmd.PersistentFlags().StringVarP(&opsfile, "opsfile", "f", "./Opsfile.yml", "opsfile")
	runCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "run task|pipeline in quiet mode")
	return runCmd
}
