# 🐝 Community Sensor Lab

Welcome to the **HoneyWire community sensors**. While our official sensors provide the foundation, this directory is where the ecosystem truly grows.

The strength of HoneyWire lies in its **Universal Event Standard**—the ability for anyone to build a specialized trap for a niche protocol and see that data come to life on the central dashboard. Whether you've built a DNS sinkhole, a malformed JWT detector, or a custom file-integrity monitor, this is the place to share it.

---

## 🛠️ Get Building
Don't start from a blank page! We’ve built the tools to get your sensor from an idea to a deployed container in minutes:

* **[🐍 Python Sensor Template](./../templates/python-sensor/README.md):** The quickest way to start. It comes pre-configured with the HoneyWire SDK.
* **[📖 Contribution Guide](./../../CONTRIBUTING.md):** Read this first. It contains the "Golden Rules" for Dockerization, Environment Variables, and the JSON Contract.
* **[🔬 Official Example](./../official/TcpTripWire/README.md):** See how we built the TCP Tarpit.

> **Note:** We strongly recommend using **Distroless** base images (as seen in the template) to ensure your sensor has the smallest possible attack surface.

---

## 🛡️ The Security Policy
To protect our users, every sensor submitted here undergoes a rigorous automated gauntlet before a human maintainer even looks at the code:

1.  **Static Analysis:** CodeQL scans your logic for security vulnerabilities.
2.  **Container Security:** Trivy scans your `Dockerfile` and base images for known CVEs.
3.  **Functional Testing:** Our CI/CD spins up your sensor and verifies its "Heartbeat" and "Event" logic against a **Mock Hub**.

> **Note:** Community sensors are awesome, but always remember to review the code before deploying them in your own production environment!

---

## 🤝 How to Contribute?
1.  **Fork** the repository.
2.  **Create** your sensor directory: `Sensors/community/your-sensor-name`.
3.  **Implement** your logic using the [HoneyWire Python SDK](../../SDKs/python-honeywire).
4.  **Open a Pull Request**.

**Let's build a smarter, faster, and more distributed defense. One sting at a time.** 🐝

---
[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](../../LICENSE)
[![Status](https://img.shields.io/badge/status-WIP-yellow.svg)]()