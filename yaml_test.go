package yaml

import (
	"bytes"
	"testing"
)

func TestResolveIncludes(t *testing.T) {
	yamlContent := []byte("incData: !include testdata/included.yaml")
	var f Fragment
	err := Unmarshal(yamlContent, &f)
	if err != nil {
		t.Fatalf("Error unmarshaling YAML: %v", err)
	}
}

func TestResolveGetFromEnv(t *testing.T) {
	t.Setenv("SOME_ENV_VAR", "SOME_ENV_VAL")
	yamlContent := []byte("envData: !env SOME_ENV_VAR")
	var f Fragment
	err := Unmarshal(yamlContent, &f)
	if err != nil {
		t.Fatalf("Error unmarshaling YAML: %v", err)
	}
}

func TestResolveGetFromVars(t *testing.T) {
	yamlContent := []byte("varData: !var some_variable")
	SetArgv(map[string]string{"some_variable": "some_value"})
	var f Fragment
	err := Unmarshal(yamlContent, &f)
	if err != nil {
		t.Fatalf("Error unmarshaling YAML: %v", err)
	}
}

func TestLoadWithCustomTags(t *testing.T) {
	t.Setenv("SOME_ENV_VAR", "SOME_ENV_VAL")

	yamlContent := []byte(`
incData: !include testdata/included.yaml
envData: !env SOME_ENV_VAR
varData: !var some_variable
`)

	var result struct {
		CustomData Fragment
		EnvData    Fragment
		VarData    Fragment
	}

	err := Load(bytes.NewReader(yamlContent), &result, []string{"some_variable=some_value"})
	if err != nil {
		t.Fatalf("Error loading YAML: %v", err)
	}
}
