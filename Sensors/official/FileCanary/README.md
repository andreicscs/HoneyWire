# HoneyWire Official Sensor: File Canary

The File Canary acts as both a Honeypot and a File Integrity Monitor (FIM). It watches a specified directory or file on the host machine. If an attacker modifies, deletes, or drops a file into the watched area, the sensor immediately fires an alert to the HoneyWire Hub.

## Features
* **Zero-Setup SDK Integration:** Natively built on the HoneyWire Go SDK.
* **Dual-Mode Operation:** Can monitor highly sensitive, real system files (FIM) or act as a standalone honeypot directory (Trap).
* **Safe Permissions Handling:** Uses Access Control Lists (ACLs) to securely read target directories without altering their original host ownership.
* **Failsafe Mounts:** Designed to halt deployment if the target directory doesn't exist, preventing false-positive monitoring.

## Configuration

Configuration is managed through an `.env` file located in the same directory as the `docker-compose.yml`.

### Core Ecosystem Variables
| Variable | Description | Example |
|---|---|---|
| `HW_HUB_ENDPOINT` | The URL of your central HoneyWire Hub. | `http://127.0.0.1:8080` |
| `HW_HUB_KEY` | The shared secret API key to authenticate with the Hub. | `super_secret_key_123` |
| `HW_SENSOR_ID` | A unique identifier for this specific trap. | `file-canary-01` |
| `HW_SEVERITY` | Alert severity sent to the Hub (`info` to `critical`). | `critical` |
### Sensor-Specific Variables
| Variable | Description | Default |
|---|---|---|
| `TRAP_PATH` | The physical path on the host machine to monitor. | `./trap_directory` or `/etc/passwd` |
## Deployment
It is highly recommended to deploy this sensor using the provided `docker-compose.yml` configuration and the provided `Dockerfile` (if building from source). The compose file orchestrates the required permission fixers and read-only mounts automatically.

```bash
docker compose up -d
```

## Security Architecture

This sensor is architected for extreme resilience against exploits. By utilizing a minimal attack surface and enforcing strict container sandboxing, it ensures the host filesystem remains protected.

**Core Defense-in-Depth Measures:**
* **Unprivileged Execution:** Runs entirely as a non-root user (`UID 65532`), preventing system-level modifications even in the event of a container breach.
* **Read-Only Mounts:** The target directory is mounted with strict `read_only: true` flags, ensuring the container cannot write to or modify the host files it is monitoring under any circumstances.
* **ACL Integration:** Instead of changing host file ownership, a temporary initialization container uses `setfacl` to grant the non-root user specific, read-only traverse rights, keeping your original host permissions completely intact.
* **Kernel Capability Stripping:** Drops all default Linux kernel capabilities (`cap_drop: ALL`) via the Docker Compose configuration, neutralizing advanced kernel exploitation techniques.
* **Distroless Isolation:** Built on a statically-linked Distroless image. It completely lacks a shell (`/bin/sh`), package managers, or standard Linux utilities, leaving attackers with zero tools to pivot to the host.

*Recommendation: For optimal security, always deploy this sensor using the official `docker-compose.yml` and `Dockerfile` to ensure these sandbox protections are strictly enforced by the container runtime.*