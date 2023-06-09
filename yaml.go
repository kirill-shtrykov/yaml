package yaml

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var tagResolvers = make(map[string]func(*yaml.Node) (*yaml.Node, error))
var argv = make(map[string]string)

type Decoder yaml.Decoder

type Fragment struct {
	content *yaml.Node
}

func (f *Fragment) UnmarshalYAML(value *yaml.Node) error {
	var err error
	// process includes in fragments
	f.content, err = resolveTags(value)
	return err
}

type CustomTagProcessor struct {
	target interface{}
}

func (i *CustomTagProcessor) UnmarshalYAML(value *yaml.Node) error {
	resolved, err := resolveTags(value)
	if err != nil {
		return err
	}
	return resolved.Decode(i.target)
}

func resolveTags(node *yaml.Node) (*yaml.Node, error) {
	for tag, fn := range tagResolvers {
		if node.Tag == tag {
			return fn(node)
		}
	}
	if node.Kind == yaml.SequenceNode || node.Kind == yaml.MappingNode {
		var err error
		for i := range node.Content {
			node.Content[i], err = resolveTags(node.Content[i])
			if err != nil {
				return nil, err
			}
		}
	}
	return node, nil
}

func resolveIncludes(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind != yaml.ScalarNode {
		return nil, errors.New("!include on a non-scalar node")
	}
	file, err := os.ReadFile(node.Value)
	if err != nil {
		return nil, err
	}
	var f Fragment
	err = yaml.Unmarshal(file, &f)
	return f.content, err
}

func resolveGetFromEnv(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind != yaml.ScalarNode {
		return nil, errors.New("!env on a non-scalar node")
	}
	value := os.Getenv(node.Value)
	if value == "" {
		return nil, fmt.Errorf("environment variable %v not set", node.Value)
	}
	var f Fragment
	err := yaml.Unmarshal([]byte(value), &f)
	return f.content, err
}

func resolveGetFromVars(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind != yaml.ScalarNode {
		return nil, errors.New("!var on a non-scalar node")
	}
	value, ok := argv[node.Value]
	if !ok {
		return nil, fmt.Errorf("variable %v not set", node.Value)
	}
	var f Fragment
	err := yaml.Unmarshal([]byte(value), &f)
	return f.content, err
}

func AddResolvers(tag string, fn func(*yaml.Node) (*yaml.Node, error)) {
	tagResolvers[tag] = fn
}

func toVars(vars []string) map[string]string {
	attrs := make(map[string]string, len(vars))
	for _, arg := range vars {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			attrs[parts[0]] = parts[1]
		}
	}

	return attrs
}

func Load(r io.Reader, v interface{}, vars []string) error {
	argv = toVars(vars)
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, r)
	err := yaml.Unmarshal(buf.Bytes(), &CustomTagProcessor{v})
	return err
}

func init() {
	AddResolvers("!include", resolveIncludes)
	AddResolvers("!env", resolveGetFromEnv)
	AddResolvers("!var", resolveGetFromVars)
}
