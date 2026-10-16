package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
)

// -values-json carries the chart settings a user chose in the ACE experiment
// builder, as one JSON document shaped like a Helm values file. The GraphQL
// server validates it against the settings the chart declares before the
// workflow runs; the installer only checks that it is a JSON object, writes
// it to a file and hands that to Helm.

// writeValuesJSON writes a -values-json document to a temporary values file
// and returns its path. JSON is a subset of YAML, so Helm reads it as is.
func writeValuesJSON(doc string) (string, error) {
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(doc), &values); err != nil || values == nil {
		return "", fmt.Errorf("-values-json must be a JSON object: %v", err)
	}

	file, err := os.CreateTemp("", "values-json-*.json")
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.WriteString(doc); err != nil {
		os.Remove(file.Name())
		return "", err
	}
	log.Printf("Applying chart settings from the experiment builder: %v", valuesJSONKeys(values))
	return file.Name(), nil
}

// valuesJSONKeys lists the dotted paths a values document sets, for the log.
// Values are left out: the installer cannot tell which of them are sensitive.
func valuesJSONKeys(values map[string]interface{}) []string {
	var keys []string
	var walk func(prefix string, node map[string]interface{})
	walk = func(prefix string, node map[string]interface{}) {
		for name, value := range node {
			key := prefix + name
			if child, ok := value.(map[string]interface{}); ok {
				walk(key+".", child)
				continue
			}
			keys = append(keys, key)
		}
	}
	walk("", values)
	sort.Strings(keys)
	return keys
}

// helmValuesArgs returns the values flags shared by `helm template` (resource
// adoption) and the install itself, in Helm's precedence order: chart default
// < -values < -values-json < --set. The platform's own values come last, as
// --set, so a user setting can never override them.
func helmValuesArgs(config *Config) []string {
	var args []string
	if config.ValuesFile != "" {
		args = append(args, "-f", config.ValuesFile)
	}
	if config.ValuesJSONFile != "" {
		args = append(args, "-f", config.ValuesJSONFile)
	}
	for _, setValue := range config.SetValues {
		args = append(args, "--set", escapeHelmSetValue(setValue))
	}
	return args
}
