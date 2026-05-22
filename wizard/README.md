# HoneyWire Wizard - Intelligent Deception CLI

## Overview

The HoneyWire Wizard is a zero-friction, intelligent command-line agent designed to assess a host's attack surface and automatically deploy contextual deception infrastructure (honeypots).

It uses correlated discovery to map active processes to network sockets and Docker containers, keeping deployments precise and low-noise. The Wizard fetches manifests from the HoneyWire Hub and executes them locally without hardcoding sensor behavior.

## Architecture

The current architecture separates runtime orchestration from platform domain logic.

### Command entry point
- `cmd/wizard/main.go` — CLI startup, flags, and command dispatch.

### Runtime / orchestration layer (`internal/`)
- `internal/app/` — runtime session, node config, and app state.
- `internal/cli/` — terminal rendering, prompts, and UX helpers.
- `internal/commands/` — high-level command orchestrators.
- `internal/deploy/` — Docker/Compose deployment and uninstall logic.
- `internal/system/` — host health checks and environment readiness.
- `internal/util/` — shared application helpers.

### Core domain layer (`core/`)
- `core/api/` — Hub API client.
- `core/discovery/` — recommendation engine and discovery pipeline.
- `core/scanner/` — host inspection engine for native and containerized services.
- `core/schema/` — manifest contracts and deployment models.

This layout keeps command code thin and concentrates domain systems in `core/`, while `internal/` remains the runtime host and orchestration layer.

## Core subsystems

### `core/scanner`
Discovers the host's attack surface with OS-native inspection.
- parses `/proc/net/tcp(6)` and `/proc/[pid]/fd`
- correlates ports with processes and containers
- integrates Docker socket data for container-aware discovery

### `core/discovery`
Builds sensor recommendations from manifests and host state.
- matches manifests against discovered services
- filters already deployed sensors
- renders deployment templates for a targeted deception strategy

### `core/api`
Fetches manifest data and interacts with the HoneyWire Hub for node linking and state.

### `core/schema`
Defines shared manifest and deployment contracts used across discovery and deployment.

## Usage

Build and run the Wizard on the host:

```bash
make build
./build/wizard
```

Run the module test suite:

```bash
make test
```

## Cleanup

To tear down managed deception infrastructure:

```bash
./build/wizard --uninstall
```

This safely removes the managed compose deployment and leaves the host in a clean state.
