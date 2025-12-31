# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2025-12-31

### Fixed
- Fixed PostgreSQL connection DSN in values.yaml (was using `-primary` suffix incorrectly)

### Added
- Support for CEMC, NES, and KUB electricity providers
- PDF parsing for rate extraction
- REST API for rate retrieval
- Prometheus metrics
- Helm chart for Kubernetes deployment
