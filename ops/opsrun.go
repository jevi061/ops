package ops

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"ops/runner"
	"os"
	"path/filepath"
	"strings"
)

// OpsRun is minimum unit of task for ops runner to run
type OpsRun struct {
	runners      []runner.Runner
	task         *Task
	input        io.Reader
	inputTrigger func() error // func to provide stdin for task
}

func (tr *OpsRun) MustParse(cmdline string) (string, []string) {
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
func (tr *OpsRun) GenerateRunnerJob() *runner.Job {
	if tr.task.LocalCmd != "" {
		cmd, args := tr.MustParse(tr.task.LocalCmd)
		return &runner.Job{Cmd: cmd, Args: args}
	}
	if tr.task.Cmd != "" {
		cmd, args := tr.MustParse(tr.task.Cmd)
		return &runner.Job{Cmd: cmd, Args: args}
	}
	if tr.task.Upload != nil {
		return &runner.Job{Cmd: fmt.Sprintf("tar -C %s -xzf -", tr.task.Upload.Dest), Input: tr.input}
	}
	return nil
}

func NewOpsRun(task *Task, envs map[string]string, runners []runner.Runner) (*OpsRun, error) {
	if task == nil {
		return nil, errors.New("empty task not allowed")
	}
	if len(runners) <= 0 {
		return nil, fmt.Errorf("no runners provided to run task:%s", task.Name)
	}
	if task.Cmd == "" && task.LocalCmd == "" && task.Upload == nil {
		return nil, errors.New("no cmd/local-cmd/upload directive found")
	}
	if task.LocalCmd != "" {
		r := runner.NewLocalRunner(envs)
		if err := r.Connect(); err != nil {
			return nil, err
		}
		return &OpsRun{task: task, runners: []runner.Runner{r}}, nil
	}
	if task.Upload != nil && task.Download != nil {
		return nil, errors.New("upload and download should seperated into different task")
	}

	if task.Upload != nil {
		// resovle file path
		if task.Upload.Src == "" || task.Upload.Dest == "" {
			return nil, fmt.Errorf("src and dest are required to upload file")
		}
		src, err := filepath.Abs(task.Upload.Src)
		if err != nil {
			return nil, fmt.Errorf("resolve upload src file path failed:%w", err)
		}
		absSrc := os.Expand(src, func(s string) string { return task.Envs[s] })
		fmt.Println("tar files:", absSrc)
		buf, trigger, err := tarFiles(absSrc)
		fmt.Println("buf:", buf, "trigger:", trigger)
		if err != nil {
			return nil, err
		}
		return &OpsRun{task: task, input: buf, inputTrigger: trigger, runners: runners}, nil

	}

	return &OpsRun{task: task, input: nil, runners: runners}, nil
}

func tarFiles(src string) (io.Reader, func() error, error) {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return nil, nil, fmt.Errorf("unable to tar: %s :%w", src, err)
	}
	buf := new(bytes.Buffer)
	return buf, func() error {

		gzw := gzip.NewWriter(buf)
		defer gzw.Close()

		tw := tar.NewWriter(gzw)
		defer tw.Close()

		// walk path
		return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
			fmt.Println("upload file:", file, "src:", src)
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
			f.Close()

			return nil
		})
	}, nil

}
