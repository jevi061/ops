package ops

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/jevi061/ops/internal/multiwritecloser"
	"github.com/jevi061/ops/internal/prefixer"
	"github.com/jevi061/ops/internal/runner"
	"golang.org/x/term"
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
func (ops *Ops) PrepareOpsRuns(computers []*Computer, pipelines []string) ([]*OpsRun, error) {
	// prepare runners for computers
	runners := make([]runner.Runner, 0)
	for _, c := range computers {
		runners = append(runners, runner.NewSSHRunner(c.Host,
			runner.WithPort(c.Port), runner.WithUser(c.User), runner.WithPassword(c.Password)))
	}
	// connect to runners
	for _, r := range runners {
		if err := r.Connect(); err != nil {
			return nil, &ConnectError{Host: r.Host(), Err: err}
		}
	}
	// prepare TaskRuns
	runs := make([]*OpsRun, 0)
	for _, p := range pipelines {
		// pipeline
		if pipeline, ok := ops.conf.Pipelines.Names[p]; ok {
			for _, t := range pipeline {
				if task, ok := ops.conf.Tasks.Names[t]; !ok {
					return nil, &ParseError{target: p, Err: fmt.Errorf("task: %s is not defined in pipeline: %s", t, p)}
				} else {
					run, err := NewOpsRun(task, ops.conf.Environments.Envs, runners)
					if err != nil {
						return nil, fmt.Errorf("parse task: %s error:%w", t, err)
					}
					runs = append(runs, run)
				}
			}
			// task
		} else if t, ok := ops.conf.Tasks.Names[p]; ok {
			run, err := NewOpsRun(t, ops.conf.Environments.Envs, runners)
			if err != nil {
				return nil, fmt.Errorf("parse task: %s error:%w", t.Name, err)
			}
			runs = append(runs, run)
		} else {
			return nil, &ParseError{target: p, Err: fmt.Errorf("%s is not a pipeline or a task", p)}
		}
	}
	return runs, nil

}

func (ops *Ops) CollectRunners(taskRuns []*OpsRun) []runner.Runner {
	runners := make([]runner.Runner, 0)
	cache := make(map[string]runner.Runner, 0)
	for _, run := range taskRuns {
		for _, r := range run.runners {
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
	colors := []func(a ...interface{}) string{color.New(color.FgBlack).SprintFunc(), color.New(color.FgYellow).SprintFunc(),
		color.New(color.FgMagenta).SprintFunc(), color.New(color.FgCyan).SprintFunc()}
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
func (ops *Ops) Execute(taskRuns []*OpsRun) error {
	max := 0
	for _, run := range taskRuns {
		if len(run.task.Name) > max {
			max = len(run.task.Name)
		}
	}
	green, blue, red := color.New(color.FgHiGreen).Add(color.Bold), color.New(color.FgBlue), color.New(color.FgRed)
	for _, run := range taskRuns {
		green.Fprintln(os.Stdout, fmt.Sprintf("Task [%-"+strconv.Itoa(max)+"s] %s", run.task.Name, run.task.Desc))
		w, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			panic(err)
		}
		blue.Println(strings.Repeat("*", w))
		var (
			wg      sync.WaitGroup
			writers []io.WriteCloser
		)
		for _, r := range run.runners {
			job := run.GenerateRunnerJob()
			if err := r.Run(job, run.input); err != nil {
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
			writers = append(writers, r.Stdin())
		}
		// copy input of task to remote computer's stdin
		if run.input != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w := multiwritecloser.NewMultiWriteCloser(writers...)
				defer w.Close()
				_, err := io.Copy(w, run.input)
				if err != nil {
					fmt.Fprintln(os.Stderr, fmt.Errorf("copy data to remote stdin failed:%w", err))
				}
			}()
		}
		wg.Wait()
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		go ops.RelaySignals(run.runners, signals)
		for _, c := range run.runners {
			if err := c.Wait(); err != nil {
				fmt.Fprintln(os.Stdout, c.Host(), red.SprintFunc()("failed:"+err.Error()))
			} else {
				fmt.Fprintln(os.Stdout, c.Host(), green.SprintFunc()("done"))
			}
		}
		signal.Stop(signals)
		close(signals)
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
				return errors.New("send signal to runner failed")
			}
		}

	}
}
