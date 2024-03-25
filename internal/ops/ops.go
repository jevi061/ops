package ops

import (
	"fmt"
	"hash/fnv"
	"os"
	"strings"

	"github.com/gookit/color"
	"github.com/jevi061/ops/internal/runner"
)

type Ops struct {
	conf                   *Opsfile
	debug                  bool
	preparedExpandableTask map[string]int
}

type OpsOption func(*Ops)

func WithDebug(debug bool) OpsOption {
	return func(o *Ops) {
		o.debug = debug
	}
}

func NewOps(conf *Opsfile, options ...OpsOption) *Ops {
	ops := &Ops{conf: conf}
	ops.preparedExpandableTask = make(map[string]int)
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
func (ops *Ops) PrepareTaskRuns(taskName string, runners []runner.Runner) ([]runner.TaskRun, error) {
	runs := make([]runner.TaskRun, 0)
	// valid task
	if task, ok := ops.conf.Tasks.Names[taskName]; ok {
		// deps
		if len(task.Deps) > 0 {
			ops.preparedExpandableTask[taskName]++
			for _, depTaskName := range task.Deps {
				if ops.preparedExpandableTask[taskName] > 1 && len(ops.conf.Tasks.Names[depTaskName].Deps) > 0 {
					return nil, fmt.Errorf("ParseTaskError: found circular task node: %s", depTaskName)
				}
				if depTaskRuns, err := ops.PrepareTaskRuns(depTaskName, runners); err != nil {
					return nil, err
				} else {
					runs = append(runs, depTaskRuns...)
				}
			}
		}
		// task itself
		run, err := NewTaskRun(task, ops.conf.Environments.Envs, runners)
		if err != nil {
			return nil, fmt.Errorf("parse task: %s error:%w", taskName, err)
		}
		runs = append(runs, run)
	} else { // invalid task
		return nil, &ParseError{target: taskName, Err: fmt.Errorf("%s is not a valid task", taskName)}
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
func (ops *Ops) PrepareRunners(servers map[string]*Server, tag string) []runner.Runner {
	selectedServers := make([]*Server, 0)
	if tag == "" {
		for _, v := range servers {
			selectedServers = append(selectedServers, v)
		}
	} else {
		for _, v := range servers {
			for _, t := range v.Tags {
				if t == tag {
					selectedServers = append(selectedServers, v)
				}
			}
		}
	}
	runners := make([]runner.Runner, 0)
	for _, c := range servers {
		runners = append(runners, runner.NewSSHRunner(c.Host,
			runner.WithPort(c.Port), runner.WithUser(c.User), runner.WithPassword(c.Password)))
	}
	return runners
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
			err := r.Signal(sig)
			if err != nil {
				return fmt.Errorf("send signal to runner: [%s] failed", r.Host())
			}
		}
	}
}
