# GitHub Actions Workflows

This directory contains the GitHub Actions workflows for the DataCrunch Cluster API Provider project. These workflows provide comprehensive CI/CD automation for testing, building, and releasing the project.

## Workflows Overview

### 1. Test Workflow (`test.yml`)

**Triggers:** Push to main/master, Pull Requests
**Purpose:** Primary CI pipeline for code quality and testing

**Jobs:**
- **unit-tests**: Runs Go unit tests with coverage reporting
- **e2e-tests**: Executes end-to-end tests using the mock DataCrunch API
- **lint**: Code quality checks using golangci-lint
- **security**: Security scanning with Gosec
- **build**: Builds binaries and container images
- **dependency-review**: Reviews dependency changes in PRs

**Features:**
- Go module caching for faster builds
- Coverage reporting to Codecov
- Artifact upload for failed e2e tests
- Multi-platform binary builds
- Container image testing

### 2. Release Workflow (`release.yml`)

**Triggers:** Git tags matching `v*`
**Purpose:** Automated releases and container image publishing

**Jobs:**
- **test**: Runs full test suite before release
- **build-and-push**: Builds and pushes multi-arch container images to GitHub Container Registry
- **create-release**: Creates GitHub releases with binaries and changelog

**Features:**
- Multi-architecture container images (linux/amd64, linux/arm64)
- Automated changelog generation
- Cross-platform binary builds
- GitHub Container Registry integration
- Semantic versioning support



## Setup Requirements

### Repository Secrets

No additional secrets are required for the basic workflows. The workflows use:
- `GITHUB_TOKEN`: Automatically provided by GitHub Actions

### Optional Integrations

1. **Codecov**: Add `CODECOV_TOKEN` secret for enhanced coverage reporting
2. **Container Registry**: Uses GitHub Container Registry by default (no setup needed)

## Workflow Configuration

### Environment Variables

All workflows use these common environment variables:
- `GO_VERSION`: "1.21" (can be updated centrally)
- `REGISTRY`: ghcr.io (GitHub Container Registry)
- `IMAGE_NAME`: Uses the repository name automatically

### Customization

#### Changing Go Version
Update the `GO_VERSION` environment variable in each workflow file.

#### Adding New Test Types
Extend the test workflow by adding new jobs or modifying existing ones.

#### Container Registry
To use a different container registry:
1. Update the `REGISTRY` environment variable
2. Add appropriate authentication secrets
3. Modify the login step in the release workflow

## Usage Examples

### Running Tests Locally

To run the same tests that run in CI:

```bash
# Unit tests
make test-unit

# E2E tests
make test-e2e

# All tests
make test-all

# Linting
golangci-lint run

# Build
make build

# Container image
make docker-build
```

### Manual Workflow Triggers

Some workflows support manual triggering:

```bash
# Trigger workflows manually
gh workflow run test.yml

# Check workflow status
gh workflow list
gh run list
```

### Creating Releases

Releases are automatically created when pushing tags:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will:
1. Run the full test suite
2. Build multi-arch container images
3. Create GitHub release with binaries
4. Generate changelog from commits

## Monitoring and Troubleshooting

### Workflow Status

Monitor workflow status through:
- GitHub Actions tab in the repository
- Commit status checks on PRs
- Email notifications (if enabled)

### Common Issues

1. **Test Failures**: Check the test artifacts uploaded on failure
2. **Build Failures**: Verify Go version compatibility
3. **Container Build Issues**: Check Docker configuration and base image availability
4. **Dependency Issues**: Review Dependabot PRs for dependency updates

### Debugging

1. **Enable Debug Logging**: Add `ACTIONS_STEP_DEBUG: true` to workflow environment
2. **SSH into Runners**: Use `mxschmitt/action-tmate@v3` for interactive debugging
3. **Artifact Analysis**: Download artifacts from failed runs for local investigation

## Maintenance

### Regular Updates

1. **Dependencies**: Dependabot automatically creates PRs for updates
2. **Actions Versions**: Update action versions in workflows quarterly
3. **Go Versions**: Update supported Go versions in test matrix annually
4. **Security Scanning**: Review security scan results regularly

### Performance Optimization

1. **Caching**: All workflows use Go module caching
2. **Parallel Execution**: Jobs run in parallel where possible
3. **Matrix Strategy**: Workflows use parallel execution for efficiency
4. **Container Layers**: Dockerfile is optimized for layer caching

## Security Considerations

1. **Secrets Management**: Use GitHub Secrets for sensitive data
2. **Dependency Scanning**: Automated security scanning in all workflows
3. **Container Security**: Uses distroless base images
4. **Permission Model**: Workflows use minimal required permissions
5. **Branch Protection**: Requires status checks before merging

## Integration with Development Workflow

### Pull Request Process

1. Developer creates PR
2. Test workflow runs automatically
3. All checks must pass before merge
4. Dependency review runs for dependency changes
5. Merge to main triggers additional testing

### Release Process

1. Maintainer creates and pushes version tag
2. Release workflow builds and tests
3. Container images published to registry
4. GitHub release created with artifacts
5. Documentation updated automatically

This workflow system ensures high code quality, comprehensive testing, and reliable releases while minimizing manual intervention. 