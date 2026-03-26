# 🕸️ HoneyWire

**HoneyWire** is an ultra-lightweight, distributed micro-honeypot and command center. It is designed to deploy silent, asynchronous "tarpit" sensors across multiple servers that detect unauthorized scans, trap automated botnets, and report telemetry back to a centralized dashboard in real-time.

Developed in collaboration with Gemini (Google AI). 
Architected by Termine Andrea, implementation and boilerplate assisted by LLM.

There already exist lightweight honeypots but none had a simple clean dashboard with webhook notification integrations, and the ones that do are incredibly resource intensive.This project aims at filling that gap in the cybersecurity software and tools landscape for hobbyists.

---

## Screenshots

### Main Dashboard
![Dashboard](screenshots/dashboard.png)

### Payload Inspector
![Payload Inspector](screenshots/payload-inspector.png)

---

## ✨ Features

* **The Echo Tarpit:** Sensors use Python's `asyncio` to hold thousands of malicious connections open simultaneously with almost zero RAM usage. It echoes payloads back to attackers to break automated exploitation scripts.
* **Distributed Architecture:** A decoupled Hub-and-Spoke model. Run the Hub safely on your internal network, and deploy Agents to public-facing VPS instances or IoT devices.
* **Zero-Touch Dashboard:** Built with FastAPI, Alpine.js, and Chart.js. Real-time fleet health monitoring, top targeted ports, and interactive JSON payload inspection.
* **Push Notifications:** Native integration with [ntfy.sh](https://ntfy.sh/) for instant mobile alerts when a sensor is tripped.
* **Featherweight:** Both the Hub and Agent are containerized using `python:3.11-alpine`, resulting in minimal resource footprints.

---

## 🏗️ Architecture

HoneyWire is split into two independent microservices:

1. **The Hub (`/Hub`)**: The central brain. It runs a FastAPI backend, an SQLite database, and the web dashboard.
2. **The Agent (`/Agent`)**: The decoy sensor. It listens on vulnerable ports, traps attackers, and securely POSTs the intrusion data back to the Hub.

---

## 🚀 Quick Start Guide

### Prerequisites
* Docker and Docker Compose installed on your host machine(s).

### 1. Deploy the Hub (Command Center)
Spin up the Hub first so it is ready to receive incoming sensor data.

```bash
cd Hub
```

Create your `.env` file:
```env
# The port the dashboard will be accessible on
HUB_PORT=8080

# The master password for your fleet
API_SECRET=super_secret_key_123

# Optional: Get push notifications to your phone
NTFY_URL=[https://ntfy.sh/your_private_topic](https://ntfy.sh/your_private_topic)
```

Build and start the Hub:
```bash
sudo docker compose up -d --build
```
*Access the dashboard at `http://localhost:8080` (or your server's IP).*

### 2. Deploy an Agent (Sensor)
You can deploy this on the same machine as the Hub, or move the `Agent` folder to a completely different server across the world.

```bash
cd Agent
```

Create your `.env` file:
```env
# Point this to your Hub's IP address and Port
HUB_URL=[http://127.0.0.1:8080](http://127.0.0.1:8080)

# Must match the Hub's secret
API_SECRET=super_secret_key_123

# Identify this specific sensor and its IP
SENSOR_ID=dmz-node-01
SENSOR_IP=192.168.1.50

# A comma-separated list of fake ports to open
DECOY_PORTS=21,22,2222,3306,8080
```

Build and start the Agent:
```bash
sudo docker compose up -d --build
```
*(Note: The Agent runs in `network_mode: "host"` to accurately capture port scans against the physical machine's interfaces.)*

---

## 🧪 Testing the Trap

Once both containers are running, check your Hub dashboard. The Agent should appear in the **Fleet Health** bar as `ONLINE` within 30 seconds.

To simulate an attack, use `netcat` to connect to one of your decoy ports:
```bash
nc <agent-ip> 2222
```
1. Type a fake exploit payload (e.g., `admin`).
2. Notice the tarpit delay as it echoes it back.
3. Press `Ctrl+C` to drop the connection.
4. Watch the alert instantly appear on your HoneyWire dashboard!

---

## 🛡️ Security Notes
* **API Secret:** Ensure your `API_SECRET` is strong and identical on both the Hub and the Agents. The Hub will reject any payloads with mismatched keys.
* **System Arming:** You can toggle the "System Armed" button in the Hub UI to temporarily disable push notifications while doing internal network maintenance or vulnerability scanning.

## 🛠️ Tech Stack
* **Backend:** Python 3.11, FastAPI, SQLite3, Asyncio
* **Frontend:** HTML5, TailwindCSS, Alpine.js, Chart.js
* **Infrastructure:** Docker, Docker Compose, Alpine Linux