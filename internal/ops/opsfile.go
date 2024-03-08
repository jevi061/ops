package ops

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Opsfile struct {
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
	return nil
}

type Tasks struct {
	Names map[string]*Task
}

func (t *Tasks) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%d:%d require mappings for tasks", node.Line, node.Column)
	}
	var tasks map[string]*Task
	if err := node.Decode(&tasks); err != nil {
		return err
	}
	t.Names = tasks
	for k, v := range t.Names {
		if v != nil {
			v.Cmd = strings.TrimSpace(v.Cmd)
			v.Name = k
		}
	}
	return nil
}

type Task struct {
	Name  string            `yaml:"name"`
	Cmd   string            `yaml:"cmd"`
	Desc  string            `yaml:"desc"`
	Local bool              `yaml:"local"`
	Envs  map[string]string `yaml:"environments"`
	Deps  []string          `yaml:"deps"`
}

type Environments struct {
	Envs map[string]string
}

func (e *Environments) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%d:%d require mappings for environments", node.Line, node.Column)
	}
	var envs map[string]string
	if err := node.Decode(&envs); err != nil {
		return err
	}
	e.Envs = envs
	return nil
}
func NewOpsfile(data []byte) (*Opsfile, error) {
	var file Opsfile
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
