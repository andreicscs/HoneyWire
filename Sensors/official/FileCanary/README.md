# HoneyWire Official Sensor: File Canary

The File Canary is a low-noise Host-Level Tamper and Ransomware detector. It safely provisions discrete decoy files across your system and explicitly maps them into an unprivileged sandbox using individual read-only mounts. 

## Features
* **Zero-Setup SDK Integration:** Natively built on the HoneyWire Go SDK.
* **Tamper & Ransomware Tripwires:** Triggers on `IN_MODIFY`, `IN_DELETE_SELF`, and `IN_ATTRIB` to instantly alert on encryption routines or cleanup scripts tampering with bait artifacts.
* **Strict Initializer Sandbox:** Validates paths, drops symlinks, ensures strict absolute targeting, and automatically creates missing artifacts without altering surrounding directory permissions.
* **Explicit File Isolation:** Eschews broad directory tracking in favor of explicit individual file monitoring to dramatically cut false positives.
* **Optional Access Detection:** Toggle `HW_ALERT_ON_OPEN` to catch reconnaissance activity like `cat` and `less` via `IN_OPEN` / `IN_ACCESS` events.

## Configuration

Configuration is managed through an `.env` file located in the same directory as the `docker-compose.yml`.

### Core Ecosystem Variables
| Variable | Description | Example |
|---|---|---|
| `HW_HUB_ENDPOINT` | The URL of your central HoneyWire Hub. | `http://127.0.0.1:8080` |
| `HW_HUB_KEY` | The Node Key to authenticate with the Hub. | `super_secret_key_123` |
| `HW_SENSOR_ID` | A unique identifier for this specific trap. | `file-canary-01` |
| `HW_SEVERITY` | Alert severity sent to the Hub (`info` to `critical`). | `critical` |
### Sensor-Specific Variables
| Variable | Description | Default |
|---|---|---|
| `HW_DECOY_FILES` | Comma-separated list of absolute file paths to deploy and monitor. | `/var/www/html/.backup-config.php, /opt/.env.backup` |
| `HW_ALERT_ON_OPEN` | Generates alerts when files are simply read/opened. | `false` |
## Deployment
It is highly recommended to deploy this sensor using the provided `docker-compose.yml` configuration and the provided `Dockerfile` (if building from source). The compose file orchestrates the required permission fixers and read-only mounts automatically.

```bash
docker compose up -d
```

## Security Architecture

This sensor is architected for extreme resilience against exploits. By utilizing a minimal attack surface and enforcing strict container sandboxing, it ensures the host filesystem remains protected.

**Core Defense-in-Depth Measures:**
* **Unprivileged Execution:** Runs entirely as a non-root user (`UID 65532`), preventing system-level modifications even in the event of a container breach.
* **Explicit Single-File Mounts:** Completely abandons broad directory mounting. Individual files are explicitly mounted with `read_only: true` flags, leaving zero surface area for recursive filesystem traversal.
* **Symlink & Traversal Protections:** The initialization provisioner actively refuses to follow symlinks, utilize relative paths, or touch directories, entirely mitigating path traversal injections.
* **Kernel Capability Stripping:** Drops all default Linux kernel capabilities (`cap_drop: ALL`) via the Docker Compose configuration, neutralizing advanced kernel exploitation techniques.
* **Distroless Isolation:** Built on a statically-linked Distroless image. It completely lacks a shell (`/bin/sh`), package managers, or standard Linux utilities, leaving attackers with zero tools to pivot to the host.

*Recommendation: For optimal security, always deploy this sensor using the official `docker-compose.yml` and `Dockerfile` to ensure these sandbox protections are strictly enforced by the container runtime.*