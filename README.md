# 🕸️ HoneyWire

**HoneyWire** is an ultra-lightweight, distributed micro-honeypot and command center. It is designed to deploy silent, asynchronous "tarpit" sensors across multiple servers that detect unauthorized scans, trap automated botnets, and report telemetry back to a centralized dashboard in real-time.

Developed in collaboration with Gemini (Google AI). 
Architected by Termine Andrea, implementation and boilerplate assisted by LLM.

There already exist lightweight honeypots but none had a simple clean dashboard with webhook notification integrations, and the ones that do are incredibly resource intensive. This project aims at filling that gap in the cybersecurity software and tools landscape for hobbyists.

---

## Screenshots

### Main Dashboard
![Dashboard](screenshots/dashboard.png)

### Payload Inspector
![Payload Inspector](screenshots/payload-inspector.png)

---

## ✨ Features

* **The Chameleon Tarpit:** Sensors use Python's `asyncio` to hold thousands of malicious connections open simultaneously. You can dynamically choose how the trap reacts:
  * `hold`: The Black Hole. Keeps the connection open indefinitely, saying nothing, permanently burning the attacker's scanning threads.
  * `echo`: The Parrot. Echoes payloads back to attackers to break automated exploitation scripts.
  * `close`: The Slammed Door. Drops the connection immediately after logging the IP.
* **Service Spoofing:** Fully customizable service banners. Fool scanners like Shodan into thinking your tarpit is a legitimate OpenSSH or vsFTPd server.
* **Distributed Architecture:** A decoupled Hub-and-Spoke model. Run the Hub safely on your internal network, and deploy Agents to public-facing VPS instances or IoT devices.
* **Push Notifications:** Native integration with [ntfy.sh](https://ntfy.sh/) and [Gotify](https://gotify.net/) for instant mobile alerts when a sensor is tripped.
* **Enterprise-Grade Containers:** Built using multi-stage Google `distroless/python3` images. The Hub runs completely `rootless`, and the Agent operates in a "void" with zero shell access (`/bin/sh`), virtually eliminating the container attack surface.

---

## 🏗️ Architecture

HoneyWire is split into two independent microservices:

1. **The Hub (`/Hub`)**: The central brain. It runs a FastAPI backend, an SQLite database, and the web dashboard. It runs as a `nonroot` user inside a Distroless container, safely mounting data to a dedicated volume.
2. **The Agent (`/Agent`)**: The decoy sensor. It listens on vulnerable ports, traps attackers, and securely POSTs the intrusion data back to the Hub.

---

## 🚀 Quick Start Guide

### Prerequisites
* Docker and Docker Compose installed on your host machine(s).

### 1. Deploy the Hub (Command Center)
Spin up the Hub first so it is ready to receive incoming sensor data.

```bash
cd Hub
