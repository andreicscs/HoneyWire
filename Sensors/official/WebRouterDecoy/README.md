# HoneyWire Official Sensor: Web Router Decoy

The Web Router Decoy is a web honeypot designed to detect credential stuffing, automated web scanners, and targeted administrative panel attacks. It serves a deceptive, router login page. When an attacker attempts to log in, the sensor captures their IP, user agent, and attempted credentials, silently reports them to the HoneyWire Hub, and safely returns a "401 Unauthorized" response to keep them guessing.

## Features
* **Zero-Setup SDK Integration:** Natively built on the HoneyWire Go SDK.
* **Dynamic Brand Variable:** Automatically injects the specified router brand (e.g., Cisco, Netgear, ASUS) directly into the HTML template to make the trap more convincing.
* **Distroless Container:** Compiled as a statically-linked binary running inside a hardened, unprivileged `:nonroot` Distroless Docker image to prevent container breakouts.

## Configuration

All configuration is handled via Environment Variables. Copy the `.env.example` file to `.env` before running.

### Core Ecosystem Variables (Required)
| Variable | Description | Example |
|---|---|---|
| `HW_HUB_ENDPOINT` | The URL of your central HoneyWire Hub. | `http://127.0.0.1:8080` |
| `HW_HUB_KEY` | The shared secret API key to authenticate with the Hub. | `super_secret_key_123` |
| `HW_SENSOR_ID` | A unique identifier for this specific trap. | `web-decoy-01` |
| `HW_SEVERITY` | Alert severity sent to the Hub (`info` to `critical`). | `critical` |

### Sensor-Specific Variables
| Variable | Description | Default |
|---|---|---|
| `HW_BIND_PORT` | The TCP port the fake web server will listen on. | `8080` |
| `HW_ROUTER_BRAND` | The brand name injected into the fake login page. | `Netgear` |

## Deployment

It is highly recommended to deploy this sensor using the provided docker-compose.yml configuration and the provided Dockerfile (if building from source). The compose file automatically applies the strict network and kernel capability rules required for safe execution.
```Bash
docker compose up -d
```

## Security Architecture

This sensor is architected for extreme resilience against web-based exploits. By utilizing a minimal attack surface and enforcing strict container sandboxing.

**Core Defense-in-Depth Measures:**
* **Framework-Free Execution:** Built purely on Go's native `net/http` library, eliminating the massive attack surface and supply-chain risks associated with heavy third-party web frameworks (like FastAPI, Flask, or Express).
* **Unprivileged Execution:** Runs entirely as a non-root user (`UID 65532`), preventing system-level modifications even in the event of a container breach.
* **Kernel Capability Stripping:** Drops all Linux kernel capabilities (`cap_drop: ALL`) via the Docker Compose configuration, neutralizing advanced kernel exploitation techniques.
* **Distroless Isolation:** Built on a statically-linked Distroless image. It completely lacks a shell (`/bin/sh`), package managers, or standard Linux utilities (like `curl` or `wget`), leaving attackers with zero tools to download secondary payloads or pivot to the host network.
* **In-Memory Operation:** Processes all payload data exclusively in memory, ensuring zero malicious disk I/O operations occur on the host system.

*Recommendation: For optimal security, always deploy this sensor using the official `docker-compose.yml` and `Dockerfile` to ensure these sandbox protections are strictly enforced by the container runtime.*
