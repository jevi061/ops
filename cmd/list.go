package cmd

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jevi061/ops/internal/ops"
	"github.com/spf13/cobra"
)

var (
	conf            string
	listServersOnly bool
	listTasksOnly   bool
)

func NewListCmd() *cobra.Command {
	boxStyle := table.StyleLight
	boxStyle.Options = table.OptionsNoBordersAndSeparators
	boxStyle.Options.SeparateHeader = true

	const (
		serverPrompet = "\nAvaliable servers:\n"
		taskPrompet   = "\nAvaliable tasks:\n"
	)
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

			stw := table.NewWriter()
			stw.SetStyle(boxStyle)
			stw.SetOutputMirror(os.Stdout)
			stw.AppendHeader(table.Row{"Server", "Host", "Port", "User"})
			for k, v := range conf.Servers.Names {
				stw.AppendRow(table.Row{k, v.Host, v.Port, v.User})
			}

			ttw := table.NewWriter()
			ttw.SetStyle(boxStyle)
			ttw.SetOutputMirror(os.Stdout)
			ttw.AppendHeader(table.Row{"Task", "Local", "Desc"})
			for _, task := range conf.Tasks.Names {
				ttw.AppendRow(table.Row{task.Name, task.Local, task.Desc})
			}
			if listServersOnly {
				fmt.Fprintln(os.Stdout, serverPrompet)
				stw.Render()
			} else if listTasksOnly {
				fmt.Fprintln(os.Stdout, taskPrompet)
				ttw.Render()
			} else {
				fmt.Fprintln(os.Stdout, serverPrompet)
				stw.Render()
				fmt.Fprintln(os.Stdout, taskPrompet)
				ttw.Render()
			}
		},
	}

	listCmd.PersistentFlags().StringVarP(&conf, "opsfile", "f", "./Opsfile.yml", "opsfile")
	listCmd.Flags().BoolVarP(&listServersOnly, "server-only", "s", false, "list avaliable servers without list tasks")
	listCmd.Flags().BoolVarP(&listTasksOnly, "task-only", "t", false, "list avaliable tasks without list servers")
	return listCmd
}
