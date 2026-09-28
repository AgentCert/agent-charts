package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestReleaseDeploymentNames(t *testing.T) {
	list := []byte(`{"items":[
	  {"metadata":{"name":"agent-release","annotations":{"meta.helm.sh/release-name":"flash-agent"}}},
	  {"metadata":{"name":"front-end","annotations":{"meta.helm.sh/release-name":"sock-shop"}}},
	  {"metadata":{"name":"sidecar-release","annotations":{"meta.helm.sh/release-name":"flash-agent"}}},
	  {"metadata":{"name":"unmanaged"}}
	]}`)

	got, err := releaseDeploymentNames(list, "flash-agent")
	if err != nil {
		t.Fatalf("releaseDeploymentNames() error = %v", err)
	}
	want := []string{"agent-release", "sidecar-release"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("releaseDeploymentNames() = %#v, want %#v", got, want)
	}
}

func TestReleaseDeploymentNamesRejectsMalformedList(t *testing.T) {
	if _, err := releaseDeploymentNames([]byte("kind: Deployment"), "flash-agent"); err == nil {
		t.Fatal("releaseDeploymentNames() accepted a non-JSON list")
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
