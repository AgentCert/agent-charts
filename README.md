<div align="center">

# agent-charts

**Helm charts, installer image, and LiteLLM manifests for the AI agents under test in
the AgentCert platform.**

This repository is the **AgentHub** source consumed by AgentCert ChaosCenter: when an
operator runs a scenario, the in-cluster subscriber pulls a chart from here and installs
the agent (plus its identity-injection sidecar) into the target namespace.

![Helm](https://img.shields.io/badge/Helm-v3.14-0F1689?style=flat-square&logo=helm)
![Kubernetes](https://img.shields.io/badge/Kubernetes-1.29-326CE5?style=flat-square&logo=kubernetes)
![Go](https://img.shields.io/badge/install--agent-Go%201.21-00ADD8?style=flat-square&logo=go)
![License](https://img.shields.io/badge/License-MIT-lightgrey?style=flat-square)

</div>

---

## Table of Contents

- [What's in here](#whats-in-here)
- [Charts](#charts)
  - [`flash-agent`](#flash-agent)
  - [`k8s-agent`](#k8s-agent)
- [The `install-agent` CLI image](#the-install-agent-cli-image)
- [LiteLLM manifests](#litellm-manifests)
- [Installing an agent](#installing-an-agent)
- [How it fits into AgentCert](#how-it-fits-into-agentcert)
- [Versions](#versions)
- [License](#license)

---

## What's in here

```
agent-charts/
├── charts/
│   ├── agents.chartserviceversion.yaml   # AgentCert API: capabilities + context-injection rules
│   ├── flash-agent/                      # Helm chart: FLASH-style ITOps agent
│   │   ├── Chart.yaml                    # v0.1.0 (appVersion 1.0.0)
│   │   ├── values.yaml                   # config, sidecar block, MCP_URLS, image refs
│   │   └── templates/                    # Deployment, ConfigMap, Secret, SA, ClusterRole, RB
│   └── k8s-agent/                        # Helm chart: generic K8s fault-detection agent
│       ├── Chart.yaml                    # v0.1.0 (appVersion 1.0.0)
│       ├── values.yaml
│       ├── k8s-agent-values.yaml         # alternate values for scheduled mode
│       └── templates/                    # Deployment + CronJob + ConfigMap + RBAC
│
├── install-agent/                        # Go CLI + Docker image used by ChaosCenter
│   ├── main.go                           # helm-upgrade + RegisterAgent → re-helm with agentId
│   ├── Dockerfile                        # alpine + helm 3.14 + kubectl 1.29, charts baked in
│   ├── Makefile                          # build / push / tag / kind-load / install
│   ├── build-install-agent.sh            # CI: timestamp tag, minikube load, env sync
│   └── README.md                         # CLI reference
│
├── litellm/                              # Kubernetes manifests for the LLM gateway
│   ├── namespace.yaml
│   ├── secret.yaml                       # AZURE + LANGFUSE keys
│   ├── configmap.yaml                    # litellm config.yaml + custom_callbacks.py
│   └── deployment.yaml                   # litellm/litellm:v1.82.0-stable, port 4000
│
├── LICENSE
└── README.md
```

---

## Charts

### `flash-agent`

LLM-powered ITOps agent that scans a Kubernetes namespace through MCP tools and produces
structured health analyses with hindsight reflection. See
[`flash-agent`](../flash-agent) for the agent source.

| Aspect | Value |
|---|---|
| Chart | `charts/flash-agent` (v0.1.0, appVersion 1.0.0) |
| Agent image | `docker.io/agentcert/agentcert-flash-agent:latest` (Always pull) |
| Sidecar image | `docker.io/agentcert/agent-sidecar:latest` |
| Mode | Single-replica `Deployment` |
| RBAC | `ServiceAccount` + `ClusterRole` (read pods/logs/events/deployments/RS/STS) |

Key values (excerpt from [`charts/flash-agent/values.yaml`](charts/flash-agent/values.yaml)):

```yaml
agent:
  config:
    OPENAI_BASE_URL: http://litellm-proxy.litellm.svc.cluster.local:4000/v1
    MODEL_ALIAS:      gpt-4o
    MCP_URLS:         "http://kubernetes-mcp-server.sock-shop.svc.cluster.local:8081/sse,\
                       http://prometheus-mcp-server.sock-shop.svc.cluster.local:8082/sse"
    MCP_TIMEOUT:      "30"
    SCAN_INTERVAL:    "0"          # 0 = run once, >0 = continuous loop
    AGENT_SCOPE_NAMESPACE: sock-shop

  secrets:
    OPENAI_API_KEY: "sk-agentcert-2026"

sidecar:
  enabled: true
  port:     4001
  upstreamUrl: http://litellm-proxy.litellm.svc.cluster.local:4000
  injectionMode: openai-metadata    # or http-header / none
```

### `k8s-agent`

Generic Kubernetes fault-detection + auto-remediation agent template. Supports both a
continuous `Deployment` (active mode) and a `CronJob` (scheduled mode, default
`*/5 * * * *`). The shipped values use `nginx:latest` as a placeholder image — you supply
your own.

| Aspect | Value |
|---|---|
| Chart | `charts/k8s-agent` (v0.1.0, appVersion 1.0.0) |
| Modes | `Deployment` (active) + `CronJob` (scheduled) |
| RBAC | `ClusterRole` with read + `delete`/`patch` verbs (for remediation) |
| Connects to | `http://chaoscenter.litmus.svc.cluster.local` (Litmus Chaos Center) |
| Default LLM | `gemini-3-flash` via LiteLLM |

---

## The `install-agent` CLI image

Charts in this repo are not consumed directly by AgentCert — the platform calls a
**baked-in Docker image** (`agentcert/agentcert-install-agent`) that bundles the CLI **and
every chart under `charts/`** into a single artifact.

This is the workflow ChaosCenter triggers when a scenario asks for an agent install:

```
ChaosCenter ── runs Argo step ──▶ install-agent container
                                  ├── helm upgrade --install <chart>
                                  ├── GraphQL RegisterAgent(...) ──▶ returns agentId (UUID)
                                  └── helm upgrade --install --set agent.config.AGENT_ID=<uuid>
```

The two-pass install lets agents receive the registry UUID **at runtime** — so emitted
Langfuse traces correlate back to the registry without an out-of-band lookup.

### Building the installer image

```bash
cd install-agent

make build                       # docker build → agentcert/agentcert-install-agent:latest
make push                        # push to registry
make build-push                  # build + push
make tag NEW_TAG=v1.0.0          # retag for release
make kind-load                   # docker save → kind load
make install FOLDER=flash-agent NAMESPACE=test-ns   # run the container against a kubeconfig
```

The Dockerfile is multi-stage — `golang:1.21-alpine3.19` builder, `alpine:3.19` runtime
with `helm v3.14.0` + `kubectl v1.29.0` + a non-root user (UID 1000). Charts are copied
at build time into `/charts/` so the binary can `helm install` without a network fetch.

`build-install-agent.sh` adds CI niceties: timestamped tags, prune-then-rebuild, dual
`latest` / `dev` tagging, `minikube image load`, and syncing the
`INSTALL_AGENT_IMAGE` env var to a running `litmusportal-server` deployment.

See [`install-agent/README.md`](install-agent/README.md) for every CLI flag.

---

## LiteLLM manifests

The Kubernetes deployment of the LiteLLM gateway used by the agents. These manifests are
a **downstream** of [`agentcert-stack/litellm-setup`](../agentcert-stack/litellm-setup) —
the stack repo is the source of truth, and `build-litellm.sh` in that repo substitutes
templated values into [`litellm/configmap.yaml`](litellm/configmap.yaml) before apply.

```
litellm/
├── namespace.yaml           # creates the `litellm` namespace
├── secret.yaml              # AZURE_API_KEY / _BASE / _MODEL / _API_VERSION
│                            # LITELLM_MASTER_KEY
│                            # LANGFUSE_PUBLIC_KEY / _SECRET_KEY / _HOST
├── configmap.yaml           # litellm config.yaml + custom_callbacks.py (LegacySpanEmitter)
│                            # — emits rich Langfuse spans with agent_id / experiment_id /
│                            #   workflow_name / scan_id metadata
└── deployment.yaml          # Deployment (1 replica) + ClusterIP Service on :4000
                             # Image: docker.io/litellm/litellm:v1.82.0-stable (pinned)
                             # Probes: HTTP /health/liveliness
                             # Resources: 250m/1 CPU, 512Mi/2Gi memory
```

Apply directly:

```bash
kubectl apply -f litellm/namespace.yaml
kubectl apply -f litellm/secret.yaml         # fill in real keys before apply
kubectl apply -f litellm/configmap.yaml
kubectl apply -f litellm/deployment.yaml
```

In-cluster endpoint: `http://litellm-proxy.litellm.svc.cluster.local:4000`.

---

## Installing an agent

### Via Helm directly (for development)

```bash
# flash-agent
helm upgrade --install flash-agent charts/flash-agent \
  --namespace flash-agent --create-namespace \
  -f charts/flash-agent/values.yaml \
  --set agent.config.AGENT_SCOPE_NAMESPACE=sock-shop \
  --set agent.secrets.OPENAI_API_KEY=$OPENAI_API_KEY
```

```bash
# k8s-agent
helm upgrade --install k8s-agent charts/k8s-agent \
  --namespace k8s-agent --create-namespace \
  -f charts/k8s-agent/values.yaml
```

### Via AgentCert ChaosCenter (production path)

1. In the UI, **AgentHub → Add Hub** pointing at this repository's URL.
2. The repository's `agents.chartserviceversion.yaml` declares each agent's metadata,
   capabilities, and **context-injection rules** (e.g. injecting
   `{{workflow.labels.workflow_id}}` into `agent.config.EXPERIMENT_ID`).
3. When a scenario installs the agent, ChaosCenter calls the `install-agent` image
   with the right `--folder`, namespace, and `--set` overrides.

---

## How it fits into AgentCert

| Component | Role |
|---|---|
| [`agentcert-stack`](../agentcert-stack) | Source of truth for the LiteLLM config templated into `litellm/configmap.yaml` |
| [`AgentCert`](../AgentCert) | Calls the `install-agent` container, registers the agent in the Mongo registry, drives Argo workflows |
| [`agent-sidecar`](../agent-sidecar) | Injected as a sidecar in `flash-agent` Deployments; transparently rewrites LLM requests with experiment/agent metadata |
| [`flash-agent`](../flash-agent) | Source for the `agentcert/agentcert-flash-agent` image bundled by `charts/flash-agent` |
| [`app-charts`](../app-charts) | Deploys the MCP servers (`kubernetes-mcp-server`, `prometheus-mcp-server`) in the target namespace that `MCP_URLS` points at |
| [`certifier`](../certifier) | Consumes the Langfuse traces emitted via `custom_callbacks.py` and produces the certification report |

---

## Versions

| Item | Version |
|---|---|
| `flash-agent` chart | `0.1.0` (appVersion `1.0.0`) |
| `k8s-agent` chart | `0.1.0` (appVersion `1.0.0`) |
| `install-agent` image | `agentcert/agentcert-install-agent:latest` (overridable) |
| Helm binary in image | `v3.14.0` |
| `kubectl` in image | `v1.29.0` |
| LiteLLM image | `litellm/litellm:v1.82.0-stable` |

---

## License

MIT — see [LICENSE](LICENSE).
