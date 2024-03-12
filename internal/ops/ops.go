package ops

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strings"

	"github.com/gookit/color"
	"github.com/jevi061/ops/internal/runner"
)

type Ops struct {
	conf  *Opsfile
	debug bool
}

type OpsOption func(*Ops)

func WithDebug(debug bool) OpsOption {
	return func(o *Ops) {
		o.debug = debug
	}
}

func NewOps(conf *Opsfile, options ...OpsOption) *Ops {
	ops := &Ops{conf: conf}
	for _, v := range options {
		v(ops)
	}
	return ops
}

type ConnectError struct {
	Host string
	Err  error
}
type RunError struct {
	host string
	err  error
}
type ParseError struct {
	target string
	Err    error
}

func (te *RunError) Error() string {
	return te.err.Error()
}

func (ce *ConnectError) Error() string {
	return ce.Err.Error()
}
func (pe *ParseError) Error() string {
	return pe.Err.Error()
}
func (ops *Ops) PrepareTaskRuns(servers []*Server, tasks []string) ([]runner.TaskRun, error) {
	// prepare runners for computers
	runners := make([]runner.Runner, 0)
	for _, c := range servers {
		runners = append(runners, runner.NewSSHRunner(c.Host,
			runner.WithPort(c.Port), runner.WithUser(c.User), runner.WithPassword(c.Password)))
	}
	// prepare TaskRuns
	runs := make([]runner.TaskRun, 0)
	for _, t := range tasks {
		// dependencies
		if task, ok := ops.conf.Tasks.Names[t]; ok {
			for _, dep := range task.Deps {
				if depTask, ok := ops.conf.Tasks.Names[dep]; !ok {
					return nil, &ParseError{target: dep, Err: fmt.Errorf("task: %s has invalid dependency: %s", t, dep)}
				} else {
					run, err := NewTaskRun(depTask, ops.conf.Environments.Envs, runners)
					if err != nil {
						return nil, fmt.Errorf("parse task: %s error:%w", t, err)
					}
					runs = append(runs, run)
				}
			}
			// task itself
			run, err := NewTaskRun(task, ops.conf.Environments.Envs, runners)
			if err != nil {
				return nil, fmt.Errorf("parse task: %s error:%w", t, err)
			}
			runs = append(runs, run)
		} else {
			return nil, &ParseError{target: t, Err: fmt.Errorf("%s is not a valid task", t)}
		}
	}
	return runs, nil

}

func (ops *Ops) ConnectRunners(runners []runner.Runner) *ConnectError {
	for _, runner := range runners {
		if err := runner.Connect(); err != nil {
			return &ConnectError{Host: runner.Host(), Err: err}
		}
	}
	return nil
}
func (ops *Ops) CollectRunners(taskRuns []runner.TaskRun) []runner.Runner {
	runners := make([]runner.Runner, 0)
	cache := make(map[string]runner.Runner, 0)
	for _, run := range taskRuns {
		for _, r := range run.Runners() {
			if _, ok := cache[r.ID()]; !ok {
				runners = append(runners, r)
				cache[r.ID()] = r
			}
		}
	}
	return runners
}
func (ops *Ops) SetRunnersRunningMode(runners []runner.Runner, debug bool) {
	for _, r := range runners {
		r.SetDebug(debug)
	}
}
func (ops *Ops) AlignAndColorRunnersPromets(runners []runner.Runner) {
	//fmt.Println("align runners promets")
	// align and color runners promets
	colors := []func(a ...interface{}) string{color.Yellow.Render, color.Cyan.Render,
		color.Magenta.Render, color.Blue.Render}
	max := 0

	for _, r := range runners {
		prefixLen := len(r.Promet())
		if prefixLen > max {
			max = prefixLen
		}
	}
	//fmt.Println("max promets len:", max)
	hash := fnv.New32a()

	for _, r := range runners {
		prefixLen := len(r.Promet())
		if prefixLen <= max { // Left padding.
			p := strings.Repeat(" ", max-prefixLen) + r.Promet()
			hash.Write([]byte(p))
			color := colors[int(hash.Sum32())%len(colors)]
			r.SetPromet(color(p))
			//fmt.Println("update runner:", r.Host(), "promets with:", max-prefixLen, "blanks")
		}
	}

}
func (ops *Ops) Execute(taskRuns []runner.TaskRun) error {
	// max := 0
	// for _, run := range taskRuns {
	// 	if len(run.task.Name) > max {
	// 		max = len(run.task.Name)
	// 	}
	// }
	//green, blue, red := color.New(color.FgHiGreen).Add(color.Bold), color.New(color.FgBlue).Add(color.Bold), color.New(color.FgRed)
	for _, run := range taskRuns {
		run.Run()
	}

	return nil
}

func (ops *Ops) CloseRunners(runners []runner.Runner) error {
	for _, r := range runners {
		if err := r.Close(); err != nil {
			fmt.Println("close runners failed:", err)
		}
	}
	return nil
}

// RelaySignals realy incoming signals to avaliable runners, it will block until signals chan closed
func (ops *Ops) RelaySignals(runners []runner.Runner, signals chan os.Signal) error {
	for {

		sig, ok := <-signals
		if !ok {
			return nil
		}

		for _, r := range runners {
			fmt.Println("send sig:", sig, "to runner:", r.Host())
			err := r.Signal(sig)
			if err != nil {
				return errors.New("send signal to runner failed")
			}
		}

	}
}
