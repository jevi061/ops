package connector

import (
	"io"
)

// Task represents executable/runnable task through connector
type Task interface {
	// Shell defines environment for command to run. Currently, only sh and bash are supported.
	Shell() string
	// Shell command or scripts of task
	Commands() []string
	Environments() map[string]string
	Stdin() func() (io.Reader, error)
	Sudo() bool
	Local() bool
	Name() string
	Desc() string
	Prompt() string
}

// CommonTask is minimum unit of task with target runners for ops to run
type CommonTask struct {
	shell    string
	commands []string
	envs     map[string]string
	stdin    func() (io.Reader, error) // input generator
	sudo     bool
	local    bool
	name     string
	desc     string
	prompt   string // task prompt
}

func NewCommonTask(options ...func(*CommonTask)) *CommonTask {
	ct := &CommonTask{}
	for _, option := range options {
		option(ct)
	}
	return ct
}
func WithShell(shell string) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.shell = shell
	}
}
func WithCommand(command ...string) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.commands = append(ct.commands, command...)
	}
}
func WithEnvironments(envs map[string]string) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.envs = envs
	}
}
func WithSudo(sudo bool) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.sudo = sudo
	}
}
func WithName(name string) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.name = name
	}
}
func WithDesc(desc string) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.desc = desc
	}
}
func WithLocal(local bool) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.local = local
	}
}
func WithPrompt(prompt string) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.prompt = prompt
	}
}
func WithStdin(stdin func() (io.Reader, error)) func(*CommonTask) {
	return func(ct *CommonTask) {
		ct.stdin = stdin
	}
}
func (ct *CommonTask) Shell() string {
	return ct.shell
}

// Command return executable sh/bash commands
func (ct *CommonTask) Commands() []string {
	return ct.commands
}
func (ct *CommonTask) Environments() map[string]string {
	return ct.envs
}
func (ct *CommonTask) Stdin() func() (io.Reader, error) {
	return ct.stdin
}
func (ct *CommonTask) Local() bool {
	return ct.local
}

func (ct *CommonTask) Sudo() bool {
	return ct.sudo
}
func (ct *CommonTask) Name() string {
	return ct.name
}
func (ct *CommonTask) Desc() string {
	return ct.desc
}
func (ct *CommonTask) Prompt() string {
	return ct.prompt
}
