# Sensor Development Guide

HoneyWire's architecture separates the central Hub from the deceptive traps (Sensors). This guide explains how to build a custom sensor from scratch, ensuring it complies with HoneyWire's Universal Event Standard and strict security model.

## 1. Anatomy of a Sensor

Every sensor in HoneyWire consists of three core components:
1. **The Application Logic:** Typically a pure Go binary utilizing the HoneyWire SDK.
2. **The Security Sandbox:** A highly restricted `Dockerfile`.
3. **The Deployment Manifest:** A `manifest.json` file that tells the Hub and Wizard how to deploy and configure it.

---

## 2. Using the Go SDK

While you can technically write a sensor in any language using HTTP POST requests, the **HoneyWire Go SDK** (`SDKs/go-honeywire`) is the official standard. It handles authentication, backoff retries, heartbeats, and strict adherence to the data contract.

### Basic Implementation

Instead of starting from scratch, always use the official Go Sensor Template located at `Sensors/templates/go-sensor/`. 
This directory contains a pre-configured `main.go` file with detailed comments, a hardened `Dockerfile`, and a `docker-compose.yml` file that orchestrates the required security capabilities out-of-the-box.

---

## 3. The Deployment Manifest (`manifest.json`)

The manifest is the single source of truth for your sensor. It defines how the Hub renders its UI card, how the Wizard deploys it, and what heuristics trigger a recommendation.

Instead of starting from scratch, base your manifest on the template provided at `Sensors/templates/go-sensor/manifest.json`.

### Key Sections
*   **`heuristics`:** Instructs the Wizard when to recommend this sensor. For example, if you build an Nginx honeypot, you would set `"processes": ["nginx"]` and `"ports": [80, 443]`.
*   **`deployment.env_vars`:** Defines configurable parameters that the Hub UI will generate forms for. If your sensor needs a custom configurable port, define it here.
*   **`deployment.image`:** The Docker image to pull.

*(See `Docs/architecture/dataContracts.md` for the full manifest schema and required fields).*

---

## 4. The Security Sandbox

HoneyWire treats sensors as untrusted, potentially hostile execution units. **If a sensor is compromised, the blast radius must be minimal.**

To ensure compliance, **always use the `Dockerfile` and `docker-compose.yml` or manifest deployment configuration provided in `Sensors/templates/go-sensor/`**. It is pre-configured to enforce the following baseline rules:

1.  **Distroless Base:** The Dockerfile must use a minimal base image like `gcr.io/distroless/static-debian12`. No shells (`/bin/sh`), no package managers, no utilities.
2.  **Prefer unprivileged Execution:** It is better for the to container run as a non-root user (e.g., `USER 65532:65532`).
3.  **Capability Stripping:** The deployment must drop all Linux capabilities. The Hub will automatically append `cap_drop: ["ALL"]` to the Compose output. If your sensor strictly requires a capability, it must be explicitly requested in the manifest and approved by the Hub compiler.
4.  **No New Privileges:** The Hub enforces `security_opt: ["no-new-privileges:true"]` automatically.

---

## 5. Local Testing

Do not boot the entire Hub to test a sensor's data payload. Use the Mock Hub script:

1.  Start the Mock Hub:
    ```bash
    python3 scripts/mock_hub.py
    ```
2.  Run your sensor container locally:
    ```bash
    docker run --rm \
      -e HW_HUB_ENDPOINT=http://<YOUR_LOCAL_IP>:8080 \
      -e HW_HUB_KEY=test_key \
      -e HW_SENSOR_ID=test_sensor \
      -e HW_TEST_MODE=true \
      your-sensor-image:latest
    ```

The Mock Hub will strictly validate your payload against the V1.0 Data Contract and print `[EVENT] OK` or provide specific failure reasons.

## 5.1. Testing Methodologies (CI/CD vs Live)

HoneyWire distinguishes between two distinct ways of testing a sensor payload. Because of the strict security model (no exposed APIs, no shell access), testing relies on process state rather than direct interaction:

1. **Boot-Time CI/CD Testing (`HW_TEST_MODE=true`):**
   When a sensor container boots with this flag, it acts as a short-lived execution check. It synchronously sends a test payload directly to the Hub wire (bypassing internal queues) and immediately shuts down with exit code `0`. This guarantees payload delivery before the container exits. It is strictly used for automated CI pipelines.

2. **Live In-Flight Testing (`SIGUSR1`):**
   When a sensor is actively running in production, it can be tested on-demand via the Wizard CLI (`wizard test <sensor-id>`). The Wizard sends a `SIGUSR1` Unix signal to the target Docker container.
   Unlike the CI test, the signal handler routes the synthetic event into the sensor's asynchronous event queue (`ReportEvent()`). This ensures the test payload traverses the exact same real-world code paths (including exponential backoff, rate limits, and network retries) as a genuine intrusion event, all without interrupting the sensor's active uptime.