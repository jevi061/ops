package ops

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Opsfile struct {
	Servers      *Servers      `yaml:"servers"`
	Tasks        *Tasks        `yaml:"tasks"`
	Pipelines    *Pipelines    `yaml:"pipelines"`
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

	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("%d:%d require sequence node for servers", node.Line, node.Column)
	}
	var servers []*Server
	if err := node.Decode(&servers); err != nil {
		return err
	}
	c.Names = make(map[string]*Server)
	for _, v := range servers {
		if v != nil {
			c.Names[v.Host] = v
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
	p.Names = make(map[string][]string, 0)
	var pipes map[string][]string
	if err := node.Decode(&pipes); err != nil {
		return err
	}
	if len(pipes) > 0 {
		p.Names = pipes
	}
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
	if file.Pipelines == nil {
		file.Pipelines = &Pipelines{Names: map[string][]string{}}
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
