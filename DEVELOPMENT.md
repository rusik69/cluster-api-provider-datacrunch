# Development Guide

This guide explains how to develop and contribute to the Cluster API Provider for DataCrunch.

## Development Environment Setup

### Prerequisites

- Go 1.21+
- Docker
- kubectl
- A Kubernetes cluster for testing (kind/minikube recommended)

### Clone and Setup

```bash
git clone https://github.com/rusik69/cluster-api-provider-datacrunch.git
cd cluster-api-provider-datacrunch
make deps
```

## Building

### Build the Manager Binary

```bash
make build
```

### Generate Manifests

```bash
make manifests
```

### Generate Code

```bash
make generate
```

## Testing

### Run Unit Tests

```bash
# Note: Due to dependency issues with k8s.io/component-base/metrics/testutil,
# you may need to run tests with specific build tags or use go test directly
go test ./api/... ./pkg/...
```

### Run Controller Locally

```bash
make run
```

## Docker Images

### Build Docker Image

```bash
make docker-build IMG=your-registry/cluster-api-provider-datacrunch:latest
```

### Push Docker Image

```bash
make docker-push IMG=your-registry/cluster-api-provider-datacrunch:latest
```

## Project Structure

```
├── api/v1beta1/              # API definitions
│   ├── datacrunchcluster_types.go
│   ├── datacrunchmachine_types.go
│   └── conditions_consts.go
├── cmd/                      # Main application
│   └── main.go
├── internal/controller/      # Controllers
│   ├── datacrunchcluster_controller.go
│   └── datacrunchmachine_controller.go
├── pkg/cloud/               # Cloud client interfaces
│   ├── interfaces.go
│   └── datacrunch/         # DataCrunch implementation
│       └── client.go
├── config/                  # Kubernetes manifests
│   ├── crd/                # CRD definitions
│   ├── default/            # Default deployment
│   ├── manager/            # Manager deployment
│   ├── rbac/               # RBAC configuration
│   └── samples/            # Sample resources
├── dist/                   # Generated manifests
└── version/                # Version information
```

## Key Components

### Controllers

- **DataCrunchClusterReconciler**: Manages cluster infrastructure
- **DataCrunchMachineReconciler**: Manages individual machines

### Cloud Client

The cloud client interface (`pkg/cloud/interfaces.go`) defines the contract for interacting with cloud providers. The DataCrunch implementation (`pkg/cloud/datacrunch/client.go`) provides:

- Instance management (create, start, stop, delete)
- Image management
- SSH key management
- Load balancer operations (placeholder)

### API Types

- **DataCrunchCluster**: Represents cluster-level infrastructure
- **DataCrunchMachine**: Represents individual machine instances

## Adding New Features

### Adding New Instance Types

1. Update the DataCrunch client with new instance type constants
2. Update documentation and samples
3. Test with actual DataCrunch API

### Adding New Regions

1. Update region constants in the API types
2. Update validation logic if needed
3. Update documentation

### Implementing Load Balancer Support

1. Extend the cloud client interface
2. Implement DataCrunch load balancer API calls
3. Update the cluster controller reconciliation logic

## Contributing

### Code Style

- Follow standard Go conventions
- Use gofmt and golint
- Add unit tests for new functionality
- Update documentation

### Commit Messages

Use conventional commit format:
```
feat: add support for B200 instances
fix: resolve instance state polling issue
docs: update deployment guide
```

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Update documentation
6. Submit a pull request

## Debugging

### Enable Debug Logging

Set the log level in the manager deployment:

```yaml
containers:
- command:
  - /manager
  args:
  - --leader-elect
  - --v=4  # Add for debug logging
```

### Common Issues

1. **Compilation Errors**: Due to dependency issues with k8s.io/component-base, you may need to skip the vet step during development
2. **API Authentication**: Ensure DataCrunch credentials are properly configured
3. **Network Issues**: Check connectivity between the management cluster and DataCrunch API

### Testing with Local Changes

1. Build a custom image with your changes
2. Update the deployment manifest to use your image
3. Deploy to a test cluster
4. Verify functionality with sample resources

## Release Process

1. Update version in `version/version.go`
2. Update CHANGELOG.md
3. Create and push a git tag
4. Build and push release images
5. Generate release manifests
6. Create GitHub release with artifacts

## Architecture Notes

### Controller Design

The controllers follow the standard Cluster API provider pattern:
- Watch for changes to infrastructure resources
- Reconcile desired state with actual cloud resources
- Update status with current state and conditions

### Error Handling

- Use conditions to report status
- Implement proper finalizers for cleanup
- Handle API rate limiting and retries

### Security

- Store credentials in Kubernetes secrets
- Use RBAC for minimal required permissions
- Validate input parameters to prevent injection 