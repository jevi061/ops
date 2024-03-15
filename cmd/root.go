package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ops",
	Short: "A Simple pipeline tool",
	Long:  `A simple pipeline tool that allows you to run shell commands on local or remote ssh servers`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func Execute() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(NewSShCommand())
	rootCmd.AddCommand(NewInitCmd())
	rootCmd.AddCommand(NewRunCmd())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
