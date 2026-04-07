# HoneyWire Official Sensor: Network Scan Detector 

The Network Scan Detector is a low-overhead network sensor designed to silently detect horizontal port scans. By monitoring raw SYN packets directly on the network interface, it identifies scanning activity aimed at closed or unused ports before it ever reaches a firewall or application log.

## Features
* **Zero-Setup SDK Integration:** Natively built on the HoneyWire Go SDK.
* **In-Memory Parsing:** Analyzes raw TCP headers directly in memory.
* **Configurable Thresholds:** Easily adjust how many unique ports must be hit within a specific time window to trigger an alert.
* **Distroless Container:** Compiled as a statically-linked binary running inside a minimal Docker image.

## Configuration

All configuration is handled via Environment Variables.

### Core Ecosystem Variables
| Variable | Description | Example |
|---|---|---|
| `HW_HUB_ENDPOINT` | The URL of your central HoneyWire Hub. | `http://127.0.0.1:8080` |
| `HW_HUB_KEY` | The shared secret API key to authenticate with the Hub. | `super_secret_key_123` |
| `HW_SENSOR_ID` | A unique identifier for this specific trap. | `scan-detector-01` |
| `HW_SEVERITY` | Alert severity sent to the Hub (`info` to `critical`). | `critical` |

### Sensor-Specific Variables
| Variable | Description | Default |
|---|---|---|
| `HW_SCAN_THRESHOLD` | Number of unique ports that must be hit to trigger an alert. | `5` |
| `HW_SCAN_WINDOW` | The time window (in seconds) to track the threshold. | `5` |
| `HW_IGNORE_PORTS` | Comma-separated ports to ignore (e.g., actual open services). | `80,443` |

## Deployment

It is highly recommended to deploy this sensor using the provided docker-compose.yml configuration and the provided Dockerfile (if building from source). The compose file automatically applies the strict network and kernel capability rules required for safe execution.

```bash
docker compose up -d
```

## Security Architecture

This sensor is architected for extreme resilience against exploits. By utilizing a minimal attack surface and enforcing strict container sandboxing, it safely handles raw network traffic.

**Core Defense-in-Depth Measures:**
* **Raw Socket Isolation:** Bypasses heavy NIDS frameworks by interacting directly with network packets in pure Go, eliminating external C-library vulnerabilities.
* **Least Privilege Execution:** Runs as container root strictly to bind the raw socket, but relies on capability dropping to prevent privilege escalation.
* **Kernel Capability Stripping:** Drops all default Linux kernel capabilities (`cap_drop: ALL`) and only adds back `NET_RAW`, ensuring the sensor can read packets but cannot modify the system.
* **Distroless Isolation:** Built on a statically-linked Distroless image. It completely lacks a shell (`/bin/sh`), package managers, or standard Linux utilities, leaving attackers with zero tools to pivot to the host network.
* **In-Memory Operation:** Processes all payload data exclusively in memory, ensuring zero malicious disk I/O operations occur on the host system.

*Recommendation: For optimal security, always deploy this sensor using the official `docker-compose.yml` and `Dockerfile` to ensure these sandbox protections are strictly enforced by the container runtime.*