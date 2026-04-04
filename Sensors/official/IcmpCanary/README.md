# HoneyWire Official Sensor: Ping Canary

The Ping Canary sensor is a network-layer deception trap that listens for ICMP Echo Requests directed at the container's own IP address. It is designed to be deployed on an isolated IP where absolutely zero legitimate traffic should occur.

## Features
* **Passive ICMP detection:** Monitors for ping requests on a dark IP.
* **Low-interaction detection:** Detects ping sweeps and reconnaissance without opening any real service ports.
* **HoneyWire-native reporting:** Sends standardized JSON events to the HoneyWire Hub.

## Configuration
Copy `.env.example` to `.env` and update the core HoneyWire settings.

| Variable | Description | Default |
|---|---|---|
| `HW_HUB_ENDPOINT` | HoneyWire Hub API endpoint | `http://192.168.1.100:8080` |
| `HW_HUB_KEY` | HoneyWire API key | `super_secret_key_123` |
| `HW_SENSOR_ID` | Unique sensor identifier | `ping-canary-01` |
| `HW_SEVERITY` | Alert severity | `high` |
| `HW_PING_CANARY_IFACE` | Network interface to monitor | (default Scapy interface) |

## Deployment
This sensor requires host networking and raw packet capabilities.

Example `docker-compose.yml`:

```yaml
services:
  ping-canary:
    build:
      context: ../../../
      dockerfile: Sensors/official/IcmpCanary/Dockerfile
    container_name: hw-ping-canary
    network_mode: "host"
    cap_add:
      - NET_RAW
    env_file:
      - .env
```

Run:

```bash
docker-compose up -d
```

## Event Example
```json
{
  "event_type": "icmp_ping_received",
  "severity": "high",
  "source": "192.168.1.45",
  "target": "ICMP Listener",
  "metadata": {"packet_size": 64, "ttl": 64}
}
```
