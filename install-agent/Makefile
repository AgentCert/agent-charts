# Makefile for install-agent Docker image
# Build and push Helm chart installer for agents

# Anchored to this Makefile, not to the caller's working directory, so the
# targets behave the same however make is invoked.
ACE_REPO_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/../..)
ACE_REGISTRY_LIB := $(ACE_REPO_ROOT)/scripts/lib/registry.sh
ACE_ENV_FILE ?= $(ACE_REPO_ROOT)/.env

# Configuration - override these with environment variables or command line
# IMAGE is the image's PUBLIC name (e.g. agentcert/agentcert-install-agent:latest).
# Which registry it is built from and pushed to is resolved at run time by
# scripts/lib/registry.sh from IMAGE_REGISTRY / IMAGE_MIRROR_NAMESPACE, so a
# registry host must never be baked into this reference -- doing so drops the
# mirror namespace and publishes where no other ACE tool looks.
IMAGE_NAMESPACE ?= agentcert
IMAGE_NAME ?= agentcert-install-agent
IMAGE_TAG ?= latest

# Full image reference
IMAGE = $(IMAGE_NAMESPACE)/$(IMAGE_NAME):$(IMAGE_TAG)

# Default chart folder
FOLDER ?= flash-agent

KUBECONFIG ?= $(HOME)/.kube/config

# Build context - parent directory (agent-charts root)
BUILD_CONTEXT = ..

# Run a command with registry.sh sourced and the env file loaded. The caller's
# IMAGE_REGISTRY is deliberately left intact: registry_load reads the
# environment first, so `IMAGE_REGISTRY= make build` forces the public path.
ACE_WITH_REGISTRY = bash -c 'source "$$1"; registry_load "$$2" || exit; shift 2; "$$@"' _ "$(ACE_REGISTRY_LIB)" "$(ACE_ENV_FILE)"
# Tag the public name to its resolved destination and push that.
ACE_PUSH = bash -c 'source "$$1"; registry_load "$$2" || exit; dst="$$(registry_ref "$$3")" || exit; [ "$$dst" = "$$3" ] || docker tag "$$3" "$$dst" || exit; echo "Pushing Docker image: $$dst"; docker push "$$dst"' _ "$(ACE_REGISTRY_LIB)" "$(ACE_ENV_FILE)"
# Print where a public name actually resolves to under the current .env.
ACE_RESOLVE = bash -c 'source "$$1"; registry_load "$$2" >/dev/null || exit; echo "Resolved:  $$(registry_ref "$$3")"' _ "$(ACE_REGISTRY_LIB)" "$(ACE_ENV_FILE)"

.PHONY: require-ace-lib
require-ace-lib: ## Fail early (and readably) outside an ace-monorepo checkout
	@test -f "$(ACE_REGISTRY_LIB)" || { \
		echo "ERROR: $(ACE_REGISTRY_LIB) not found." >&2; \
		echo "       This target resolves images through the shared ACE registry library;" >&2; \
		echo "       run it from an ace-monorepo checkout (or set ACE_REPO_ROOT)." >&2; \
		exit 1; }

.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; } /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: build
build: require-ace-lib ## Build the Docker image
	@echo "Building Docker image: $(IMAGE)"
	$(ACE_WITH_REGISTRY) docker_build_resolved Dockerfile -t "$(IMAGE)" "$(BUILD_CONTEXT)"

.PHONY: build-no-cache
build-no-cache: require-ace-lib ## Build the Docker image without cache
	@echo "Building Docker image (no cache): $(IMAGE)"
	$(ACE_WITH_REGISTRY) docker_build_resolved Dockerfile --no-cache -t "$(IMAGE)" "$(BUILD_CONTEXT)"

.PHONY: push
push: require-ace-lib ## Push the Docker image to the resolved registry
	$(ACE_PUSH) "$(IMAGE)"

.PHONY: build-push
build-push: build ## Build successfully before publishing, including under make -j
	$(MAKE) push

.PHONY: run
run: ## Run the container with help
	docker run --rm $(IMAGE)

.PHONY: run-shell
run-shell: ## Run container with interactive shell
	docker run --rm -it --entrypoint /bin/bash $(IMAGE)

.PHONY: list-charts
list-charts: ## List charts available in the image
	docker run --rm --entrypoint /bin/sh $(IMAGE) -c "ls -la /charts"

.PHONY: install
install: ## Install chart (FOLDER=flash-agent by default, NAMESPACE=<ns> KUBECONFIG=<path>)
	@test -f "$(KUBECONFIG)" || { echo "KUBECONFIG file not found" >&2; exit 2; }
	$(if $(VALUES),@test -f "$(VALUES)" || { echo "VALUES file not found" >&2; exit 2; })
	docker run --rm \
		-v "$(abspath $(KUBECONFIG)):/kubeconfig:ro" -e KUBECONFIG=/kubeconfig \
		--network host \
		$(if $(VALUES),-v "$(abspath $(VALUES)):/values.yaml:ro") \
		$(IMAGE) \
		-folder "$(FOLDER)" \
		$(if $(KUBE_CONTEXT),-context "$(KUBE_CONTEXT)") \
		$(if $(NAMESPACE),-namespace $(NAMESPACE)) \
		$(if $(RELEASE),-release $(RELEASE)) \
		$(if $(VALUES),-values /values.yaml) \
		$(if $(filter 1 true yes,$(DRY_RUN)),-dry-run)

.PHONY: install-local
install-local: install ## Install using local kubeconfig (same checks as install)

.PHONY: tag
tag: ## Tag image with additional tag (use NEW_TAG=<tag>)
ifndef NEW_TAG
	$(error NEW_TAG is required. Usage: make tag NEW_TAG=v1.0.0)
endif
	docker tag $(IMAGE) $(IMAGE_NAMESPACE)/$(IMAGE_NAME):$(NEW_TAG)

.PHONY: clean
clean: ## Remove local Docker image
	docker rmi $(IMAGE) || true

.PHONY: test
test: ## Run tests
	@echo "Running Go tests..."
	go test -v ./...

.PHONY: lint
lint: ## Run linter
	@echo "Running Go linter..."
	golangci-lint run ./...

.PHONY: version
version: ## Show current version configuration
	@echo "Image:     $(IMAGE)"
	@echo "Namespace: $(IMAGE_NAMESPACE)"
	@echo "Name:      $(IMAGE_NAME)"
	@echo "Tag:       $(IMAGE_TAG)"
	@test -f "$(ACE_REGISTRY_LIB)" \
		&& $(ACE_RESOLVE) "$(IMAGE)" \
		|| echo "Resolved:  (unavailable outside an ace-monorepo checkout)"
