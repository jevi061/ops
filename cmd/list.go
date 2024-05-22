package cmd

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jevi061/ops/internal/ops"
	"github.com/spf13/cobra"
)

var (
	conf string
)

func NewListCmd() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Args:    cobra.MatchAll(cobra.NoArgs),
		Short:   "List avaliable tasks",
		Long:    `List tasks defined in Opsfile`,
		Run: func(cmd *cobra.Command, args []string) {
			conf, err := ops.NewOpsfileFromPath(conf)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.AppendHeader(table.Row{"Task", "Local", "Desc"})
			for _, task := range conf.Tasks.Names {
				t.AppendRow(table.Row{task.Name, task.Local, task.Desc})
				t.AppendSeparator()
			}
			t.SetStyle(table.StyleLight)
			t.Render()
		},
	}

	listCmd.PersistentFlags().StringVarP(&conf, "opsfile", "f", "./Opsfile.yml", "opsfile")
	return listCmd
}
