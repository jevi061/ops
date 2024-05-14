package cmd

import (
	"fmt"
	"os"

	"github.com/jevi061/ops/internal/ops"
	"github.com/spf13/cobra"
)

var (
	tag     string
	opsfile string
	debug   bool
	dryRun  bool
)

func NewRunCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run TASK...",
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1)),
		Short: "Run tasks",
		Long:  `Run tasks defined in Opsfile, eg: ops run task1 task2 ...`,
		Run: func(cmd *cobra.Command, args []string) {
			conf, err := ops.NewOpsfileFromPath(opsfile)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			o := ops.NewOps(conf, ops.WithDebug(debug), ops.WithDryRun(dryRun))
			if err := o.Run(tag, args...); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}
	runCmd.PersistentFlags().StringVarP(&tag, "tag", "t", "", "server tag")
	runCmd.PersistentFlags().StringVarP(&opsfile, "opsfile", "f", "./Opsfile.yml", "opsfile")
	runCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "run tasks in debug mode")
	runCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "", false, "test task without applying changes")
	return runCmd
}
