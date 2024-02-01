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

// Parse string argemnts like "key=value" to variables map
func parseArgs(args []string) map[string]string {
	vars := make(map[string]string, len(args))
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}

	return vars
}

// Set custom variables to global map
func SetArgv(vars map[string]string) {
	argv = vars
}

// Wrapper for original yaml.Unmarshal with custom tag processor
func Unmarshal(in []byte, out interface{}) error {
	return yaml.Unmarshal(in, &CustomTagProcessor{out})
}

func Load(r io.Reader, v interface{}, args []string) error {
	SetArgv(parseArgs(args))
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, r)
	err := Unmarshal(buf.Bytes(), v)
	return err
}

func init() {
	AddResolvers("!include", resolveIncludes)
	AddResolvers("!env", resolveGetFromEnv)
	AddResolvers("!var", resolveGetFromVars)
}
