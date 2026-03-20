# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

## v0.1.0

### Fixed
- Exclude golangci-lint v1 transitive dep, update kafka to v1.22.8
- fix: pin opencontainers/runtime-spec to v1.2.1 to resolve containerd v1.7.30 compilation failure with Go 1.26

### Added
- Initial project structure from go-skeleton
- Module github.com/bborbe/task-watcher
