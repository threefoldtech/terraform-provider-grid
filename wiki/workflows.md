# Workflows

## lint workflow

Mainly to run the formatters, golangci-lint and staticcheck

## test workflow

- Runs against terraform matrix (only version 1.0.8) and only runs the tests in internal/provider directory
- Uses GoReleaser to test building the provider on all supported platforms.

TODO: check terraform versions

## terratests workflow

Runs the integration tests, it requires yggdrasil installation

## release workflow

Uses go-releaser

TODO: document the release process and link to terraform release documentation
