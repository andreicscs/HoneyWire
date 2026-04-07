# HoneyWire Official Sensor: ICMP Canary

The ICMP Canary (Ping Canary) is a simple, highly effective network tripwire. It listens for ICMP Echo Requests (pings) directed at the host machine. It is best deployed on isolated IPs, darknets, or unused subnets where any inbound ICMP traffic is inherently suspicious.

## Features
* **Zero-Setup SDK Integration:** Natively built on the HoneyWire Go SDK.
* **Raw Socket Listening:** Uses pure Go to listen directly for protocol 1 (ICMP) packets without external C-dependencies.
* **Low Overhead:** Requires minimal CPU and RAM to operate, making it ideal for widespread deployment.
* **Distroless Container:** Compiled as a statically-linked binary running inside a minimal Docker image.

## Configuration

Configuration is managed through Environment Variables.

### Core Ecosystem Variables
| Variable | Description | Example |
|---|---|---|
| `HW_HUB_ENDPOINT` | The URL of your central HoneyWire Hub. | `http://127.0.0.1:8080` |
| `HW_HUB_KEY` | The shared secret API key to authenticate with the Hub. | `super_secret_key_123` |
| `HW_SENSOR_ID` | A unique identifier for this specific trap. | `ping-canary-01` |
| `HW_SEVERITY` | Alert severity sent to the Hub (`info` to `critical`). | `high` |

## Deployment

It is highly recommended to deploy this sensor using the provided `docker-compose.yml` configuration and the provided `Dockerfile` (if building from source). The compose file automatically applies the strict network and kernel capability rules required for safe execution.

```bash
docker compose up -d
```

## Security Architecture

This sensor is architected for extreme resilience against exploits. By utilizing a minimal attack surface and enforcing strict container sandboxing, it safely intercepts raw ICMP traffic.

**Core Defense-in-Depth Measures:**
* **Raw Socket Isolation:** Bypasses heavy NIDS frameworks by interacting directly with network packets in pure Go, eliminating external C-library vulnerabilities.
* **Least Privilege Execution:** Runs as container root strictly to bind the raw socket, relying on container boundaries to limit system access.
* **Kernel Capability Stripping:** Drops all default Linux kernel capabilities (`cap_drop: ALL`) and only adds back `NET_RAW`, ensuring the sensor can intercept pings but cannot modify the host filesystem or OS.
* **Distroless Isolation:** Built on a statically-linked Distroless image. It completely lacks a shell (`/bin/sh`), package managers, or standard Linux utilities, leaving attackers with zero tools to execute secondary payloads.
* **In-Memory Operation:** Processes all packet data exclusively in memory, ensuring zero malicious disk I/O operations occur on the host system.

*Recommendation: For optimal security, always deploy this sensor using the official `docker-compose.yml` and `Dockerfile` to ensure these sandbox protections are strictly enforced by the container runtime.*