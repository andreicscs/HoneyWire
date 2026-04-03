# HoneyWire Official Sensor: TCP TripWire 🕸️

The TCP TripWire is a high-fidelity, low-interaction honeypot designed to detect network reconnaissance and brute-force attempts. It acts as a "Tarpit," binding to decoy ports and intentionally stalling attackers to waste their time while silently extracting their IP and payload data to report to the HoneyWire Hub.

## Features
* **Zero-Setup SDK Integration:** Natively built on the HoneyWire Python SDK.
* **Tarpit Modes:** Supports `hold` (silent stall), `echo` (repeat data back), or `close` (immediate drop).
* **Forensic Capture:** Safely buffers up to 10 lines of payload data without risking memory exhaustion.
* **Distroless Container:** Runs as a hardened, Distroless Docker image to prevent container breakouts.

## Configuration

All configuration is handled via Environment Variables. Copy the `.env.example` file to `.env` before running.

### Core Ecosystem Variables (Required)
| Variable | Description | Example |
|---|---|---|
| `HW_HUB_ENDPOINT` | The URL of your central HoneyWire Hub. | `http://192.168.1.100:8080` |
| `HW_HUB_KEY` | The shared secret API key to authenticate with the Hub. | `super_secret_key_123` |
| `HW_SENSOR_ID` | A unique identifier for this specific trap. | `dmz-ssh-tarpit-01` |

### Sensor-Specific Variables
| Variable | Description | Default |
|---|---|---|
| `HW_DECOY_PORTS` | Comma-separated list of TCP ports to monitor. | `2222,3306` |
| `HW_TARPIT_MODE` | The behavior of the trap: `hold`, `echo`, or `close`. | `hold` |
| `HW_SEVERITY` | Alert severity sent to the Hub (`info` to `critical`). | `high` |
| `HW_TARPIT_BANNER` | (Optional) A fake service banner to send on connect. | `SSH-2.0-OpenSSH_8.2p1` |

## Tarpit Modes Explained
* **`hold` (Default):** The sensor accepts the connection but sends nothing. It holds the TCP socket open as long as possible, draining the attacker's resources and slowing down automated scanners like Nmap.
* **`echo`:** The sensor acts as an echo server, repeating whatever the attacker sends back to them. Useful for confusing automated scripts.
* **`close`:** The sensor logs the connection, captures the initial payload, and forcefully closes the socket. 

## Deployment

The easiest way to deploy this sensor is via Docker Compose. Note that if you intend to bind to privileged ports (any port under 1024, like Port 22 or 80), the container must run as `root` (which is the default).

### 1. Using Docker Compose
Create a `docker-compose.yml` file:

```yaml
services:
  tcp-tripwire:
    image: ghcr.io/andreicscs/honeywire-tcptripwire:latest
    container_name: hw-tcp-tripwire
    restart: unless-stopped

    # CRITICAL: Binds directly to the host machine's network.
    # 1. Preserves the real source IP of the attacker.
    # 2. Automatically exposes whatever ports are set in HW_DECOY_PORTS.
    network_mode: "host" 
    
    env_file:
      - .env
```
Run it in the background:
```Bash
docker-compose up -d
```

### 2. Testing Locally (Build from Source)
If you are developing or testing locally:
```Bash
docker build -t honeywire-tcptripwire .
docker run --rm --network host --env-file .env honeywire-tcptripwire
```

## Security Note
This sensor uses asyncio with strict memory limits (50KB or 10 lines per connection) and a global semaphore to limit concurrent connections. This ensures the sensor cannot be easily DOS'd or used to exhaust the host machine's File Descriptors.