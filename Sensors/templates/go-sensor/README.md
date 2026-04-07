# HoneyWire Go Sensor Template

Welcome to the HoneyWire ecosystem! This template contains everything you need to build a blazing-fast, strictly sandboxed custom Go security sensor that natively reports to the HoneyWire Hub.

## How to Build Your Sensor

1. **Copy this folder** to your desired location (e.g., `Sensors/custom/MyNewSensor`).
2. **Initialize the Go Module**: Link your new sensor to the local HoneyWire SDK by running these commands inside your new folder:
   ```bash
   go mod init github.com/honeywire/sensors/mynewsensor
   go mod edit -replace github.com/honeywire/sdk-go=../../../SDKs/go-honeywire
   go mod tidy
   ```
3. **Write your logic** inside `main.go`. The file is heavily commented and shows you exactly where to put your detection loops.
4. **Update `docker-compose.yml` and `Dockerfile`**: Ensure the paths point to your newly named directory.

## Deployment

It is highly recommended to deploy this sensor using the provided `docker-compose.yml` configuration and the provided `Dockerfile` (if building from source). The compose file orchestrates the required security capabilities, user permissions, and network modes automatically.

```bash
docker compose up -d
```

## Security Architecture

This template is architected for extreme resilience against exploits out-of-the-box. By utilizing a minimal attack surface and enforcing strict container sandboxing, it ensures your custom detection logic cannot be weaponized against the host.

**Core Defense-in-Depth Measures:**
* **Unprivileged Execution:** Defaults to running entirely as a non-root user (`UID 65532`), preventing system-level modifications even in the event of a container breach.
* **Kernel Capability Stripping:** Drops all Linux kernel capabilities (`cap_drop: ALL`) via the Docker Compose configuration, neutralizing advanced kernel exploitation techniques.
* **Distroless Isolation:** Built on a statically-linked Distroless image. It completely lacks a shell (`/bin/sh`), package managers, or standard Linux utilities (like `curl` or `wget`), leaving attackers with zero tools to download secondary payloads or pivot to the host network.
* **In-Memory Operation:** Written in pure Go, encouraging in-memory packet or log processing without the need for vulnerable C-bindings or heavy framework dependencies.

*Recommendation: For optimal security, always deploy this sensor using the official `docker-compose.yml` and `Dockerfile` to ensure these sandbox protections are strictly enforced by the container runtime. If your sensor doesn't follow this security standard there should be a clear reason or a feature that requires otherwise.*

## CI/CD Requirement

To ensure your sensor works, our GitHub Actions will run it with `HW_TEST_MODE=true`. The HoneyWire Go SDK handles this automatically! When `hw.Start()` is called in that mode, it fires a synthetic payload to the Hub and gracefully exits. You do not need to write any custom testing logic.