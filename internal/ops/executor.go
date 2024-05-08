package ops

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/containerd/console"
	"github.com/gookit/color"
	"github.com/jevi061/ops/internal/connector"
	"github.com/jevi061/ops/internal/prefixer"
	"github.com/mattn/go-runewidth"
)

type cliExecutor struct {
	conf   *Opsfile
	debug  bool
	dryRun bool
}

func NewExecutor(conf *Opsfile, debug bool, dryRun bool) *cliExecutor {
	return &cliExecutor{conf: conf, debug: debug, dryRun: dryRun}
}
func (e *cliExecutor) Execute(tasks []connector.Task, connectors []connector.Connector) error {
	hasRemoteTask := e.hasRemoteTask(tasks)
	// setup running modes and connect
	for _, c := range connectors {
		if !(!c.Local() && !hasRemoteTask) {
			c.SetDebug(e.debug)
			if err := c.Connect(); err != nil {
				return err
			} else {
				defer c.Close()
			}
		}
	}
	// relay signals to runners
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go func() {
		if err := e.RelaySignals(connectors, signals); err != nil {
			fmt.Fprintln(os.Stderr, "RUN ERROR:", err)
			os.Exit(1)
		}
	}()
	defer func() {
		signal.Stop(signals)
		close(signals)
	}()
	// update prompets
	if e.debug {
		e.AlignAndColorTaskRunnersPromets(connectors)
	}
	// execute tasks through connectors
	gray, bold := color.Gray.Render, color.Bold.Render
	green, red := color.Green.Render, color.Red.Render

	for _, t := range tasks {
		for _, c := range connectors {
			if t.Local() == c.Local() {
				fmt.Printf("%s [%s] %s\n", bold("Task:"), bold(t.Name()), gray(t.Desc()))
				e.PrintDivider('-')
				if !e.dryRun {
					//fmt.Printf("run task: [%s] on connector: [%s]\n", t.Name(), c.Host())
					if err := c.Run(t); err != nil {
						if e.conf.FailFast {
							return err
						} else {
							fmt.Fprintln(os.Stderr, color.Red.Render(err.Error()))
						}
					}
					e.HandleInputAndOutput(t, c)
					if err := c.Wait(); err != nil {
						fmt.Printf("Server: [%s] Status: %s Reason: %s\n", c.Host(), red("Error"), red(err.Error()))
						if e.conf.FailFast {
							return err
						}
					} else {
						fmt.Printf("Server: [%s] Status: %s\n", c.Host(), green("Success"))
					}
				}
			}
		}
	}

	return nil
}

// RelaySignals realy incoming signals to avaliable runners, it will block until signals chan closed
func (e *cliExecutor) RelaySignals(runners []connector.Connector, signals chan os.Signal) error {
	for {
		sig, ok := <-signals
		if !ok {
			return nil
		}
		for _, r := range runners {
			err := r.Signal(sig)
			if err != nil {
				return fmt.Errorf("send signal to connector: [%s] failed: %w", r.Host(), err)
			}
		}
	}
}

func (e *cliExecutor) HandleInputAndOutput(task connector.Task, c connector.Connector) error {
	var wg sync.WaitGroup
	if e.debug {
		// copy remote computer's stdout to current
		wg.Add(1)
		go func(rn connector.Connector) {
			defer wg.Done()
			_, err := io.Copy(os.Stdout, prefixer.NewPrefixReader(rn.Stdout(), rn.Promet()))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}(c)
		// copy remote computer's stderr to current
		wg.Add(1)
		go func(rn connector.Connector) {
			defer wg.Done()
			_, err := io.Copy(os.Stderr, prefixer.NewPrefixReader(rn.Stderr(), rn.Promet()))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}(c)
	} else {
		// discard stdout
		wg.Add(1)
		go func(rn connector.Connector) {
			defer wg.Done()
			io.Copy(io.Discard, rn.Stdout())
		}(c)
		// discard stderr
		wg.Add(1)
		go func(rn connector.Connector) {
			defer wg.Done()
			io.Copy(io.Discard, rn.Stderr())
		}(c)
	}
	if task.Stdin() != nil {
		stdin, err := task.Stdin()()
		if err != nil {
			return err
		}
		wg.Add(1)
		go func(rn connector.Connector) {
			defer wg.Done()
			defer rn.Stdin().Close()

			io.Copy(rn.Stdin(), stdin)
		}(c)
	}
	wg.Wait()
	return nil
}

func (e *cliExecutor) AlignAndColorTaskRunnersPromets(connectors []connector.Connector) {
	// align and color connector promets
	colors := []func(a ...interface{}) string{color.Yellow.Render, color.Cyan.Render,
		color.Magenta.Render, color.Blue.Render}
	max := 0

	for _, r := range connectors {
		prefixLen := runewidth.StringWidth(r.Promet())
		if prefixLen > max {
			max = prefixLen
		}
	}
	//fmt.Println("max promets len:", max)
	hash := fnv.New32a()

	for _, r := range connectors {
		prefixLen := runewidth.StringWidth(r.Promet())
		if prefixLen <= max { // Left padding.
			p := strings.Repeat(" ", max-prefixLen) + r.Promet()
			hash.Write([]byte(p))
			color := colors[int(hash.Sum32())%len(colors)]
			r.SetPromet(color(p))
			//fmt.Println("update runner:", r.Host(), "promets with:", max-prefixLen, "blanks")
		}
	}

}

func (e *cliExecutor) hasRemoteTask(tasks []connector.Task) bool {
	for _, t := range tasks {
		if !t.Local() {
			return true
		}
	}
	return false
}

func (e *cliExecutor) PrintDivider(divider byte) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(strings.Repeat(string(divider), 10))
		}
	}()
	current := console.Current()
	if ws, err := current.Size(); err != nil {
		fmt.Println(strings.Repeat(string(divider), 10))
	} else {
		fmt.Println(strings.Repeat(string(divider), int(ws.Width)))
	}
}
