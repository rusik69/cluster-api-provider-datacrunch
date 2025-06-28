# Changelog

All notable changes to the Cluster API Provider for DataCrunch will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial implementation of Cluster API Provider for DataCrunch
- Support for DataCrunchCluster infrastructure resource
- Support for DataCrunchMachine infrastructure resource
- Complete instance type support including:
  - H100, H200, B200 series (latest GPU architectures)
  - A100 series (40GB and 80GB variants)
  - L40S and RTX 6000 Ada series
  - V100 series (legacy support)
  - CPU-only instances
- OAuth2 authentication with DataCrunch API
- Instance lifecycle management (create, start, stop, delete)
- SSH key management integration
- Image selection and management
- Status conditions and failure handling
- Finalizers for proper resource cleanup
- RBAC permissions and service accounts
- Deployment manifests and CRDs
- Sample resource configurations
- Comprehensive documentation

### Infrastructure
- Go module with Cluster API v1.7.3 compatibility
- Controller runtime v0.18.4 integration
- Kubernetes v0.30.2 support
- Kubebuilder annotations and code generation
- Docker build support
- Kustomize deployment configuration

### Documentation
- README with installation and usage instructions
- Deployment guide with step-by-step instructions
- Development guide for contributors
- API reference documentation
- Sample configurations for all resource types

### Known Issues
- Build process has dependency conflicts with k8s.io/component-base/metrics/testutil
- Some advanced watch configurations disabled due to API changes
- Load balancer implementation is placeholder (ready for DataCrunch API integration)

## [0.1.0] - 2024-06-28

### Added
- Initial project structure
- Basic API types and controllers
- Cloud client interface and DataCrunch implementation
- Build system and development tools 