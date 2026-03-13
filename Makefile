IMG_LOCAL ?= harbor-operator:local
IMG ?= $(IMG_LOCAL)
HARBOR_API_GROUP ?= harbor.harbor-operator.io
HARBOR_OPENAPI_URL ?= https://raw.githubusercontent.com/goharbor/harbor/refs/heads/main/api/v2.0/swagger.yaml
CRD_REF_DOCS_OUTPUT ?= docs/reference/api.md
DOCS_CONTAINER_IMAGE ?= squidfunk/mkdocs-material:9.7
MKDOCS_CONFIG ?= hack/mkdocs.yml

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate DeepCopy, DeepCopyInto, and DeepCopyObject implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

.PHONY: test-e2e
test-e2e: manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	@command -v kind >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@kind get clusters | grep -q 'kind' || { \
		echo "No Kind cluster is running. Please start a Kind cluster before running the e2e tests."; \
		exit 1; \
	}
	go test -tags=e2e ./test/e2e/ -v -ginkgo.v

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

.PHONY: sync-chart-crds
sync-chart-crds: ## Sync CRDs into the Helm chart
	./hack/sync-chart-crds.sh

.PHONY: sync-chart-rbac
sync-chart-rbac: ## Sync RBAC into the Helm chart
	./hack/sync-chart-rbac.sh

.PHONY: sync-chart
sync-chart: sync-chart-crds sync-chart-rbac ## Sync generated CRDs and RBAC into the Helm chart

.PHONY: generate-docs
generate-docs: crd-ref-docs ## Generate the CRD API reference documentation
	mkdir -p $(dir $(CRD_REF_DOCS_OUTPUT))
	$(CRD_REF_DOCS) \
		--config hack/crd-ref-docs.yaml \
		--renderer markdown \
		--source-path ./api \
		--output-path $(CRD_REF_DOCS_OUTPUT)

.PHONY: docs-site-build
docs-site-build: ## Build the MkDocs documentation site
	$(CONTAINER_TOOL) run --rm -v "$(CURDIR)":/docs $(DOCS_CONTAINER_IMAGE) build --strict -f $(MKDOCS_CONFIG)

.PHONY: docs-build
docs-build: generate-docs docs-site-build ## Generate API reference docs and build the MkDocs documentation site

.PHONY: docs-serve
docs-serve: generate-docs ## Serve the MkDocs documentation site locally
	$(CONTAINER_TOOL) run --rm -p 8000:8000 -v "$(CURDIR)":/docs $(DOCS_CONTAINER_IMAGE) serve --dev-addr 0.0.0.0:8000 -f $(MKDOCS_CONFIG)

.PHONY: update-harbor-openapi
update-harbor-openapi: ## Download the Harbor OpenAPI spec into hack/harbor-openapi.yaml
	curl -fsSL $(HARBOR_OPENAPI_URL) -o hack/harbor-openapi.yaml

.PHONY: apply-crds
apply-crds: ## Apply the latest CRDs to the current cluster
	$(KUBECTL) apply -f charts/harbor-operator/crds

.PHONY: prepare-deploy
prepare-deploy: manifests generate sync-chart apply-crds ## Generate code/manifests, sync chart assets, and apply CRDs

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

##@ Deployment

.PHONY: uninstall
uninstall: ## Uninstall CRDs for the harbor.harbor-operator.io API group.
	$(KUBECTL) delete --ignore-not-found -f charts/harbor-operator/crds

.PHONY: deploy
deploy: prepare-deploy ## Deploy controller via Helm chart with synced CRDs and RBAC.
	$(KUBECTL) create namespace harbor-operator-system --dry-run=client -o yaml | $(KUBECTL) apply -f -
	IMG_REPO=$$(echo $(IMG) | cut -d: -f1); \
	IMG_TAG=$$(echo $(IMG) | cut -d: -f2); \
	helm upgrade --install harbor-operator ./charts/harbor-operator \
		--namespace harbor-operator-system \
		--set image.repository=$${IMG_REPO} \
		--set image.tag=$${IMG_TAG} \
		--wait
	$(KUBECTL) rollout restart deployment/harbor-operator -n harbor-operator-system
	$(KUBECTL) rollout status deployment/harbor-operator -n harbor-operator-system --timeout=180s

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster.
	helm uninstall harbor-operator -n harbor-operator-system --ignore-not-found

##@ Samples

.PHONY: apply-samples
apply-samples: ## Apply sample CRs
	$(KUBECTL) apply -k config/samples

.PHONY: delete-crs
delete-crs: ## Delete all CRs in harbor.harbor-operator.io API group (HarborConnection last)
	HARBOR_API_GROUP=$(HARBOR_API_GROUP) hack/delete-crs.sh

##@ Kind

.PHONY: kind-install-stack
kind-install-stack: ## Install Traefik + Harbor + harbor-operator into the current cluster
	helm upgrade --install traefik oci://ghcr.io/traefik/helm/traefik\
		--set service.type=NodePort \
		--set ports.web.nodePort=30080 \
		--set ports.websecure.nodePort=30443 \
		--wait
	helm repo add harbor https://helm.goharbor.io --force-update
	helm upgrade --install harbor harbor/harbor --wait
	$(MAKE) kind-deploy

.PHONY: kind-up
kind-up: ## Create Kind cluster (default CNI) + Traefik + Harbor + harbor-operator
	kind create cluster --config hack/kind-configuration.yaml
	$(MAKE) kind-install-stack

.PHONY: kind-up-cilium
kind-up-cilium: ## Create Kind cluster with Cilium CNI + Traefik + Harbor + harbor-operator
	kind create cluster --config hack/kind-configuration-cilium.yaml
	helm repo add cilium https://helm.cilium.io --force-update
	helm repo update
	helm upgrade --install cilium cilium/cilium \
		--namespace kube-system \
		--set hubble.enabled=true \
		--set hubble.relay.enabled=true \
		--set hubble.ui.enabled=true \
		--set operator.replicas=1 \
		--set k8sServiceHost=kind-control-plane \
		--set k8sServicePort=6443 \
		--wait
	$(MAKE) kind-install-stack

.PHONY: kind-down
kind-down: ## Delete the Kind cluster named "kind"
	kind delete cluster

.PHONY: kind-load-image
kind-load-image: ## Load the local operator image into the Kind cluster
	kind load docker-image $(IMG_LOCAL) --name kind

.PHONY: kind-deploy
kind-deploy: ## Build the image, sync/apply manifests, load it into Kind, and deploy the operator to the current Kind cluster.
	@echo "Building local image..."
	$(MAKE) docker-build
	@echo "Loading image into Kind cluster..."
	$(MAKE) kind-load-image
	@echo "Deploying harbor-operator to the cluster..."
	$(MAKE) deploy

.PHONY: kind-refresh
kind-refresh: ## Iterative operator refresh for the current Kind cluster.
	$(MAKE) kind-deploy

.PHONY: kind-reset
kind-reset: delete-crs undeploy uninstall ## Remove operator deployment, CRDs and samples

.PHONY: kind-redeploy
kind-redeploy: kind-reset kind-deploy ## Full reset + redeploy operator on existing Kind cluster

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.20.1
ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
GOLANGCI_LINT_VERSION ?= v1.63.4
CRD_REF_DOCS_VERSION ?= v0.3.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: crd-ref-docs
crd-ref-docs: $(CRD_REF_DOCS) ## Download crd-ref-docs locally if necessary.
$(CRD_REF_DOCS): $(LOCALBIN)
	$(call go-install-tool,$(CRD_REF_DOCS),github.com/elastic/crd-ref-docs,$(CRD_REF_DOCS_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
