package ops

import (
	"bufio"
	"bytes"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/gookit/color"
	"github.com/jevi061/ops/internal/connector"
	"github.com/jevi061/ops/internal/prefixer"
	"github.com/jevi061/ops/internal/termsize"
	"github.com/mattn/go-runewidth"
)

type cliExecutor struct {
	conf          *Opsfile
	debug         bool
	dryRun        bool
	alwaysConfirm bool
}

var (
	gray, bold = color.Gray.Render, color.Bold.Render
	green, red = color.Green.Render, color.Red.Render
)

func NewExecutor(conf *Opsfile, debug bool, dryRun bool, alwaysConfirm bool) *cliExecutor {
	return &cliExecutor{conf: conf, debug: debug, dryRun: dryRun, alwaysConfirm: alwaysConfirm}
}
func (e *cliExecutor) Execute(tasks []connector.Task, connectors []connector.Connector) error {
	printer := newExecPrinter(tasks, connectors)
	hasRemoteTask := e.hasRemoteTask(tasks)
	// connect
	for _, c := range connectors {
		remoteTaskWithoutConnectors := (!c.Local() && !hasRemoteTask)
		if !remoteTaskWithoutConnectors {
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
	if e.debug || e.dryRun {
		e.AlignAndColorConnectorPromets(connectors)
	}
	// execute tasks through connectors

	sp := spinner.New(spinner.CharSets[11], 100*time.Millisecond, spinner.WithHiddenCursor(true), spinner.WithFinalMSG(""))
	for _, t := range tasks {
		for _, c := range connectors {
			if t.Local() == c.Local() {
				printer.PrintTaskHeader(t, '·')
				//fmt.Printf("run task: [%s] on connector: [%s]\n", t.Name(), c.Host())
				if !e.debug && !e.dryRun {
					shouldConfirm := t.Prompt() != "" && !e.alwaysConfirm
					if shouldConfirm {
						if !askForConfirmation(t.Prompt()) {
							return nil
						}
					}
					sp.Start()
				}
				startAt := time.Now()
				if err := c.Run(t, &connector.RunOptions{Debug: e.debug, DryRun: e.dryRun}); err != nil {
					if e.conf.FailFast {
						return err
					} else {
						fmt.Fprintln(os.Stderr, red(err.Error()))
					}
				}
				if !e.dryRun {
					output := e.HandleInputAndOutput(t, c)
					sp.Stop()
					err := c.Wait()
					printer.PrintTaskStatus(startAt, c.Host(), t, err, output)
					if err != nil && e.conf.FailFast {
						return err
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

func (e *cliExecutor) HandleInputAndOutput(task connector.Task, c connector.Connector) string {
	var wg sync.WaitGroup
	var (
		errOutput bytes.Buffer
		outOutput bytes.Buffer
	)
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
		wg.Add(2)
		go func(rn connector.Connector) {
			defer wg.Done()
			if _, err := io.Copy(&outOutput, rn.Stdout()); err != nil {
				fmt.Println("copy stdout error:", err)
			}
		}(c)
		// discard stderr
		go func(rn connector.Connector) {
			defer wg.Done()
			if _, err := io.Copy(&errOutput, rn.Stderr()); err != nil {
				fmt.Println("copy stderr error:", err)
			}
		}(c)
	}
	if task.Stdin() != nil {
		stdin, err := task.Stdin()()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return outOutput.String() + errOutput.String()
		}
		wg.Add(1)
		go func(rn connector.Connector) {
			defer wg.Done()
			defer rn.Stdin().Close()

			io.Copy(rn.Stdin(), stdin)
		}(c)
	}
	wg.Wait()
	return outOutput.String() + errOutput.String()
}

func (e *cliExecutor) AlignAndColorConnectorPromets(connectors []connector.Connector) {
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

func (p *execPrinter) PrintTaskHeader(t connector.Task, divider byte) {
	fmt.Println()
	name := t.Name()
	title := fmt.Sprintf("%s [%s] %s", bold("Task:"), bold(name), t.Desc())
	titleLen := runewidth.StringWidth(title)
	w, _ := termsize.DefaultSize(10, 0)
	suffix := ""
	suffixLen := w - titleLen
	if suffixLen > 0 {
		suffix = strings.Repeat(gray(string(divider)), suffixLen)
	}
	fmt.Printf("%s %s\n", title, suffix)
}

func (p *execPrinter) PrintTaskStatus(startAt time.Time, host string, t connector.Task, err error, output string) {
	dura := time.Since(startAt)
	serverHost := host
	w := runewidth.StringWidth(serverHost)
	if w < p.maxConnHostLength {
		serverHost = serverHost + strings.Repeat(" ", p.maxConnHostLength-w)
	}
	if err != nil {
		fmt.Printf("Server: %s    Status: %s    Time: %s    Reason: %s\n", serverHost, red("Failure"), dura, red(err.Error()))
		fmt.Println(red(output))
	} else {
		fmt.Printf("Server: %s    Status: %s    Time: %s\n", serverHost, green("Success"), dura)
	}
}

type execPrinter struct {
	tasks             []connector.Task
	connectors        []connector.Connector
	maxTaskNameLength int
	maxConnHostLength int
}

func newExecPrinter(tasks []connector.Task, connectors []connector.Connector) *execPrinter {
	maxTaskNameLength, maxConnHostLength := 0, 0
	for _, task := range tasks {
		w := runewidth.StringWidth(task.Name())
		if w > maxTaskNameLength {
			maxTaskNameLength = w
		}
	}
	for _, conn := range connectors {
		w := runewidth.StringWidth(conn.Host())
		if w > maxConnHostLength {
			maxConnHostLength = w
		}
	}
	return &execPrinter{tasks: tasks, connectors: connectors, maxTaskNameLength: maxTaskNameLength, maxConnHostLength: maxConnHostLength}
}

// askForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will consider it as no.
func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/n]: ", s)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "y" || response == "yes" {
		return true
	}
	return false

}
