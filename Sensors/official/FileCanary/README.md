# HoneyWire Official Sensor: File Canary

The File Canary monitors a mounted host directory for file tampering events like modifications, deletions, and renames. It is intended for use with a read-only mounted lure directory containing attractive files.

## Features
* **Host-level deception:** Monitors a mounted directory for suspicious local activity.
* **File event capture:** Reports `modified`, `deleted`, and `moved` file system events.
* **Minimal blast radius:** The container only needs read access to the watched directory.
* **HoneyWire-native reporting:** Sends standardized JSON events to the Hub.

## Configuration
Copy `.env.example` to `.env` and update the core HoneyWire settings.

| Variable | Description | Default |
|---|---|---|
| `HW_HUB_ENDPOINT` | HoneyWire Hub API endpoint | `http://192.168.1.100:8080` |
| `HW_HUB_KEY` | HoneyWire API key | `super_secret_key_123` |
| `HW_SENSOR_ID` | Unique sensor identifier | `file-canary-01` |
| `HW_SEVERITY` | Alert severity | `critical` |
| `HW_HONEY_DIR` | Mounted directory to watch | `/honey_dir` |
| `HW_POLLING_INTERVAL` | Observer sleep interval in seconds | `1` |

## Deployment
Mount the folder you want to watch into the container as a read-only volume.

Example `docker-compose.yml`:

```yaml
services:
  file-canary:
    build:
      context: ../../../
      dockerfile: Sensors/official/FileCanary/Dockerfile
    container_name: hw-file-canary
    env_file:
      - .env
    volumes:
      - /mnt/finance_share:/honey_dir:ro
```

Run:

```bash
docker-compose up -d
```

## Event Example
```json
{
  "event_type": "file_tampered",
  "severity": "critical",
  "source": "Unknown (Local OS)",
  "target": "/honey_dir/AWS_Root_Keys.csv",
  "metadata": {
    "action": "File Modified/Encrypted",
    "timestamp_os": "1697284411.5"
  }
}
```
