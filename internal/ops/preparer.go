package ops

import (
	"fmt"

	"github.com/jevi061/ops/internal/connector"
	"github.com/jevi061/ops/internal/transfer"
)

type Preparer interface {
	Prepare(*Opsfile)
}

type connectorPreparer struct {
}

type connectorTaskPreparer struct {
	preparedExpandableTask map[string]int
}

func (p *connectorPreparer) Prepare(conf *Opsfile, tag string) []connector.Connector {
	localConnector := connector.NewLocalConnector()
	if conf.Servers != nil && len(conf.Servers.Names) > 0 {
		selectedServers := make([]*Server, 0)
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
		connectors := make([]connector.Connector, len(selectedServers))
		for i, c := range selectedServers {
			connectors[i] = connector.NewSSHConnector(c.Host,
				connector.WithPort(c.Port), connector.WithUser(c.User), connector.WithPassword(c.Password))
		}
		return append(connectors, localConnector)
	}
	return []connector.Connector{localConnector}
}

func (p *connectorTaskPreparer) Prepare(conf *Opsfile, tasks ...string) ([]connector.Task, error) {
	p.preparedExpandableTask = make(map[string]int)
	connectorTasks := make([]connector.Task, 0)
	for _, taskName := range tasks {
		if runs, err := p.PrepareTask(conf, taskName); err != nil {
			return nil, err
		} else {
			connectorTasks = append(connectorTasks, runs...)
		}
	}
	return connectorTasks, nil
}

func (p *connectorTaskPreparer) PrepareTask(conf *Opsfile, taskName string) ([]connector.Task, error) {

	tasks := make([]connector.Task, 0)
	// valid task
	if task, ok := conf.Tasks.Names[taskName]; ok {
		// deps
		if len(task.Deps) > 0 {
			p.preparedExpandableTask[taskName]++
			for _, depTaskName := range task.Deps {
				if p.preparedExpandableTask[taskName] > 1 && len(conf.Tasks.Names[depTaskName].Deps) > 0 {
					return nil, fmt.Errorf("ParseTaskError: found circular task node: %s", depTaskName)
				}
				if depTaskRuns, err := p.Prepare(conf, depTaskName); err != nil {
					return nil, err
				} else {
					tasks = append(tasks, depTaskRuns...)
				}
			}
		}
		// task itself
		if task.Transfer != "" { // upload task
			absSrc, dest, err := transfer.ParseTransferWithEnvs(task.Transfer, task.Envs)
			if err != nil {
				return nil, fmt.Errorf("invalid task: %s : %w", task.Name, err)
			}
			// build cmd
			cmd := fmt.Sprintf(`tar -C %s -xvzf - `, dest)
			stdin := transfer.PipeFile(absSrc)
			t := connector.NewCommonTask(connector.WithName(task.Name),
				connector.WithDesc(task.Desc),
				connector.WithShell(conf.Shell),
				connector.WithCommand(cmd),
				connector.WithEnvironments(task.Envs),
				connector.WithLocal(false),
				connector.WithStdin(stdin))
			tasks = append(tasks, t)

		} else {
			t := connector.NewCommonTask(connector.WithName(task.Name),
				connector.WithDesc(task.Desc),
				connector.WithShell(conf.Shell),
				connector.WithCommand(task.Cmd),
				connector.WithEnvironments(task.Envs),
				connector.WithLocal(task.Local))
			tasks = append(tasks, t)
		}
	} else { // invalid task
		return nil, &ParseError{target: taskName, Err: fmt.Errorf("%s is not a valid task", taskName)}
	}
	return tasks, nil

}
