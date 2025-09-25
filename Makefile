<<<<<<< HEAD
# FleetForge Makefile

# Variables
BINARY_NAME=fleetforge
DOCKER_REGISTRY ?= ghcr.io/astrosteveo
IMAGE_TAG ?= latest
GO_VERSION = 1.21

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build targets
.PHONY: all build clean test coverage lint fmt vet deps help
.PHONY: docker-build docker-push docker-run
.PHONY: crd-generate crd-install crd-uninstall
.PHONY: kind-create kind-delete kind-load

## Build commands
all: deps lint test build ## Run all checks and build

build: ## Build the binary
	$(GOBUILD) -o bin/$(BINARY_NAME) ./cmd/fleetforge

build-cell: ## Build the cell service
	$(GOBUILD) -o bin/cell ./cmd/cell

build-gateway: ## Build the gateway service
	$(GOBUILD) -o bin/gateway ./cmd/gateway

build-controller: ## Build the controller
	$(GOBUILD) -o bin/controller ./cmd/controller

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf bin/

## Testing commands
test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

test-integration: ## Run integration tests
	$(GOTEST) -v -tags=integration ./...

## Code quality commands
lint: ## Run linter
	$(GOLINT) run

fmt: ## Format code
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOCMD) vet ./...

## Dependencies
deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	$(GOMOD) get -u all
	$(GOMOD) tidy

## Docker commands
docker-build: ## Build Docker image
	docker build -t $(DOCKER_REGISTRY)/$(BINARY_NAME):$(IMAGE_TAG) .

docker-push: ## Push Docker image
	docker push $(DOCKER_REGISTRY)/$(BINARY_NAME):$(IMAGE_TAG)

docker-run: ## Run Docker container
	docker run -p 8080:8080 $(DOCKER_REGISTRY)/$(BINARY_NAME):$(IMAGE_TAG)

## Kubernetes CRD commands
crd-generate: ## Generate CRD manifests
	controller-gen crd:trivialVersions=true paths="./api/..." output:crd:artifacts:config=config/crd

crd-install: ## Install CRDs in cluster
	kubectl apply -f config/crd/

crd-uninstall: ## Uninstall CRDs from cluster
	kubectl delete -f config/crd/

## Kind cluster commands
kind-create: ## Create kind cluster for development
	kind create cluster --name fleetforge --config deploy/kind-config.yaml

kind-delete: ## Delete kind cluster
	kind delete cluster --name fleetforge

kind-load: docker-build ## Load Docker image into kind cluster
	kind load docker-image $(DOCKER_REGISTRY)/$(BINARY_NAME):$(IMAGE_TAG) --name fleetforge

## Development helpers
dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	@echo "Development environment ready!"

gen: ## Generate code (CRDs, clients, etc.)
	$(GOCMD) generate ./...

## Help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
=======
# FleetForge - Makefile for development and deployment

# Image URL to use all building/pushing image targets
IMG ?= fleetforge:latest
CONTROLLER_IMG ?= fleetforge-controller:latest
CELL_IMG ?= fleetforge-cell:latest

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

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/controller-manager/main.go

.PHONY: run-cell
run-cell: fmt vet ## Run a cell simulator from your host.
	go run ./cmd/cell-simulator/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/dev-best-practices/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${CONTROLLER_IMG} -f Dockerfile.controller .

.PHONY: docker-build-cell
docker-build-cell: ## Build docker image with the cell simulator.
	docker build -t ${CELL_IMG} -f Dockerfile.cell .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${CONTROLLER_IMG}

.PHONY: docker-push-cell
docker-push-cell: ## Push docker image with the cell simulator.
	docker push ${CELL_IMG}

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
cluster-load: docker-build docker-build-cell ## Load docker images into Kind cluster
	kind load docker-image $(CONTROLLER_IMG) --name $(CLUSTER_NAME)
	kind load docker-image $(CELL_IMG) --name $(CLUSTER_NAME)

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
>>>>>>> origin/main
