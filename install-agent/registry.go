package main

// Private-registry support for chart installs (docs/setup/registry-migration-plan.md,
// Phase 5). graphql sets ACE_IMAGE_REGISTRY / ACE_IMAGE_MIRROR_NAMESPACE /
// ACE_IMAGE_PULL_SECRET on this step only when the chart's images are pulled
// from the registry at run time; locally side-loaded images keep public names.
// Identical in app-charts/install-app and agent-charts/install-agent.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// runPostRenderIfRequested makes this binary a Helm post-renderer when helm
// invokes it with ACE_HELM_POST_RENDER=1: manifests on stdin, rewritten
// manifests on stdout. Returns false when not in post-render mode.
func runPostRenderIfRequested() bool {
	if os.Getenv(PostRenderEnv) != "1" {
		return false
	}
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "post-render: reading manifests:", err)
		os.Exit(1)
	}
	fmt.Print(PostRender(string(in)))
	return true
}

// postRendererArgs returns the helm flags and environment that run this binary
// as the image post-renderer, or nil when graphql did not request it.
func postRendererArgs() (args, env []string) {
	if os.Getenv("ACE_IMAGE_REGISTRY") == "" && os.Getenv("ACE_IMAGE_MIRROR_NAMESPACE") == "" && os.Getenv("ACE_IMAGE_TAG") == "" {
		return nil, nil
	}
	self, err := os.Executable()
	if err != nil {
		log.Printf("Warning: cannot locate own binary for the image post-renderer: %v", err)
		return nil, nil
	}
	return []string{"--post-renderer", self}, []string{PostRenderEnv + "=1"}
}

// ensurePullSecret copies the image-pull Secret (ACE_IMAGE_PULL_SECRET) from
// this pod's namespace into the target namespace before helm runs. The
// namespace may have been created a moment ago, before registry-secret-sync
// could reach it, and pods admitted without the secret would fail with 401.
func ensurePullSecret(targetNS string) {
	name := strings.TrimSpace(os.Getenv("ACE_IMAGE_PULL_SECRET"))
	if name == "" {
		return
	}
	srcNS := ownNamespace()
	if srcNS == targetNS {
		return
	}
	out, err := exec.Command("kubectl", "get", "secret", name, "-n", srcNS, "-o", "json").Output()
	if err != nil {
		log.Printf("Warning: pull secret %s/%s not found (%v); relying on registry-secret-sync", srcNS, name, err)
		return
	}
	var src struct {
		Type string            `json:"type"`
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(out, &src); err != nil {
		log.Printf("Warning: cannot parse pull secret %s/%s: %v", srcNS, name, err)
		return
	}
	copied, _ := json.Marshal(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata":   map[string]string{"name": name, "namespace": targetNS},
		"type":       src.Type,
		"data":       src.Data,
	})
	apply := exec.Command("kubectl", "apply", "-f", "-")
	apply.Stdin = strings.NewReader(string(copied))
	if msg, err := apply.CombinedOutput(); err != nil {
		log.Printf("Warning: could not copy pull secret %s into %s: %v (%s)", name, targetNS, err, strings.TrimSpace(string(msg)))
		return
	}
	log.Printf("Pull secret %s copied from %s into %s", name, srcNS, targetNS)
}

// ownNamespace is the namespace this pod runs in (the chaos infra namespace).
func ownNamespace() string {
	if b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(b)); ns != "" {
			return ns
		}
	}
	if ns := strings.TrimSpace(os.Getenv("POD_NAMESPACE")); ns != "" {
		return ns
	}
	return "litmus"
}
