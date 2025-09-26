# Contributing to FleetForge

Thank you for your interest in contributing to FleetForge! This document provides guidelines and information for contributors.

## Code of Conduct

We are committed to fostering a welcoming and inclusive community. Please be respectful and professional in all interactions.

## Getting Started

### Prerequisites

Before contributing, ensure you have the following installed:

- Go 1.21 or later
- Docker and Docker Compose
- kubectl (for Kubernetes cluster interaction)
- kind (for local Kubernetes development)
- make (for running build commands)

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/fleetforge.git
   cd fleetforge
   ```

2. **Install Dependencies**
   ```bash
   go mod tidy
   ```

3. **Verify Setup**
   ```bash
   make test
   make lint
   make build
   ```

4. **Create Development Cluster**
   ```bash
   make cluster-create
   make install
   make deploy
   ```

For detailed setup instructions, see [DEVELOPMENT.md](DEVELOPMENT.md).

## How to Contribute

### Reporting Issues

- Use GitHub Issues to report bugs or request features
- Provide detailed information including:
  - Go version and OS
  - Steps to reproduce
  - Expected vs actual behavior
  - Relevant logs or error messages

### Making Changes

1. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

2. **Make Your Changes**
   - Follow the existing code patterns and style
   - Add tests for new functionality
   - Update documentation as needed

3. **Test Your Changes**
   ```bash
   make test
   make lint
   make build
   ```

4. **Test Integration**
   ```bash
   make cluster-load
   make deploy
   kubectl apply -f config/samples/fleetforge_v1_worldspec.yaml
   kubectl get worldspecs -o wide
   ```

5. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add new feature description"
   # or
   git commit -m "fix: resolve issue description"
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/) format:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for adding/updating tests
   - `refactor:` for code refactoring
   - `ci:` for CI/CD changes

6. **Push and Create Pull Request**
   ```bash
   git push origin feature/your-feature-name
   ```

## Development Guidelines

### Code Style

- Follow Go best practices and idiomatic Go code
- Use `gofmt` and `go vet` (automated via `make lint`)
- Ensure all code passes `golangci-lint` checks
- Keep functions focused and testable
- Use meaningful variable and function names

### Testing

- Write unit tests for all new functionality
- Maintain or improve test coverage
- Use table-driven tests where appropriate
- Test error conditions and edge cases
- Integration tests should use the sample WorldSpec

### Documentation

- Update relevant documentation for changes
- Include examples in code comments
- Update API documentation for new endpoints
- Keep DEVELOPMENT.md current with setup changes

### Architecture Guidelines

FleetForge follows the Kubernetes operator pattern:

- **WorldSpec CRD**: Defines desired game world state
- **Controller**: Reconciles actual state with desired state
- **Cell Simulator**: Runs spatial region simulations
- **Service Discovery**: Cells discover each other via Kubernetes services

Key principles:
- Controllers should be idempotent
- Use structured logging with context
- Handle errors gracefully with retries
- Follow Kubernetes API conventions

## Pull Request Process

1. **Ensure CI Passes**
   - All tests must pass
   - Linting must pass
   - Security scans must pass
   - Build must succeed

2. **Provide Clear Description**
   - Reference related issues
   - Describe changes made
   - Include testing performed
   - Note any breaking changes

3. **Request Review**
   - Tag relevant maintainers
   - Be responsive to feedback
   - Make requested changes promptly

4. **Merge Requirements**
   - At least one approving review
   - All CI checks passing
   - No conflicts with main branch
   - Up-to-date with main branch

## Project Structure

```
fleetforge/
├── api/v1/                     # API definitions and CRDs
├── cmd/                        # Main applications
│   ├── controller-manager/     # Kubernetes controller
│   └── cell-simulator/         # Game cell simulator
├── config/                     # Kubernetes configurations
│   ├── crd/                    # Custom Resource Definitions
│   ├── rbac/                   # Role-based access control
│   └── samples/                # Example resources
├── deploy/                     # Deployment configurations
├── docs/                       # Project documentation
├── pkg/                        # Reusable libraries
│   ├── controllers/            # Controller implementations
│   └── cell/                   # Cell simulation logic
└── hack/                       # Development scripts
```

## Common Development Tasks

### Adding New CRD Fields

1. Update `api/v1/*_types.go`
2. Run `make generate manifests`
3. Update samples in `config/samples/`
4. Add controller logic in `pkg/controllers/`
5. Write tests for new functionality

### Debugging Controllers

```bash
# View controller logs
kubectl logs -l app.kubernetes.io/name=fleetforge-controller-manager

# Port forward for debugging
kubectl port-forward deployment/fleetforge-controller-manager-controller-manager 8080:8080

# Check CRD status
kubectl get worldspecs -o yaml
kubectl describe worldspec worldspec-sample
```

### Performance Testing

```bash
# Apply multiple WorldSpecs
for i in {1..5}; do
  kubectl apply -f config/samples/fleetforge_v1_worldspec.yaml
done

# Monitor resource usage
kubectl top pods
kubectl top nodes
```

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create release branch
4. Tag release
5. CI automatically builds and pushes images
6. Create GitHub release with notes

## Getting Help

- Check [DEVELOPMENT.md](DEVELOPMENT.md) for setup issues
- Search existing GitHub Issues
- Ask questions in GitHub Discussions
- Review architecture docs in `docs/`

## Recognition

Contributors will be recognized in:
- Release notes for significant contributions
- GitHub contributor statistics
- Project documentation

Thank you for contributing to FleetForge! 🚀