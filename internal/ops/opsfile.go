package ops

import (
	"fmt"
	"os"
	"strings"

	"github.com/jevi061/ops/internal/transfer"
	"gopkg.in/yaml.v3"
)

type Opsfile struct {
	Version      string        `yaml:"version"`
	Shell        string        `yaml:"shell"`
	FailFast     bool          `yaml:"fail-fast"`
	Servers      *Servers      `yaml:"servers"`
	Tasks        *Tasks        `yaml:"tasks"`
	Environments *Environments `yaml:"environments"`
}
type Servers struct {
	Names map[string]*Server
}
type Server struct {
	Host     string   `yaml:"host"`
	Port     uint     `yaml:"port"`
	User     string   `yaml:"user"`
	Password string   `yaml:"password"`
	Tags     []string `yaml:"tags"`
}

func (c *Servers) UnmarshalYAML(node *yaml.Node) error {

	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("yaml: line: %d require sequence node for servers", node.Line)
	}
	c.Names = make(map[string]*Server, 0)
	if err := node.Decode(&c.Names); err != nil {
		return err
	}
	for _, s := range c.Names {
		s.Password = strings.TrimSpace(s.Password)
	}
	return nil
}

type Tasks struct {
	Names map[string]*Task
}

func (t *Tasks) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%d:%d require mappings for tasks", node.Line, node.Column)
	}
	tasks := make(map[string]*Task, 0)
	if err := node.Decode(&tasks); err != nil {
		return err
	}
	t.Names = tasks
	// setup task name
	for k, v := range t.Names {
		if v != nil {
			v.Name = k
		}
	}
	// validate transfer
	for k, v := range t.Names {
		if v.Cmd != "" && v.Transfer != "" {
			return fmt.Errorf("task: %s defined with command and transfer simultaneously", k)
		}
		if v.Transfer != "" {
			if err := transfer.Validate(v.Transfer); err != nil {
				return fmt.Errorf("invalid task: %s : %w", k, err)
			}
		}
	}
	return nil
}

type Task struct {
	Name     string            `yaml:"name"`
	Cmd      string            `yaml:"command"`
	Transfer string            `yaml:"transfer"`
	Desc     string            `yaml:"desc"`
	Local    bool              `yaml:"local"`
	Sudo     bool              `yaml:"sudo"`
	Envs     map[string]string `yaml:"environments"`
	Deps     []string          `yaml:"dependencies"`
}

type Environments struct {
	Envs map[string]string
}

func (e *Environments) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%d:%d require mappings for environments", node.Line, node.Column)
	}
	envs := make(map[string]string, 0)
	if err := node.Decode(&envs); err != nil {
		return err
	}
	e.Envs = envs
	return nil
}
func NewOpsfile(data []byte) (*Opsfile, error) {
	var file Opsfile
	// setup default values
	file.Shell = "bash"
	file.FailFast = true
	file.Environments = &Environments{Envs: make(map[string]string, 0)}
	file.Tasks = &Tasks{Names: make(map[string]*Task, 0)}
	file.Servers = &Servers{Names: make(map[string]*Server, 0)}
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return &file, nil
}

func NewOpsfileFromPath(path string) (*Opsfile, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(path); err != nil {
		return nil, err
	} else {
		conf, err := NewOpsfile(data)
		return conf, err
	}
}
