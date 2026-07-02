<p align="center">
  <img src="Hub/ui/public/favicon.svg" alt="HoneyWire Logo" width="150" />
</p>
<h1 align="center">HoneyWire</h1>

<p align="center">
  <a href="https://github.com/andreicscs/HoneyWire/releases">
    <img src="https://img.shields.io/badge/release-v2.0.3-blue.svg?style=flat-square" alt="Latest Release" />
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/license-AGPLv3-blue.svg?style=flat-square" alt="License: GPLv3" />
  </a>
  <a href="https://news.risky.biz/risky-bulletin-nist-gives-up-enriching-most-cves/#:~:text=New%20tool%E2%80%94HoneyWire%3A%20Andrea%20Termine">
    <img src="https://img.shields.io/badge/Risky%20Bulletin-New%20Tools-2E8B57?logo=RiskyBusiness&style=flat-square" alt="Risky Bulletin" />
  </a>
  <a href="Hub/docker-compose.yml">
    <img src="https://img.shields.io/badge/Docker-Native-2496ED.svg?style=flat-square&logo=docker&logoColor=white" alt="Docker Native" />
  </a>
</p>

## 📋 Table of Contents
- [Overview](#overview)
- [Showcase](#showcase)
- [The Universal Event Standard](#-the-universal-event-standard-bring-your-own-sensor)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start Guide](#-quick-start-guide)
- [Security Notes](#security-notes)
- [Tech Stack](#tech-stack)
- [Versioning and API Reference](#versioning-and-api-reference)

## Overview
**HoneyWire** is a lightweight, Distributed High-Signal Security Early-Warning System Builder, designed for internal networks. It leverages its architecture and UX to make it incredibly easy to build a new Cyber Canary server or deploy HoneyWires on existing ones. Using deception technology, it replaces the "magnifying glass" approach of traditional SIEMs which often drown analysts in false positives by surveilling legitimate traffic with a High-Fidelity Tripwire model. 

Place a sensor exactly where you want it. If it trips, you have an intruder.
  - **Production Tripwires**: Sound the alarm when active services are being poked in ways they shouldn't be. By placing a sensor on a sensitive file that should never be read or a service port that should never be accessed, you identify intruders by their deviation from the "authorized path."
  - **Synthetic Deception**: Deploy lures like the [ICMP Canary](./Sensors/official/IcmpCanary/) or [Network Scan Detector](./Sensors/official/NetworkScanDetector/) to act as decoys. Since these sensors provide no legitimate business value, 100% of their traffic is actionable intelligence.

Set up multiple and you start to have a pretty clear idea of the lateral movement of an intruder. No tuning, no noise, just instant forensics.
If you have legitimate automated security scanners tripping HoneyWires just whitelist them from the Hub's settings.

## Showcase

<div align="center">
  <video src="https://github.com/user-attachments/assets/b82d3c18-9fc9-4393-84c8-4cd7046d9517" autoplay loop muted playsinline width="100%"></video>
</div>



## 🔌 The Universal Event Standard (Bring Your Own Sensor)

[**Community Sensors**](./Sensors/community/)
> Note: If you build your sensor using the official HoneyWire SDKs, this JSON formatting and delivery is handled for you automatically.

The true power of HoneyWire is that the Hub and Wizard are **completely sensor-agnostic**. You are not limited to the included official sensors. 

By adhering to the **HoneyWire Event Standard V2.0**, you can write a script in *any* language (Bash, Go, Rust, Python) to monitor *anything*, and the Sentinel UI will dynamically parse, syntax-highlight, and render your forensic data. 

Whether it is a **Deep Packet Inspection (DPI)** engine, a **DNS sinkhole**, a **Canary Token** embedded in a PDF, an **Email Honeypot**, or a simple **TCP Port Tripwire**, just POST the **Universal Event Standard** JSON payload to the Hub. 

> **[View the full Event Data Contract here](./Docs/architecture/dataContracts.md#1-the-universal-event-standard)** 

## Features

- **The Sentinel Hub UI:** A fully responsive, Vue 3-powered dashboard featuring dynamic forensic payload inspection, Nodes and Sensors deployment and management, including sensor updates, directly from the UI.
  - **Universal Push Notifications:** Native, zero-dependency integration for routing critical alerts to **Discord, Slack, Ntfy, and Gotify**.
  - **Enterprise SIEM Integration:** Native RFC5424 Syslog forwarding (TCP/UDP) for seamlessly pushing structured telemetry to Splunk, Elastic, Wazuh, or Vector.
- **The Setup Wizard:** A deployment and testing automation tool, developed explicitly to not be a 24/7 running agent. It is simply a TUI CLI tool that automates operator tasks like applying and reconciling the Hub's desired state for a given Node, automatically handling configuration, deployment, and rollbacks on failed deployments.
- **Suite of Official HoneyWires:** Includes native [TCP Tarpit](./Sensors/official/TcpTarpit/), [Web Router Decoy](./Sensors/official/WebRouterDecoy/), [File Canary (FIM)](./Sensors/official/FileCanary/), [ICMP Canary](./Sensors/official/IcmpCanary/), and [Network Scan Detector](./Sensors/official/NetworkScanDetector/).

## Architecture

HoneyWire is split into four independent microservices:

1. `/Hub`: The central brain. A pure Go binary running an embedded SQLite database and the Vue.js dashboard. It runs as a non-root user inside a Distroless container, safely mounting data to a dedicated volume.
2. `/Sensors`: The decoy nodes. Statically-linked Go binaries that listen on vulnerable ports, trap attackers, and securely POST intrusion data back to the Hub.
3. `/SDKs`: Official libraries (like `sdk-go`) that handle secure Hub communication so community developers can easily build new sensors.
4. `/wizard`: Setup wizard cli tool to automate operator tasks such as discovery, deployment and testing of HoneyWires.

> **[Check out the full architecture docs](./Docs/architecture/README.md)**
> **[Read the User Operations Guide](./Docs/operations.md)**

## 🚀 Quick Start Guide

Deploying the HoneyWire Hub takes less than 60 seconds using our pre-built GitHub Container images.

### 1. Deploy the Hub
Create a new directory on your server, create a `docker-compose.yml` file, and paste the following:

```yaml
services:
  # 1. THE PERMISSION FIXER: Runs once to ensure the Hub can write to the data volume
  permission-fixer:
    image: alpine:latest
    container_name: honeywire-permission-fixer
    command: sh -c "chown -R 65532:65532 /data"
    volumes:
      - ./honeywire_data:/data

  # 2. THE HUB: The central Go-based dashboard and API
  hub:
    image: ghcr.io/andreicscs/honeywire-hub:latest
    container_name: honeywire-hub
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./honeywire_data:/data
    depends_on:
      permission-fixer:
        condition: service_completed_successfully
        
    # Strict Security Sandbox
    user: "65532:65532"
    read_only: true
    cap_drop: ["ALL"]
    security_opt: ["no-new-privileges:true"]
    
    environment:
      - HW_ENV=development # Required if not using HTTPS, or the cookie will have the secure flag set, in production it is highly recommended to remove this and run this behind a reverse proxy using https
      - HW_PORT=8080
      - HW_DB_PATH=/data/honeywire.db
      # Optional: Hardcode the dashboard password (disables the UI password reset feature)
      # - HW_DASHBOARD_PASSWORD=admin
```

Start the Hub:
```bash
docker compose up -d
```

### 2. Initialize the System
Navigate to `http://<your-server-ip>:HW_PORT` in your browser. You will be greeted by the **Initialize Sentinel** screen.
1. Create your Master Password.
2. Verify your Hub Endpoint URL (the IP/URL where sensors will reach the Hub).
3. Click "Initialize Hub".

### 3. Deploy Sensors
1. Create a new Node and paste the provided command to install and link the Setup Wizard to the hub
2. Install sensors directly from the Hub and then run `honeywire apply` on the node to reconcile desired state or run `honeywire discover` to let the Setup wizard automatically scan and suggest HoneyWires based on environment.

### 4. Testing the HoneyWires

Once your containers are up, the Tarpit sensor should appear as `ONLINE` within 30 seconds.
Run `honeywire firedrill` to make the HoneyWires send a mock event to the hub to test connectivity.

> **Note:** For a deeper dive into managing nodes, updating sensors, and handling rollbacks, see the **[User Operations Guide](./Docs/operations.md)**.

---

## Security Notes
* **Threat Model:** For a deep dive into trust boundaries, architecture risks, and sandboxing rules, see the **[THREATMODEL.md](./THREATMODEL.md)**.
* **Node Keys:** Ensure your sensors use their unique `Node Key` to communicate with the Hub. The Hub will reject any payloads with mismatched or invalid keys.
* **System Arming:** You can toggle the "System Armed" button in the Hub UI to temporarily disable push notifications while doing internal network maintenance or vulnerability scanning.
* **Container Hardening:** HoneyWire utilizes `gcr.io/distroless/static-debian12:nonroot`. We follow the principle of least privilege to make sure that if a container is compromised, the blast is contained.
* **Distributed Deployment:** It is highly recommended to run the Hub and its Sensors on separate physical or virtual machines. If an attacker compromises a sensor node, they should not have immediate local access to the centralized Hub.
* ⚠️ **Encryption (HTTPS):** Always serve the Hub Web UI and API over HTTPS using a reverse proxy (like Nginx, Caddy, or Traefik) in production. Failure to do so exposes your Dashboard password and Node Keys to network sniffing.


## Tech Stack
* **Backend:** Go 1.25, `net/http` (Standard Library), SQLite (ModernC Pure Go Driver)
* **Frontend:** Vue 3 (Composition API), TailwindCSS, Chart.js
* **Infrastructure:** Docker, Docker Compose v5.0.0+


## Versioning and API Reference

- HoneyWire versions are managed via Git Tags.
- `Hub` endpoint:
  - `GET /api/v2/version` → returns `{ "version": "v2.0.3" }`
- API docs file: [API.md](./Docs/architecture/hub/backend/API.md) with full backend route reference and sample payloads.
