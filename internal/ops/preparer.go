package ops

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jevi061/ops/internal/connector"
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
		mergedEnvs := mergeEnvs(conf.Environments.Envs, task.Envs)
		if strings.Contains(task.Cmd, "->") { // upload task
			fields := strings.Fields(task.Cmd)
			if len(fields) != 3 {
				return nil, fmt.Errorf("incorrect file transfer syntex,use: LOCAL_SRC -> REMOTE_DIRECTORY ")
			}
			absSrc, err := filepath.Abs(os.Expand(fields[0], func(s string) string { return task.Envs[s] }))
			if err != nil {
				return nil, fmt.Errorf("resolve upload src file path failed:%w", err)
			}
			// build cmd
			cmd := fmt.Sprintf(`tar -C %s -xvzf - `, fields[2])
			stdin := pipeFiles(absSrc)
			t := connector.NewCommonTask(connector.WithName(task.Name),
				connector.WithShell(conf.Shell),
				connector.WithCommand(cmd),
				connector.WithEnvironments(mergedEnvs),
				connector.WithLocal(false),
				connector.WithSudo(task.Sudo),
				connector.WithStdin(stdin))
			tasks = append(tasks, t)

		} else {
			t := connector.NewCommonTask(connector.WithName(task.Name),
				connector.WithShell(conf.Shell),
				connector.WithCommand(task.Cmd),
				connector.WithEnvironments(mergedEnvs),
				connector.WithLocal(task.Local),
				connector.WithSudo(task.Sudo))
			tasks = append(tasks, t)
		}
	} else { // invalid task
		return nil, &ParseError{target: taskName, Err: fmt.Errorf("%s is not a valid task", taskName)}
	}
	return tasks, nil

}
func mergeEnvs(base, special map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(special))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range special {
		merged[k] = v
	}
	return merged
}
func pipeFiles(src string) func() (io.Reader, error) {
	piper := func() (io.Reader, error) {
		//fmt.Println("pipe file:", src)
		pr, pw := io.Pipe()

		// ensure the src actually exists before trying to tar it
		if _, err := os.Stat(src); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil, err
		}

		gzipw := gzip.NewWriter(pw)

		tw := tar.NewWriter(gzipw)

		go func() {
			defer pw.Close()
			defer gzipw.Close()
			defer tw.Close()
			// walk path
			err := filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
				// return on any error
				if err != nil {
					return err
				}

				if !fi.Mode().IsRegular() {
					return nil
				}

				// create a new dir/file header
				header, err := tar.FileInfoHeader(fi, fi.Name())
				if err != nil {
					return err
				}
				pre := filepath.Dir(src)
				// update the name to correctly reflect the desired destination when untaring
				header.Name = strings.TrimPrefix(strings.Replace(file, pre, "", -1), string(os.PathSeparator))
				header.Name = strings.Replace(header.Name, string(os.PathSeparator), "/", -1)

				// write the header
				if err := tw.WriteHeader(header); err != nil {
					return err
				}

				// open files for taring
				f, err := os.Open(file)
				if err != nil {
					return err
				}

				// copy file data into tar writer
				if _, err := io.Copy(tw, f); err != nil {
					return err
				}

				// manually close here after each file operation; defering would cause each file close
				// to wait until all operations have completed.
				return f.Close()
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()

		return pr, nil
	}
	return piper

}
