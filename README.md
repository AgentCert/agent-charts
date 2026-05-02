# Agent Charts

Helm charts and tooling for deploying AI agents into Kubernetes clusters. Part of the AgentCert platform for evaluating AI agent resilience under chaos engineering conditions.

## Contents

```
agent-charts/
├── charts/
│   ├── flash-agent/      # ITOps Kubernetes log analysis agent
│   └── k8s-agent/        # Generic Kubernetes operations agent
├── install-agent/        # Go CLI tool for agent deployment
└── litellm/              # LiteLLM proxy Kubernetes manifests
```

## Charts

### flash-agent

An LLM-powered ITOps agent that monitors Kubernetes namespaces, analyzes pod logs, and identifies issues.

```bash
# Install via Helm
helm upgrade --install flash-agent charts/flash-agent \
  --namespace flash-agent --create-namespace \
  -f custom-values.yaml
```

### k8s-agent

A generic Kubernetes operations agent template.

## Install-Agent CLI

A Go binary that installs agent charts from within Kubernetes (used by AgentCert server to deploy agents).

```bash
cd install-agent

# Build
make build

# Build Docker image with baked-in charts
make docker-build
```

See [install-agent/README.md](install-agent/README.md) for details.

## LiteLLM

Kubernetes manifests for deploying LiteLLM as an LLM gateway proxy.

```bash
kubectl apply -f litellm/namespace.yaml
kubectl apply -f litellm/secret.yaml
kubectl apply -f litellm/configmap.yaml
kubectl apply -f litellm/deployment.yaml
```

## Build All

```bash
# Build install-agent Docker image (includes all charts)
cd install-agent && make docker-build

# Load into kind cluster
kind load docker-image agentcert/agentcert-install-agent:latest --name agentcert
```

## Configuration

Each chart supports configuration via Helm values. Key settings:

| Chart | Key Values |
|-------|------------|
| flash-agent | `agent.config.K8S_NAMESPACE`, `agent.config.OPENAI_BASE_URL`, `sidecar.enabled` |
| k8s-agent | `agent.namespace`, `agent.llmEndpoint` |

## License

MIT License - see [LICENSE](LICENSE)
