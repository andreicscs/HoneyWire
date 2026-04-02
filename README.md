
  [![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
  [![Status](https://img.shields.io/badge/status-WIP-yellow.svg)]()

## 📋 Table of Contents
- [Overview](#-honeywire)
- [Screenshots](#screenshots)
- [The Universal Event Standard](#-the-universal-event-standard-bring-your-own-sensor)
- [Features](#-features)
- [Architecture](#%EF%B8%8F-architecture)
- [Quick Start Guide](#-quick-start-guide)
- [Testing the Trap](#-testing-the-trap)
- [Security Notes](#%EF%B8%8F-security-notes)
- [Tech Stack](#%EF%B8%8F-tech-stack)
- [Versioning and API Reference](#-versioning-and-api-reference)
- [Operational Checklist](#-operational-checklist)

---

# 🕸️ HoneyWire

**HoneyWire Sentinel** is an ultra-lightweight, distributed deception hub and centralized Security Operations Center (SOC). It is designed to deploy silent, asynchronous sensors across multiple servers that detect unauthorized access, trap automated botnets, and report telemetry back to a centralized, high-performance dashboard in real-time.

Developed in collaboration with Gemini (Google AI). 
Architected by Termine Andrea, implementation and boilerplate assisted by LLM.

There are existing lightweight honeypots, but none feature a clean, SaaS-grade dashboard with instant webhooks, and the ones that do are incredibly resource-intensive. This project aims at filling that gap in the cybersecurity software and tools landscape for hobbyists.

---

## Screenshots

### Main Dashboard
![Dashboard](screenshots/dashboard.png)

### Payload Inspector
![Payload Inspector](screenshots/payload-inspector.png)

---

## 🔌 The Universal Event Standard (Bring Your Own Sensor)

The true power of HoneyWire is that the Hub is **completely sensor-agnostic**. You are not limited to the included Tarpit agent. 

By adhering to the **HoneyWire Event Standard**, you can write a script in *any* language (Bash, Go, Rust, Python) to monitor *anything*, and the Sentinel UI will dynamically parse, syntax-highlight, and render your forensic data perfectly. 

Whether it is a **Deep Packet Inspection (DPI)** engine, a **DNS sinkhole**, a **Canary Token** embedded in a PDF, an **Email Honeypot**, or a simple **TCP Port Tripwire**, just POST this JSON to the Hub:

```json
{
  "sensor_id": "core-dpi-engine",
  "sensor_type": "deep_packet_inspection",
  "event_type": "malformed_jwt_detected",
  "severity": "critical",
  "source": "104.28.19.12",
  "target": "Auth Gateway",
  "action_taken": "ip_banned",
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
*The Hub's Alpine.js frontend will automatically translate arrays into syntax-highlighted code blocks and primitive values into clean metadata tags.*

---

## ✨ Features

- **The Sentinel UI:** A fully responsive, glassmorphic dashboard featuring Dark/Light mode, real-time Chart.js threat distribution, and dynamic forensic payload inspection.
- **Included Tarpit Sensor:** A Python `asyncio` tripwire capable of handling thousands of concurrent connections with semantic logging to prevent file descriptor exhaustion.
  - **Three Tarpit Modes:**
    - `hold`: Keeps connections open indefinitely to waste attacker resources.
    - `echo`: Bounces malicious payloads back to the sender.
    - `close`: Terminates connections immediately after logging the IP.
  - **Service Spoofing:** Customizable banners to impersonate legitimate services (e.g., OpenSSH, vsFTPd).
  - **Hardened Security:** Built-in protection against Timing Attacks, XSS (Cross-Site Scripting), and Pass-the-Hash vulnerabilities using ephemeral session tokens and constant-time string comparison. Thread-blocking is mitigated via FastAPI Background Tasks.
  - **Instant Notifications:** Built-in support for ntfy.sh and Gotify mobile alerts.

---

## 🏗️ Architecture

HoneyWire is split into two independent microservices:

1. **The Hub (`/Hub`)**: The central brain. It runs a FastAPI backend, an SQLite database, and the web dashboard. It runs as a `nonroot` user inside a Distroless container, safely mounting data to a dedicated volume.
2. **The Agent (`/Agent`)**: The decoy sensor (or any custom script you write). It listens on vulnerable ports, traps attackers, and securely POSTs the intrusion data back to the Hub using an `API_SECRET`.

---

## 🚀 Quick Start Guide

Deploying HoneyWire takes less than 60 seconds using our pre-built GitHub Container Registry images. No compiling required.

Create a new directory on your server, and create two files: `docker-compose.yml` and `.env`.

### 1. The `docker-compose.yml`

```yaml
services:
  hub:
    image: ghcr.io/andreicscs/honeywire-hub:latest
    container_name: honeywire-hub
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - honeywire_data:/data
    env_file: 
      - .env

  agent:
    image: ghcr.io/andreicscs/honeywire-agent:latest
    container_name: honeywire-agent
    restart: unless-stopped
    network_mode: "host" # Required to accurately capture port scans against the physical machine
    env_file: 
      - .env

volumes:
  honeywire_data:
```
### 2. The .env Configuration
```
# ==========================================
# HUB CONFIGURATION
# ==========================================
# The master password for your fleet to communicate
API_SECRET=super_secret_key_123

# Protect your Web UI (Leave blank for no password)
DASHBOARD_PASSWORD=my_secure_password

# Optional: Push Notifications
NTFY_URL=https://ntfy.sh/your_private_topic
GOTIFY_URL=https://gotify.yourdomain.com/message
GOTIFY_TOKEN=your_app_token
```
```
# ==========================================
# AGENT CONFIGURATION
# ==========================================
# Point this to your Hub's IP address and Port
HUB_URL=http://127.0.0.1:8080

# Must match the Hub's secret
API_SECRET=super_secret_key_123

# Identify this specific sensor and its IP
SENSOR_ID=dmz-node-01
SENSOR_IP=192.168.1.50

# A comma-separated list of fake ports to open
DECOY_PORTS=21,22,2222,3306,8080

# Tarpit Behavior: 'hold', 'echo', or 'close'
TARPIT_MODE=hold

# UI Color Coding: info|low|medium|high|critical
SEVERITY=high

# Fake Service Banner (Use \r\n for line breaks)
# Example SSH: SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1\r\n
# Example FTP: 220 (vsFTPd 3.0.3)\r\n
TARPIT_BANNER=SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1\r\n
```
### 3. Start the Trap
Run the following command to pull the images and start the honeypot:
```Bash
docker compose up -d
```
Access the dashboard at http://localhost:8080 (or your server's IP).

---

## 🧪 Testing the Trap

Once both containers are running, check your Hub dashboard. The Agent should appear in the **Fleet Health** bar as `ONLINE` within 30 seconds.

To simulate an attack, use `netcat` to connect to one of your decoy ports:
```bash
nc <agent-ip> 2222
```
1. If your mode is hold or echo, you will immediately see your fake TARPIT_BANNER.
2. Type a fake exploit payload (e.g., admin).
3. Notice the tarpit delay (or infinite hold) as it traps your connection.
4. Press Ctrl+C to drop the connection.
5. Watch the alert and payload instantly appear on your HoneyWire dashboard and notification pushed to your phone!

---

## 🛡️ Security Notes
* **API Secret:** Ensure your `API_SECRET` is strong and identical on both the Hub and the Agents. The Hub will reject any payloads with mismatched keys.
* **System Arming:** You can toggle the "System Armed" button in the Hub UI to temporarily disable push notifications while doing internal network maintenance or vulnerability scanning.
* **Container Hardening:** HoneyWire utilizes gcr.io/distroless/python3-debian12. Do not attempt to use docker exec -it honeywire-agent sh as there is no shell binary included in the image by design.
* **Distributed Deployment:** It is highly recommended to run the Hub and its Sensors on separate physical or virtual machines. If an attacker compromises a sensor node, they should not have immediate local access to the centralized Hub.
* **Encryption (HTTPS):** Always serve the Hub Web GUI and API over HTTPS. Failure to do so exposes your API_SECRET and DASHBOARD_PASSWORD to anyone sniffing the network.

## 🛠️ Tech Stack
* **Backend:** Python 3.11, FastAPI, SQLite3, Asyncio
* **Frontend:** HTML5, TailwindCSS, Alpine.js, Chart.js
* **Infrastructure:** Docker, Docker Compose, Distroless Linux

---

## 📦 Versioning and API reference

- HoneyWire now uses a single source of truth version file: `VERSION` in the repo root.
- Runtime version is exposed via env override: `HONEYWIRE_VERSION` (Hub + Agent), and defaults to `VERSION`.
- `Hub` endpoint:
  - `GET /api/v1/version` → returns `{ "version": "1.0.0" }`
- API docs file added: [📖 API.md](./API.md). with full backend route reference and sample payloads.

### API endpoints to know
- `GET /api/v1/system/state` / `PATCH /api/v1/system/state`
- `GET /api/v1/sensors`
- `GET /api/v1/events`
- `PATCH /api/v1/events/read`, `PATCH /api/v1/events/{event_id}/read`, `DELETE /api/v1/events`
- `POST /api/v1/heartbeat` (agent heartbeat)
- `POST /api/v1/event` (agent event reports)

---

## 🧪 Operational checklist
- set `API_SECRET` for all components
- set optional `DASHBOARD_PASSWORD`
- optionally set `HONEYWIRE_VERSION` to track deployment version
- build/redeploy containers after any version bump in `VERSION` or env value
