# HoneyWire Wizard - Intelligent Deception CLI

## Overview

The HoneyWire Wizard is a zero-friction, intelligent command-line agent designed to assess a host's attack surface and automatically deploy contextual deception infrastructure (honeypots). 

It uses **Correlated Discovery** to map active processes to network sockets and Docker containers, ensuring zero-noise deployments. The Wizard is **100% agnostic** to the sensors it deploys, acting as a dynamic engine that fetches JSON manifests from the HoneyWire Hub and executes them locally.

## Architecture

### 1. **Scanner Package** (`pkg/scanner/`)
The engine discovers the host's real attack surface using pure, read-only OS parsing (No `exec.Command` injection risks).

- **ProcScanner**: Parses `/proc/net/tcp(6)` and correlates socket inodes to `/proc/[pid]/fd` to build a perfect map of native processes listening on ports.
- **Docker Integration**: Gracefully queries the local Docker socket (`/var/run/docker.sock`) to "X-Ray" past `docker-proxy` and map native ports directly to their corresponding container images.

### 2. **State & Safety** (`pkg/state/`)
The Wizard executes a pre-flight safety matrix before recommending deployments:
- **Idempotency**: Parses existing `/opt/honeywire/docker-compose.yml` configurations to prevent duplicate deployments and skip ports already managed by HoneyWire.
- **Resource Constraints**: Parses `/proc/meminfo` to warn the operator if the host lacks the RAM necessary to safely deploy multiple sensors.

### 3. **The API-Driven Engine** (`pkg/api/`)
The Wizard does not hardcode sensor logic. It requests the official `manifests.json` from the HoneyWire Registry. The manifest provides the Wizard with the exact heuristics needed to evaluate the host, and the explicit `InitContainers`, `VolumeMounts`, and `EnvVars` needed to deploy the trap securely. 

This guarantees the Wizard binary never needs to be recompiled when new sensors are added to the ecosystem.

### 4. **Dynamic Fingerprint Evasion**
The template engine includes randomized helper functions (`randWebFile`, `randDBFile`). When dropping tripwires or lures (like the File Canary), the filenames are randomized (e.g., `wp-config.old.php` vs `config.bak.php`) ensuring no two HoneyWire deployments share the exact same forensic fingerprint.

## Interactive TUI

The Wizard features a frictionless, 3-step deployment flow built for operators:
1. **Analyze**: Scans the host and prints the discovered attack surface (Native vs Containerized).
2. **Formulate**: Generates a Dry-Run of the Infrastructure-as-Code.
3. **Execute**: An interactive `[Y/n/edit]` prompt.

**The Drill-Down UX:**
Operators can type `edit` to enter the customization menu. From here they can remove sensors (e.g., `1,3`) or inspect them (e.g., `i 2`). Inspecting a sensor reveals exactly *why* the Wizard chose it, citing the specific PID and Port that triggered the heuristic.

## Usage

Build and run the agent directly on the target Linux host:

```bash
make build
./build/wizard --registry [https://raw.githubusercontent.com/andreicscs/HoneyWire/main/Sensors/official/manifests.json](https://raw.githubusercontent.com/andreicscs/HoneyWire/main/Sensors/official/manifests.json)
```

## Decommissioning

To safely tear down the deception infrastructure and wipe all forensic trails from the host:
```bash
./wizard destroy
```
This safely executes `docker compose down -v` and recursively deletes the `/opt/honeywire` deployment directory.