package ops

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Opsfile struct {
	Computers    *Computers    `yaml:"computers"`
	Tasks        *Tasks        `yaml:"tasks"`
	Pipelines    *Pipelines    `yaml:"pipelines"`
	Environments *Environments `yaml:"environments"`
}
type Computers struct {
	Names map[string]*Computer
}
type Computer struct {
	Host     string   `yaml:"host"`
	Port     uint     `yaml:"port"`
	User     string   `yaml:"user"`
	Password string   `yaml:"password"`
	Tags     []string `yaml:"tags"`
}

func (c *Computers) UnmarshalYAML(node *yaml.Node) error {

	//fmt.Println("node line:", node.Line, "node kind:", node.Kind, "node value:", node.Value)
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%d:%d require mappings for computer", node.Line, node.Column)
	}
	var hosts map[string]*Computer
	if err := node.Decode(&hosts); err != nil {
		return err
	}
	c.Names = hosts
	for k, v := range c.Names {
		if v == nil {
			c.Names[k] = &Computer{Host: k}
		} else {
			v.Host = k
		}
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
			v.Name = k
		}
	}
	return nil
}

type Task struct {
	Name     string            `yaml:"name"`
	Cmd      string            `yaml:"cmd"`
	Desc     string            `yaml:"desc"`
	LocalCmd string            `yaml:"local-cmd"`
	Upload   *Upload           `yaml:"upload"`
	Download *Upload           `yaml:"download"`
	Envs     map[string]string `yaml:"environments"`
}
type Upload struct {
	Src  string `yaml:"src"`
	Dest string `yaml:"dest"`
}

func (t *Task) AsCmdLine() string {
	if t.Cmd != "" {
		return t.Cmd
	}
	if t.LocalCmd != "" {
		return t.LocalCmd
	}
	return ""
}

type Pipelines struct {
	Names map[string][]string
}

func (p *Pipelines) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%d:%d require mappings for pipelines", node.Line, node.Column)
	}
	var pipes map[string][]string
	if err := node.Decode(&pipes); err != nil {
		return err
	}
	p.Names = pipes
	return nil
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
