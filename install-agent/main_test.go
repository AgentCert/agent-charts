package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestDeploymentNamesFromManifest(t *testing.T) {
	manifest := []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: "agent-release"
spec:
  template:
    metadata:
      name: must-not-replace-object-name
---
apiVersion: v1
kind: Service
metadata:
  name: agent-service
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sidecar-release
`)

	got := deploymentNamesFromManifest(manifest)
	want := []string{"agent-release", "sidecar-release"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deploymentNamesFromManifest() = %#v, want %#v", got, want)
	}
}

func TestDeploymentNamesFromManifestRejectsNestedKind(t *testing.T) {
	manifest := []byte(`apiVersion: batch/v1
kind: Job
metadata:
  name: installer
spec:
  template:
    spec:
      containers:
        - name: worker
          env:
            - name: kind
              value: Deployment
`)

	if got := deploymentNamesFromManifest(manifest); len(got) != 0 {
		t.Fatalf("deploymentNamesFromManifest() = %#v, want no deployments", got)
	}
}

func TestFormatHelmArgsRedactsSensitiveSetValues(t *testing.T) {
	args := []string{
		"upgrade", "agent",
		"--set", "agent.secret.OPENAI_API_KEY=top-secret",
		"--set-string=credentials.access-token=token-value",
		"--set", "agent.config.MODEL_ALIAS=qwen",
	}
	got := formatHelmArgs(args)
	if strings.Contains(got, "top-secret") || strings.Contains(got, "token-value") {
		t.Fatalf("formatHelmArgs leaked a secret: %s", got)
	}
	if !strings.Contains(got, "OPENAI_API_KEY=<redacted>") ||
		!strings.Contains(got, "access-token=<redacted>") ||
		!strings.Contains(got, "MODEL_ALIAS=qwen") {
		t.Fatalf("formatHelmArgs redaction is incorrect: %s", got)
	}
	if args[3] != "agent.secret.OPENAI_API_KEY=top-secret" {
		t.Fatal("formatHelmArgs mutated command arguments")
	}
}
