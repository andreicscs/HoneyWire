# HoneyWire Official Sensor: TCP Tarpit

The TCP Tarpit is a high-fidelity, low-interaction honeypot designed to detect network reconnaissance and brute-force attempts. It can act as a "Tarpit," binding to decoy ports and intentionally stalling attackers to waste their time while silently extracting their IP and payload data to report to the HoneyWire Hub, or instantly close the connection and report the IP to the Hub.

## Features
* **Zero-Setup SDK Integration:** Natively built on the HoneyWire Go SDK.
* **Massive Concurrency:** Powered by Go routines and channels, capable of trapping thousands of automated bots simultaneously with microscopic memory overhead.
* **Tarpit Modes:** Supports `hold` (silent stall), `echo` (repeat data back), or `close` (immediate drop).
* **Forensic Capture:** Safely buffers up to 10 lines of payload data without risking memory exhaustion.
* **Distroless Container:** Compiled as a statically-linked binary running inside a hardened, unprivileged `:nonroot` Distroless Docker image to prevent container breakouts.

## Configuration

All configuration is handled via Environment Variables. Copy the `.env.example` file to `.env` before running.

### Core Ecosystem Variables (Required)
| Variable | Description | Example |
|---|---|---|
| `HW_HUB_ENDPOINT` | The URL of your central HoneyWire Hub. | `http://127.0.0.1:8080` |
| `HW_HUB_KEY` | The shared secret API key to authenticate with the Hub. | `super_secret_key_123` |
| `HW_SENSOR_ID` | A unique identifier for this specific trap. | `ssh-tarpit-01` |
| `HW_SEVERITY` | Alert severity sent to the Hub (`info` to `critical`). | `high` |

### Sensor-Specific Variables
| Variable | Description | Default |
|---|---|---|
| `HW_DECOY_PORTS` | Comma-separated list of TCP ports to monitor. | `2222,3306` |
| `HW_TARPIT_MODE` | The behavior of the trap: `hold`, `echo`, or `close`. | `hold` |
| `HW_TARPIT_BANNER` | (Optional) A fake service banner to send on connect. | `SSH-2.0-OpenSSH_8.2p1\r\n` |

## Tarpit Modes Explained
* **`hold` (Default):** The sensor accepts the connection but sends nothing. It holds the TCP socket open as long as possible (up to 1 hour), dripping empty bytes to drain the attacker's resources and slow down automated scanners like Nmap or brute-force tools.
* **`echo`:** The sensor acts as an echo server, repeating whatever the attacker sends back to them. Useful for confusing automated scripts.
* **`close`:** The sensor logs the connection, captures the initial payload, and forcefully closes the socket. 

## Deployment

It is highly recommended to deploy this sensor using the provided docker-compose.yml configuration and the provided Dockerfile (if building from source). The compose file automatically applies the strict network and kernel capability rules required for safe execution.

```Bash
docker compose up -d
```

## Security Architecture

This sensor is architected for extreme resilience against exploitation. By adhering to the principle of least privilege and enforcing strict resource limits.

**Core Defense-in-Depth Measures:**
* **Kernel Capability Stripping:** Drops all Linux kernel capabilities (`cap_drop: ALL`) via the Docker Compose configuration, neutralizing advanced kernel exploitation techniques.
* **Distroless Isolation:** Built on a statically-linked Distroless image. It completely lacks a shell (`/bin/sh`), package managers, or common Linux utilities (like `curl` or `wget`), leaving attackers with zero tools to pivot if they achieve Remote Code Execution.
* **Concurrency Capping:** Utilizes a native Go buffered channel (semaphore) to strictly cap concurrent connections at `1000`. This prevents attackers from launching a Denial of Service (DoS) attack designed to exhaust the host machine's File Descriptors or RAM.
* **In-Memory Operation:** Processes all payload data exclusively in memory, ensuring zero malicious disk I/O operations occur on the host system.

*Recommendation: For optimal security, always deploy this sensor using the official `docker-compose.yml` and `Dockerfile` to ensure these sandbox protections are strictly enforced by the container runtime.*