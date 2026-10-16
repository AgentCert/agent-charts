package main

import (
	"os"
	"reflect"
	"testing"
)

func TestWriteValuesJSONWritesTheDocumentForHelm(t *testing.T) {
	doc := `{"agent":{"config":{"SCAN_INTERVAL":"90","GOAL":"a, b: c"}}}`
	path, err := writeValuesJSON(doc)
	if err != nil {
		t.Fatalf("writeValuesJSON() error = %v", err)
	}
	defer os.Remove(path)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Written verbatim: commas and colons need no --set escaping in a file.
	if string(got) != doc {
		t.Fatalf("file holds %q, want %q", got, doc)
	}
}

func TestWriteValuesJSONRejectsAnythingButAnObject(t *testing.T) {
	for _, doc := range []string{`[]`, `null`, `"x"`, `{"agent":`, `not json`} {
		if path, err := writeValuesJSON(doc); err == nil {
			os.Remove(path)
			t.Errorf("writeValuesJSON(%q) succeeded, want an error", doc)
		}
	}
}

func TestValuesJSONKeysListsLeafPaths(t *testing.T) {
	values := map[string]interface{}{
		"agent":    map[string]interface{}{"config": map[string]interface{}{"SCAN_INTERVAL": "90", "LOG_LEVEL": "DEBUG"}},
		"replicas": 2.0,
	}
	want := []string{"agent.config.LOG_LEVEL", "agent.config.SCAN_INTERVAL", "replicas"}
	if got := valuesJSONKeys(values); !reflect.DeepEqual(got, want) {
		t.Fatalf("valuesJSONKeys() = %v, want %v", got, want)
	}
}

func TestHelmValuesArgsAppliesUserSettingsBeforePlatformValues(t *testing.T) {
	config := &Config{
		ValuesFile:     "/custom/values.yaml",
		ValuesJSONFile: "/tmp/values-json-1.json",
		SetValues:      setFlags{"agent.config.MCP_URLS=http://a:1/mcp,http://b:2/mcp"},
	}
	want := []string{
		"-f", "/custom/values.yaml",
		"-f", "/tmp/values-json-1.json",
		"--set", `agent.config.MCP_URLS=http\://a\:1/mcp\,http\://b\:2/mcp`,
	}
	if got := helmValuesArgs(config); !reflect.DeepEqual(got, want) {
		t.Fatalf("helmValuesArgs() = %v, want %v", got, want)
	}
	if got := helmValuesArgs(&Config{}); len(got) != 0 {
		t.Fatalf("helmValuesArgs() with no values = %v, want none", got)
	}
}
