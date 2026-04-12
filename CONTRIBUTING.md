[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-WIP-yellow.svg)]()

# Contributing to HoneyWire

Welcome to HoneyWire! We are building a centralized, high-fidelity security and deception ecosystem for homelabs and SMBs. 

To keep the ecosystem stable and structurally secure, all community-submitted sensors must adhere to a strict set of DevSecOps rules. We treat sensors as **isolated, unprivileged microservices**. 

## The Golden Rules of Sensors

1. **Strict Sandboxing (Docker Only):** Every sensor must include a `Dockerfile`. We strongly enforce the use of minimal, hardened base images (like Distroless) running as non-root users (`UID 65532`) with all Linux kernel capabilities dropped (`cap_drop: ALL`).
2. **Zero Blast Radius:** Your sensor must not crash or overwhelm the main Hub. All communication must happen asynchronously via HTTP POST requests containing JSON.
3. **No Hardcoding:** All configurations (Ports, API keys, file paths, thresholds) must be handled dynamically via environment variables.

## How to Submit a New Sensor

### 1. Use the Official Template
Copy the [`Sensors/templates/go-sensor-template/`](./Sensors/templates/go-sensor-template/) folder and rename it to your sensor's name inside the [`Sensors/community/`](./Sensors/community/) directory. 
*While you can technically build a custom sensor in any language, **pure Go is the official standard** for HoneyWire due to its minimal footprint, concurrency models, and ability to compile statically without external dependencies.*

### 2. Follow the JSON Contract (v1.0)
Your sensor must POST a payload to the Hub (`HW_HUB_ENDPOINT`) matching this exact schema:

```json
{
  "contract_version": "1.0",
  "severity": "critical",
  "event_trigger": "what_just_happened",
  "source": "104.28.19.12",
  "target": "Auth Gateway",
  "sensor_id": "provided-by-env",  
  "details": {
    "ip": "192.168.1.5",
    "custom_data": "anything you want"
  }
}
```
*(Note: If you use the official HoneyWire Go SDK provided in the template, this formatting is handled for you automatically).*

### 3. Implement Test Mode (Required for CI/CD)
To ensure your code works before merging, our GitHub Actions will build your Docker container and pass `HW_TEST_MODE=true` as an environment variable. 

If this variable is present, your sensor **must immediately send a synthetic payload to the Hub and exit gracefully**. (The HoneyWire Go SDK's `Start()` method handles this natively out-of-the-box).

### 4. Provide Thorough Documentation
Provide a `README.md` within your sensor directory containing:
  * **Technical Overview:** Purpose of the sensor and the "lure" or monitoring it provides.
  * **Environment Reference:** A table of all `HW_` configuration variables.
  * **Deployment Example:** A secure `docker-compose.yml` snippet and a `.env.example`.
  * **Security Architecture:** An explicit breakdown of the capability drops, user privileges, and isolation techniques utilized by your container to ensure the host remains safe.

## Review Process
Once you open a Pull Request:
1. **Automated Security Scanning:** GitHub Actions will run **Trivy** to scan your Docker image for OS and library vulnerabilities, and **CodeQL** to perform static code analysis for memory leaks and insecure patterns.
2. **Functional Testing:** GitHub Actions will automatically build your Docker container and test it against a Mock Hub using `HW_TEST_MODE=true` to verify contract compliance.
3. **Manual Review:** A core maintainer will manually review the code, specifically checking for malicious intent, proper capability stripping, and blast-radius risks. PRs that fail automated testing or security scanning will not be reviewed.