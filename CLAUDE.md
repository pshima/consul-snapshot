# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

consul-snapshot is a backup and restore utility for HashiCorp Consul that runs as a daemon. It backs up Consul's K/V store, ACLs, and Prepared Queries to Amazon S3 or Google Cloud Storage with optional encryption.

## Development Commands

### Build and Test
```bash
# Install dependencies
make deps

# Build the project
make build

# Run tests with linting and vetting
make test

# Format code
make fmt

# Install locally
make install

# Build for all platforms
make build-all
```

### Running the Application
```bash
# Run backup daemon
./consul-snapshot backup

# Run backup once
./consul-snapshot backup -once

# Restore from backup
./consul-snapshot restore <backup-path>

# Check version
./consul-snapshot version
```

### Testing
```bash
# Run unit tests
go test ./...

# Run acceptance test (requires ACCEPTANCE_TEST=1)
ACCEPTANCE_TEST=1 make test

# Generate coverage report
make cov
```

## Architecture

### Command Structure
The application uses mitchellh/cli for command-line interface with three main commands:
- `backup` - Runs backup daemon or single backup
- `restore` - Restores from S3/GCS backup
- `version` - Shows version information

Commands are defined in `commands.go` and implemented in `command/` directory.

### Core Components
- **backup/** - Core backup logic, S3/GCS uploaders, encryption
- **restore/** - Restore functionality from cloud storage
- **config/** - Environment-based configuration management
- **consul/** - Consul API client wrapper
- **crypt/** - Encryption/decryption for backups
- **health/** - HTTP health check server

### Configuration
All configuration is environment-based. Key variables:
- `S3BUCKET`/`GCSBUCKET` - Storage destinations
- `BACKUPINTERVAL` - Backup frequency in seconds
- `CRYPTO_PASSWORD` - Encryption passphrase
- `CONSUL_HTTP_*` - Consul connection settings

### Data Flow
1. **Backup**: Consul API → JSON serialization → Optional encryption → Archive (tar.gz) → S3/GCS
2. **Restore**: S3/GCS download → Extract → Optional decryption → Parse JSON → Consul API

### Storage Format
Backups are tar.gz archives containing:
- `consul.kv.{timestamp}.json` - Key/Value data
- `consul.acl.{timestamp}.json` - ACL data
- `consul.pq.{timestamp}.json` - Prepared Query data
- `meta.json` - Backup metadata and checksums

### Health Monitoring
HTTP server on `/health` endpoint returns:
- 200 if last backup is < 1 hour old
- 500 if backup is stale or failed

## Go Module
- Go 1.24+ required (updated from 1.19)
- Uses Go modules (`go.mod`)
- Major dependencies: consul/api, aws-sdk-go, mitchellh/cli
- Recently updated dependencies and fixed compatibility issues

## Recent Updates (2024)
- Updated Go version requirement from 1.19 to 1.24
- Updated dependencies to compatible versions
- Fixed format string linting error in backup/backup.go:96
- Verified all tests pass and application builds successfully