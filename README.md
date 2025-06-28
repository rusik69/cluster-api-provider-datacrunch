# Cluster API Provider DataCrunch

A [Kubernetes Cluster API](https://cluster-api.sigs.k8s.io/) infrastructure provider for [DataCrunch](https://datacrunch.io/).

## Overview

This provider enables Kubernetes Cluster API to manage DataCrunch GPU instances and clusters. DataCrunch specializes in high-performance GPU infrastructure for machine learning, AI workloads, and HPC applications.

### Features

- **GPU-Optimized Instances**: Support for DataCrunch's full range of GPU instances including H100, A100, V100, and RTX series
- **Instance Management**: Create, update, and delete DataCrunch instances through Kubernetes CRDs
- **Cluster Lifecycle**: Complete cluster lifecycle management including control plane and worker nodes
- **Network Configuration**: Basic network setup and configuration
- **SSH Key Management**: Automated SSH key handling for instance access

### Supported DataCrunch Instance Types

- **H100 Series**: `1H100.80S.32V`, `2H100.80S.80V`, `4H100.80S.176V`, `8H100.80S.176V`
- **A100 Series**: `1A100.22V`, `2A100.44V`, `4A100.88V`, `8A100.176V`
- **V100 Series**: `1V100.6V`, `2V100.10V`, `4V100.20V`, `8V100.48V`
- **RTX Series**: `1RTX6000ADA.10V`, `2RTX6000ADA.20V`, `4RTX6000ADA.40V`, `8RTX6000ADA.80V`
- **L40S Series**: `1L40S.20V`, `2L40S.40V`, `4L40S.80V`, `8L40S.160V`
- **CPU Only**: `CPU.4V.16G`, `CPU.8V.32G`, `CPU.16V.64G`, etc.

## Quick Start

### Prerequisites

- Go 1.22+
- Kubernetes cluster (for management cluster)
- kubectl configured to access your management cluster
- DataCrunch account with API credentials

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/rusik69/cluster-api-provider-datacrunch.git
   cd cluster-api-provider-datacrunch
   ```

2. **Build and install the provider:**
   ```bash
   make build
   make install
   ```

3. **Deploy the provider:**
   ```bash
   make deploy
   ```

### Configuration

1. **Create a secret with DataCrunch credentials:**
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: datacrunch-credentials
     namespace: cluster-api-provider-datacrunch-system
   type: Opaque
   data:
     client-id: <base64-encoded-client-id>
     client-secret: <base64-encoded-client-secret>
   ```

2. **Create a DataCrunch cluster:**
   ```yaml
   apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
   kind: DataCrunchCluster
   metadata:
     name: my-cluster
   spec:
     region: "fin-01"
     controlPlaneEndpoint:
       host: ""  # Will be populated by the provider
       port: 6443
   ```

3. **Create a DataCrunch machine:**
   ```yaml
   apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
   kind: DataCrunchMachine
   metadata:
     name: my-machine
   spec:
     instanceType: "1H100.80S.32V"
     image: "ubuntu-22.04-cuda-12.1"
     sshKeyName: "my-ssh-key"
     publicIP: true
   ```

## API Reference

### DataCrunchCluster

The `DataCrunchCluster` resource represents the infrastructure for a Kubernetes cluster on DataCrunch.

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: DataCrunchCluster
metadata:
  name: example-cluster
spec:
  region: "fin-01"
  controlPlaneEndpoint:
    host: "cluster.example.com"
    port: 6443
  network:
    vpc:
      cidrBlock: "10.0.0.0/16"
    subnets:
    - cidrBlock: "10.0.1.0/24"
      isPublic: true
status:
  ready: true
  network:
    vpc:
      id: "vpc-12345"
      state: "available"
```

### DataCrunchMachine

The `DataCrunchMachine` resource represents a single DataCrunch instance.

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: DataCrunchMachine
metadata:
  name: example-machine
spec:
  instanceType: "1H100.80S.32V"
  image: "ubuntu-22.04-cuda-12.1"
  sshKeyName: "my-key"
  publicIP: true
  additionalTags:
    environment: "production"
    team: "ml-team"
  rootVolume:
    size: 100
    type: "SSD"
status:
  ready: true
  instanceState: "running"
  addresses:
  - type: "InternalIP"
    address: "10.0.1.10"
  - type: "ExternalIP"
    address: "203.0.113.10"
```

## Development

### Prerequisites for Development

- Go 1.22+
- Docker
- kubectl
- kustomize

### Building from Source

```bash
# Clone the repository
git clone https://github.com/rusik69/cluster-api-provider-datacrunch.git
cd cluster-api-provider-datacrunch

# Download dependencies
make deps

# Generate code and manifests
make generate manifests

# Build the binary
make build

# Run tests
make test

# Build docker image
make docker-build
```

### Running Locally

```bash
# Install CRDs into your cluster
make install

# Run the controller locally
make run
```

### Code Generation

This project uses controller-gen to generate CRDs and RBAC manifests:

```bash
# Generate CRDs and RBAC
make manifests

# Generate deep copy methods
make generate
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Cluster API   │    │  DataCrunch     │    │   DataCrunch    │
│   Core          │◄──►│  Provider       │◄──►│   API           │
│                 │    │  Controller     │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Cluster         │    │ DataCrunchCluster│    │ GPU Instances   │
│ Machine         │    │ DataCrunchMachine│    │ Load Balancers  │
│ MachineSet      │    │                 │    │ Networks        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for your changes
5. Run the test suite: `make test`
6. Submit a pull request

## Support

- **Documentation**: [Cluster API Book](https://cluster-api.sigs.k8s.io/)
- **DataCrunch Documentation**: [DataCrunch Docs](https://docs.datacrunch.io/)
- **Issues**: [GitHub Issues](https://github.com/rusik69/cluster-api-provider-datacrunch/issues)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Kubernetes Cluster API](https://cluster-api.sigs.k8s.io/) community
- [DataCrunch](https://datacrunch.io/) for providing GPU infrastructure
- All contributors to this project 