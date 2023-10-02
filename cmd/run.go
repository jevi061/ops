package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"ops/ops"
	"os"

	"github.com/spf13/cobra"
)

var (
	tag     string
	opsfile string
	debug   bool
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
			data, err := ioutil.ReadFile(opsfile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			conf, err := ops.NewOpsfile(data)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			for k, v := range conf.Computers.Names {
				fmt.Println("computer:", k, "detail:", v)
			}
			for k, v := range conf.Tasks.Names {
				fmt.Printf("task:%s,detail:%vv\n", k, v)
			}
			for k, v := range conf.Environments.Envs {
				fmt.Println("env:", k, v)
			}
			o := ops.NewOps(conf, ops.WithDebug(debug))
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
			taskRuns, err := o.PrepareTaskRuns(selected, args)
			if err != nil {
				var pe *ops.ParseError
				var ce *ops.ConnectError
				if errors.As(err, &pe) {
					fmt.Fprintln(os.Stderr, "PARSE ERROR:", err)
				} else if errors.As(err, &ce) {
					fmt.Fprintln(os.Stderr, fmt.Sprintf("CONNECT ERROR(%s):", ce.Host), err)
				} else {
					fmt.Fprintln(os.Stderr, err)
				}
				os.Exit(1)
			}
			runners := o.CollectRunners(taskRuns)
			defer o.CloseRunners(runners)
			o.SetRunnersRunningMode(runners, debug)
			if debug {
				o.AlignAndColorRunnersPromets(runners)
			}
			if err := o.Execute(taskRuns); err != nil {
				var (
					ce *ops.ConnectError
					te *ops.RunError
				)
				if errors.As(err, &ce) {
					fmt.Fprintln(os.Stderr, fmt.Sprintf("CONNECT ERROR(%s):", ce.Host), err)
				} else if errors.As(err, &te) {
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
	runCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "run task|pipeline in debug mode")
	return runCmd
}
