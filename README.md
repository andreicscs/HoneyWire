[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-WIP-yellow.svg)]()

## 📋 Table of Contents
- [Overview](#honeywire)
- [Screenshots](#screenshots)
- [The Universal Event Standard](#-the-universal-event-standard-bring-your-own-sensor)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start Guide](#-quick-start-guide)
- [Security Notes](#security-notes)
- [Tech Stack](#tech-stack)
- [Versioning and API Reference](#versioning-and-api-reference)
- [Operational Checklist](#operational-checklist)

---
# HoneyWire

**HoneyWire Sentinel** is a lightweight, Distributed High-Signal Security Early-Warning System, designed for internal networks. It replaces the "magnifying glass" approach of traditional SIEMs, which often drown analysts in false positives by surveilling legitimate traffic, with a High-Fidelity Tripwire model. Place a sensor that does what you need exactly where you want, for example:
  - Production Tripwires: Sound the alarm when active services are being poked in ways they shouldn't be. By placing a sensor on a sensitive file that should never be read or a service port that should never be accessed, you identify intruders by their deviation from the "authorized path."
  - Synthetic Deception: Deploy lures like the [ICMP Canary](./Sensors/official/IcmpCanary/) or [Network Scan Detector](./Sensors/official/NetworkScanDetector/) to act as decoys. Since these sensors provide no legitimate business value, 100% of their traffic is actionable intelligence.

If it is tripped, something is wrong, set up multiple and you start to have a pretty clear idea of the lateral movement of an intruder. No tuning, no noise, just instant forensics.

---

## Screenshots

### Main Dashboard
![DashboardDark](Screenshots/dashboardDark.png)

![DashboardLight](Screenshots/dashboardLight.png)

---
## 🔌 The Universal Event Standard (Bring Your Own Sensor)

[**Community Sensors**](./Sensors/community/)

The true power of HoneyWire is that the Hub is **completely sensor-agnostic**. You are not limited to the included official sensors. 

By adhering to the **HoneyWire Event Standard V1.0**, you can write a script in *any* language (Bash, Go, Rust, Python) to monitor *anything*, and the Sentinel UI will dynamically parse, syntax-highlight, and render your forensic data. 

Whether it is a **Deep Packet Inspection (DPI)** engine, a **DNS sinkhole**, a **Canary Token** embedded in a PDF, an **Email Honeypot**, or a simple **TCP Port Tripwire**, just POST this JSON to the Hub:

```json
{
  "contract_version": "1.0",
  "severity": "critical",
  "event_trigger": "malformed_jwt_detected",
  "source": "104.28.19.12",
  "target": "Auth Gateway",
  "sensor_id": "core-dpi-engine",  
  "details": {
    "protocol": "TCP",
    "headers_stripped": true,
    "payload_sample": [
      "Authorization: Bearer eyJhbG... [TRUNCATED]",
      "User-Agent: curl/7.64.1"
    ]
  }
}
```
> Note: If you build your sensor using the official HoneyWire Go SDK, this JSON formatting and delivery is handled for you automatically.

*The Hub's frontend automatically translates arrays into syntax-highlighted code blocks and primitive values into clean detail tags.*

---

## Features

- **The Sentinel Hub UI:** A fully responsive dashboard featuring Dark/Light mode, real-time Chart.js threat distribution, and dynamic forensic payload inspection.
- **Suite of Official Sensors:** Includes native [TCP Tarpit](./Sensors/official/TcpTarpit/), [Web Router Decoy](./Sensors/official/WebRouterDecoy/), [File Canary (FIM)](./Sensors/official/FileCanary/), [ICMP Canary](./Sensors/official/IcmpCanary/), and [Network Scan Detector](./Sensors/official/NetworkScanDetector/).

---

## Architecture

HoneyWire is split into three independent microservices:

1. `/Hub`: The central brain. A pure Go binary running an embedded SQLite database and the web dashboard. It runs as a non-root user inside a Distroless container, safely mounting data to a dedicated volume.
2. `/Sensors`: The decoy nodes. Statically-linked Go binaries that listen on vulnerable ports, trap attackers, and securely POST intrusion data back to the Hub.
3. `/SDKs`: Official libraries (like `sdk-go`) that handle secure Hub communication so community developers can easily build new sensors.

---

## 🚀 Quick Start Guide

Deploying HoneyWire takes less than 60 seconds using our pre-built GitHub Container images. No compiling is required.

Create a new directory on your server, and create two files: `docker-compose.yml` and `.env`.

### 1. The `docker-compose.yml`
```yaml
version: '3.8'

services:
  # 1. THE PERMISSION FIXER: Runs once to ensure the Hub can write to the data volume
  permission-fixer:
    image: alpine:latest
    command: sh -c "chown -R 65532:65532 /data"
    volumes:
      - ./honeywire_data:/data

  # 2. THE HUB: The central Go-based dashboard and API
  hub:
    image: ghcr.io/andreicscs/honeywire-hub:latest
    container_name: honeywire-hub
    restart: unless-stopped
    ports:
      - "${HW_PORT:-8080}:${HW_PORT:-8080}"
    volumes:
      - ./honeywire_data:/data
    depends_on:
      permission-fixer:
        condition: service_completed_successfully
    user: "65532:65532"
    read_only: true
    cap_drop: ["ALL"]
    security_opt: ["no-new-privileges:true"]

    env_file: 
      - .env

  # 3. EXAMPLE SENSOR: The TCP Tarpit (See /Sensors for more)
  tcp-tarpit:
    image: ghcr.io/andreicscs/honeywire-tcptarpit:latest
    container_name: hw-tcp-tarpit
    restart: unless-stopped
    network_mode: "host" # Required to capture true source IPs
    user: "0:0" # Required to bind to low ports
    # Security hardening
    cap_drop: ["ALL"]
    cap_add: ["NET_BIND_SERVICE"]
    read_only: true
    security_opt: ["no-new-privileges:true"]

    env_file: 
      - .env
```

### 2. The `.env` Configuration
```ini
# ==========================================
# HUB CONFIGURATION
# ==========================================
# Secret key used by sensors to authenticate with the Hub
HW_HUB_KEY=change_this_to_a_secure_random_string

# Optional: Protect the Web UI (Leave blank for no password)
HW_DASHBOARD_PASSWORD=admin

# Optional: Push Notifications
HW_NTFY_URL=https://ntfy.sh/your_private_topic
# HW_GOTIFY_URL=https://gotify.example.com/message
# HW_GOTIFY_TOKEN=your_token

# ==========================================
# SENSOR EXAMPLE: TCP TARPIT
# ==========================================
# Point this to your Hub's IP and Port
HW_HUB_ENDPOINT=http://127.0.0.1:8080
HW_SENSOR_ID=tarpit-01

# Ports to monitor, behavior mode, and fake service banner
HW_DECOY_PORTS=22,2222,3306
HW_TARPIT_MODE=hold
HW_SEVERITY=high
HW_TARPIT_BANNER=SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1\r\n
```

### 3. Start the Trap
Run the following command to pull the images and start the honeypot:
```bash
docker compose up -d
```
Access the dashboard at `http://localhost:8080` (or your server's IP).

---

### 4. Testing the Trap

Once your containers are up, the Tarpit sensor should appear as `ONLINE` in the **Fleet Health** section of the dashboard within 30 seconds.

To verify the detection loop, use `netcat` from a different machine (or a different terminal) to trigger the decoy:

```bash
# Connect to your decoy port (e.g., 2222) at localhost (or your server's IP).
nc localhost 2222
```

1. **Observe the Lure:** If `HW_TARPIT_MODE` is set to `hold` or `echo`, you will see your fake service banner immediately.
2. **Interact:** The connection will be intentionally stalled (Tarpit). Type a string (e.g., `admin` or `exploit_payload`) and press Enter.
3. **Close:** Press `Ctrl+C` to terminate the test connection.
4. **Verify Capture:**
   - Check the HoneyWire Dashboard; the event, your Source IP, and the payload will appear instantly.
   - If configured, you will receive a push notification on your mobile device.


---

## Security Notes
* **API Secret:** Ensure your `HW_HUB_KEY` is strong and identical on both the Hub and the Sensors. The Hub will reject any payloads with mismatched keys. We will eventually implement automatic API key generation from the Hub for each sensor.
* **System Arming:** You can toggle the "System Armed" button in the Hub UI to temporarily disable push notifications while doing internal network maintenance or vulnerability scanning.
* **Container Hardening:** HoneyWire utilizes `gcr.io/distroless/static-debian12:nonroot`. We follow the principle of least privelege to make sure that if a container is compromised, the blast is contained.
* **Distributed Deployment:** It is highly recommended to run the Hub and its Sensors on separate physical or virtual machines. If an attacker compromises a sensor node, they should not have immediate local access to the centralized Hub.
* ! **Encryption (HTTPS):** We **do not** yet implement HTTPS as this project is a work in progress. 
It is important to Always serve the Hub Web GUI and API over HTTPS using a reverse proxy (like Nginx, Caddy, or Traefik). Failure to do so exposes your `HW_HUB_KEY` and `HW_DASHBOARD_PASSWORD` to network sniffing.

---

## Tech Stack
* **Backend:** Go 1.25, `net/http` (Standard Library), SQLite (Pure Go Driver)
* **Frontend:** HTML5, TailwindCSS, Alpine.js, Chart.js
* **Infrastructure:** Docker, Docker Compose, Distroless Linux Sandbox

---

## Versioning and API Reference

- HoneyWire uses a single source of truth version file: `VERSION` in the repo root.
- The runtime version is exposed via an env override: `HW_VERSION` (Hub + Sensors), which defaults to `VERSION`.
- `Hub` endpoint:
  - `GET /api/v1/version` → returns `{ "version": "1.0.0" }`
- API docs file: [📖 API.md](./Docs/API.md) with full backend route reference and sample payloads.

---

## Operational Checklist
- [x] Set `HW_HUB_KEY` for all components.
- [x] Set optional `HW_DASHBOARD_PASSWORD`.
- [x] Rebuild/redeploy containers after any version bump in `VERSION` or environment variable changes.
