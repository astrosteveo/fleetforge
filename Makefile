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