# FleetForge - Makefile for development and deployment

# Simple documentation PDF generation
# Requires pandoc and a LaTeX engine installed (e.g., tectonic or xelatex)
DOCS_MD := $(wildcard docs/*.md)
DOCS_PDF := $(DOCS_MD:.md=.pdf)
PANDOC_ENGINE ?= tectonic
PANDOC_FROM ?= gfm+footnotes+autolink_bare_uris

# Image URL to use all building/pushing image targets
IMG ?= fleetforge:latest
CONTROLLER_IMG ?= fleetforge-controller:latest
CELL_IMG ?= fleetforge-cell:latest
GATEWAY_IMG ?= fleetforge-gateway:latest

# Kubernetes cluster context
CLUSTER_NAME ?= fleetforge-dev
K8S_VERSION ?= v1.28.3

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Documentation

.PHONY: pdfs
pdfs: $(DOCS_PDF) ## Generate PDF versions of documentation
	@echo "Generated: $(DOCS_PDF)"

# Pattern rule: docs/file.md -> docs/file.pdf
# Uses --metadata to set a basic title, and table of contents.
# To force rebuild: make clean-docs && make pdfs
docs/%.pdf: docs/%.md
	@echo "Converting $< -> $@"
	@pandoc "$<" \
	  --from $(PANDOC_FROM) \
	  --pdf-engine=$(PANDOC_ENGINE) \
	  --toc --toc-depth=3 \
	  -V geometry:margin=1in \
	  -V linkcolor:blue \
	  -o "$@"

.PHONY: clean-docs
clean-docs: ## Clean generated PDFs
	@rm -f $(DOCS_PDF)
	@echo "Removed generated PDFs"

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

.PHONY: test-with-manifests
test-with-manifests: manifests generate fmt vet ## Run tests with manifest generation.
	go test ./... -coverprofile cover.out

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/manager cmd/controller-manager/main.go

.PHONY: build-with-manifests
build-with-manifests: manifests generate fmt vet ## Build manager binary with manifest generation.
	go build -o bin/manager cmd/controller-manager/main.go

.PHONY: build-cell
build-cell: fmt vet ## Build cell simulator binary.
	go build -o bin/cell-simulator cmd/cell-simulator/main.go

.PHONY: build-gateway
build-gateway: fmt vet ## Build gateway binary.
	go build -o bin/gateway cmd/gateway/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/controller-manager/main.go

.PHONY: run-cell
run-cell: fmt vet ## Run a cell simulator from your host.
	go run ./cmd/cell-simulator/main.go

.PHONY: run-gateway
run-gateway: fmt vet ## Run a gateway from your host.
	go run ./cmd/gateway/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/dev-best-practices/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${CONTROLLER_IMG} -f Dockerfile.controller .

.PHONY: docker-build-cell
docker-build-cell: ## Build docker image with the cell simulator.
	docker build -t ${CELL_IMG} -f Dockerfile.cell .

.PHONY: docker-build-gateway
docker-build-gateway: ## Build docker image with the gateway.
	docker build -t ${GATEWAY_IMG} -f Dockerfile.gateway .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${CONTROLLER_IMG}

.PHONY: docker-push-cell
docker-push-cell: ## Push docker image with the cell simulator.
	docker push ${CELL_IMG}

.PHONY: docker-push-gateway
docker-push-gateway: ## Push docker image with the gateway.
	docker push ${GATEWAY_IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/crd/bases

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete --ignore-not-found=$(ignore-not-found) -f config/crd/bases

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && kustomize edit set image controller=${CONTROLLER_IMG}
	kubectl apply -k config/default

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete --ignore-not-found=$(ignore-not-found) -k config/default

##@ Local Development

.PHONY: cluster-create
cluster-create: ## Create a local Kind cluster for development
	kind create cluster --name $(CLUSTER_NAME) --config hack/kind-config.yaml

.PHONY: cluster-delete
cluster-delete: ## Delete the local Kind cluster
	kind delete cluster --name $(CLUSTER_NAME)

.PHONY: cluster-load
cluster-load: docker-build docker-build-cell docker-build-gateway ## Load docker images into Kind cluster
	kind load docker-image $(CONTROLLER_IMG) --name $(CLUSTER_NAME)
	kind load docker-image $(CELL_IMG) --name $(CLUSTER_NAME)
	kind load docker-image $(GATEWAY_IMG) --name $(CLUSTER_NAME)

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.13.0
GOLANGCI_LINT_VERSION ?= v1.54.2

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary. If wrong version is installed, it will be overwritten.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint && $(LOCALBIN)/golangci-lint --version | grep -q $(GOLANGCI_LINT_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)