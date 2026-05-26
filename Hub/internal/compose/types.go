package compose

import (
	"gopkg.in/yaml.v3"
)

type ComposeFile struct {
	Services OrderedServices `yaml:"services"`
}

type OrderedServices []NamedService

type NamedService struct {
	Name    string
	Service *ComposeService
}

func (os OrderedServices) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	for _, s := range os {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: s.Name,
		}

		valNode := &yaml.Node{}
		if err := valNode.Encode(s.Service); err != nil {
			return nil, err
		}

		node.Content = append(node.Content, keyNode, valNode)
	}

	return node, nil
}

type ComposeService struct {
	Image         string               `yaml:"image"`
	ContainerName string               `yaml:"container_name,omitempty"`
	Command       string               `yaml:"command,omitempty"`
	Restart       string               `yaml:"restart,omitempty"`
	NetworkMode   string               `yaml:"network_mode,omitempty"`
	DependsOn     map[string]DependsOn `yaml:"depends_on,omitempty"`

	// Sandbox
	User        string   `yaml:"user,omitempty"`
	ReadOnly    bool     `yaml:"read_only,omitempty"`
	CapDrop     []string `yaml:"cap_drop,omitempty"`
	CapAdd      []string `yaml:"cap_add,omitempty"`
	SecurityOpt []string `yaml:"security_opt,omitempty"`

	Logging     *LoggingConfig  `yaml:"logging,omitempty"`
	Environment []string        `yaml:"environment,omitempty"`
	Ports       []string        `yaml:"ports,omitempty"`
	Volumes     []ComposeVolume `yaml:"volumes,omitempty"`
}

type DependsOn struct {
	Condition string `yaml:"condition"`
}

type ComposeVolume struct {
	Type     string `yaml:"type"`
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"read_only,omitempty"`
}

type LoggingConfig struct {
	Driver  string            `yaml:"driver"`
	Options map[string]string `yaml:"options"`
}
