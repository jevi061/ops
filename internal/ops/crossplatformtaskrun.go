package ops

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/containerd/console"
	"github.com/gookit/color"
	"github.com/jevi061/ops/internal/prefixer"
	"github.com/jevi061/ops/internal/runner"
)

// CrossplatformTaskRun is minimum unit of task with target runners for ops to run
type CrossplatformTaskRun struct {
	runners []runner.Runner
	task    *Task
	envs    map[string]string
	input   io.Reader // channel for transfer data to remote stdin
	//inputTrigger func()    // func to trigger input
}

func (tr *CrossplatformTaskRun) MustParse(cmdline string) (string, []string) {
	var (
		cmd  string
		args []string
	)
	parts := strings.Split(cmdline, " ")
	if len(parts) == 1 {
		cmd = parts[0]
	} else if len(parts) > 1 {
		cmd = parts[0]
		args = parts[1:]
	} else {
		panic("invlid task")
	}
	return cmd, args
}
func (tr *CrossplatformTaskRun) Command() string {
	// if tr.task.LocalCmd != "" {
	// 	return &runner.Job{Cmd: tr.task.LocalCmd, Envs: tr.envs}
	// }
	if strings.Contains(tr.task.Cmd, "->") {
		cmd := strings.TrimSpace(tr.task.Cmd)
		feilds := strings.Fields(cmd)
		return fmt.Sprintf(`tar -xvzf - -C %s`, feilds[2])
	}
	if tr.task.Cmd != "" {
		return tr.task.Cmd
	}
	// if tr.task.Upload != nil {
	// 	return &runner.Job{Cmd: fmt.Sprintf("tar -C %s -xzf -", tr.task.Upload.Dest), Envs: tr.envs, Input: tr.input}
	// }
	return ""
}
func (tr *CrossplatformTaskRun) Environments() map[string]string {
	return tr.envs
}
func (tr *CrossplatformTaskRun) Stdin() io.Reader {
	return tr.input
}
func (tr *CrossplatformTaskRun) Runners() []runner.Runner {
	return tr.runners
}

// NewTaskRun create TaskRun with global environments
func NewTaskRun(task *Task, envs map[string]string, runners []runner.Runner) (runner.TaskRun, error) {
	if task == nil {
		return nil, errors.New("empty task not allowed")
	}
	if len(runners) <= 0 {
		return nil, fmt.Errorf("no runners provided to run task:%s", task.Name)
	}
	if task.Cmd == "" {
		return nil, fmt.Errorf("no cmd found for task: %s", task.Name)
	}
	// apply global environments to task
	vs := make(map[string]string)
	for k, v := range envs {
		vs[k] = v
	}
	for k, v := range task.Envs {
		vs[k] = v
	}
	task.Envs = vs
	// expand cmd with envs
	cmd := os.Expand(strings.TrimSpace(task.Cmd), func(s string) string { return task.Envs[s] })
	task.Cmd = cmd
	// upload
	if strings.Contains(cmd, "->") {
		fields := strings.Fields(cmd)
		if len(fields) != 3 {
			return nil, fmt.Errorf("incorrect file transfer syntex,use: LOCAL_SRC -> REMOTE_DIRECTORY ")
		}
		absSrc, err := filepath.Abs(fields[0])
		if err != nil {
			return nil, fmt.Errorf("resolve upload src file path failed:%w", err)
		}
		pr, err := pipeFiles(absSrc)
		if err != nil {
			return nil, err
		}
		return &CrossplatformTaskRun{task: task, envs: vs, input: pr, runners: runners}, nil
		// download
	} else if strings.Contains(cmd, "<-") {
		return nil, errors.New("cant download file now")
	}
	if task.Local {
		r := runner.NewLocalRunner()
		return &CrossplatformTaskRun{task: task, envs: vs, runners: []runner.Runner{r}}, nil
	}
	return &CrossplatformTaskRun{task: task, envs: vs, input: nil, runners: runners}, nil
}

func pipeFiles(src string) (io.Reader, error) {
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
func (tr *CrossplatformTaskRun) Sudo() bool {
	return tr.task.Sudo
}

// Run execute internal task
func (tr *CrossplatformTaskRun) Run() error {
	fmt.Println("")
	gray := color.Gray.Render
	bold := color.Bold.Render
	fmt.Printf("%s [%s] %s\n", bold("Task:"), bold(tr.task.Name), gray(tr.task.Desc))
	current := console.Current()
	ws, err := current.Size()
	if err != nil {
		fmt.Println(strings.Repeat("-", 10))
	} else {
		fmt.Println(strings.Repeat("-", int(ws.Width)))
	}
	var (
		wg sync.WaitGroup
	)
	for _, r := range tr.runners {
		//job := tr.GenerateRunnerJob()
		if err := r.Run(tr); err != nil {
			return &RunError{err: err}
		}
		if r.Debug() {
			// copy remote computer's stdout to current
			wg.Add(1)
			go func(rn runner.Runner) {
				defer wg.Done()
				_, err := io.Copy(os.Stdout, prefixer.NewPrefixReader(rn.Stdout(), rn.Promet()))
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
			}(r)
		}
		// copy remote computer's stderr to current
		wg.Add(1)
		go func(rn runner.Runner) {
			defer wg.Done()
			_, err := io.Copy(os.Stderr, prefixer.NewPrefixReader(rn.Stderr(), rn.Promet()))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}(r)
		if tr.Stdin() != nil {
			wg.Add(1)
			go func(rn runner.Runner) {
				defer wg.Done()
				defer rn.Stdin().Close()
				io.Copy(rn.Stdin(), tr.Stdin())
			}(r)
		}
	}
	wg.Wait()
	for _, c := range tr.runners {
		if err := c.Wait(); err != nil {
			color.Redf("[%s]: failed:%s\n", c.Host(), err.Error())
		} else {
			color.Greenf("[%s]: ok\n", c.Host())
		}
	}
	return nil
}
