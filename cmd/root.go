package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ops",
	Short: "Simple pipeline tool",
	Long:  `A simple agentless pipeline tool for deploying applications to unix like systems`,
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
