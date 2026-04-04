# HoneyWire Official Sensor: Network Scan Detector

The Network Scan Detector is a network-layer IDS sensor that silently watches raw TCP SYN traffic and alerts when a single source probes too many closed ports in a short time window.

## Features
* **Passive detection:** No ports are opened by the sensor.
* **Raw packet visibility:** Uses Scapy to observe SYN probes on the host network.
* **Threshold alerting:** Fires when one source hits multiple unique ports quickly.
* **HoneyWire-native reporting:** Sends standardized JSON events to the Hub.

## Configuration
Copy `.env.example` to `.env` and update the core HoneyWire settings.

| Variable | Description | Default |
|---|---|---|
| `HW_HUB_ENDPOINT` | HoneyWire Hub API endpoint | `http://192.168.1.100:8080` |
| `HW_HUB_KEY` | HoneyWire API key | `super_secret_key_123` |
| `HW_SENSOR_ID` | Unique sensor identifier | `network-scan-detector-01` |
| `HW_SEVERITY` | Alert severity | `high` |
| `HW_SCAN_THRESHOLD` | Unique closed ports required to trigger | `5` |
| `HW_SCAN_WINDOW` | Window in seconds for detection | `5` |
| `HW_IGNORE_PORTS` | Comma-separated ports to ignore as legitimate | `80,443` |

## Deployment
This sensor requires host networking and raw packet capture capability.

Example `docker-compose.yml`:

```yaml
services:
  network-scan-detector:
    build:
      context: ../../../
      dockerfile: Sensors/official/NetworkScanDetector/Dockerfile
    container_name: hw-network-scan-detector
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
  "event_type": "network_scan_detected",
  "severity": "high",
  "source": "10.0.0.99",
  "target": "Multiple Ports",
  "metadata": {"ports_hit": [21, 22, 23, 25, 80], "count": 5, "window_sec": 5}
}
```
