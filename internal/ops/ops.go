package ops

type Ops struct {
	conf          *Opsfile
	debug         bool
	dryRun        bool
	alwaysConfirm bool
}

type OpsOption func(*Ops)

func WithDebug(debug bool) OpsOption {
	return func(o *Ops) {
		o.debug = debug
	}
}
func WithDryRun(dryRun bool) OpsOption {
	return func(o *Ops) {
		o.dryRun = dryRun
	}
}
func WithAlwaysConfirm(alwaysConfirm bool) OpsOption {
	return func(o *Ops) {
		o.alwaysConfirm = alwaysConfirm
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

// Run
func (ops *Ops) Run(serverTag string, tasks ...string) error {
	cp := &connectorPreparer{}
	connectors := cp.Prepare(ops.conf, serverTag)

	ctp := &connectorTaskPreparer{}
	connectorTasks, err := ctp.Prepare(ops.conf, tasks...)
	if err != nil {
		return err
	}
	exec := NewExecutor(ops.conf, ops.debug, ops.dryRun, ops.alwaysConfirm)
	if err := exec.Execute(connectorTasks, connectors); err != nil {
		return err
	}
	return nil
}
