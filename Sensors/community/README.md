# Community Sensor Lab

Welcome to the **HoneyWire Community Sensor Lab**. While our official sensors provide the core foundation, this directory is where the decentralized deception ecosystem truly thrives.

The power of HoneyWire lies in the **Universal Event Standard**—the architectural flexibility for anyone to build a specialized trap for any niche protocol and see that telemetry normalized on a central dashboard. Whether you have engineered a DNS sinkhole, a malformed JWT detector, or a custom file-integrity monitor, this is the environment to deploy and share it.

---

## 🛠️ Engineering Resources
Don't start from a blank repository. We have provided the scaffolding to move your sensor from a conceptual trap to a hardened, deployed container in minutes:

* **[🐹 Go Sensor Template](./../templates/go-sensor-template/README.md):** The high-velocity starting point, natively integrated with the HoneyWire Go SDK and pre-configured for Distroless isolation.
* **[📖 Contribution Guide](./../../CONTRIBUTING.md):** **Mandatory reading.** Contains the "Golden Rules" for capability stripping, environment parity, and the JSON Contract.
* **[🔬 Reference Implementation](./../official/TcpTarpit/README.md):** A deep dive into the architecture of our production-grade TCP Tarpit, demonstrating Go routine concurrency and semaphore limits.

> **🛡️ Security Standard:** The HoneyWire ecosystem has moved past heavy interpreters. We strongly encourage community submissions to follow our official architecture: **pure Go, statically-linked binaries running as unprivileged users inside `:nonroot` Distroless containers, with all Linux kernel capabilities dropped.**

---

## 🛡️ The Security Policy
To protect our users, every sensor submitted here undergoes a rigorous automated gauntlet before a human maintainer even looks at the code:

1.  **Static Analysis:** CodeQL scans your Go logic for security vulnerabilities and memory leaks.
2.  **Container Security:** Trivy scans your `Dockerfile` and base images for known CVEs.
3.  **Functional Testing:** Our CI/CD spins up your sensor with `HW_TEST_MODE=true` and verifies its "Heartbeat" and "Event" logic against a mock network contract.

> **Note:** Community sensors expand the ecosystem's detection capabilities significantly, but always remember to review the code and container privileges before deploying them in your own production environment!

---

## 🤝 How to Contribute?
1.  **Fork** the repository.
2.  **Create** your sensor directory: `Sensors/community/your-sensor-name`.
3.  **Implement** your logic using the [HoneyWire Go SDK](../../SDKs/go-honeywire) based on the provided template.
4.  **Harden** your `docker-compose.yml` to adhere to the principle of least privilege.
5.  **Open a Pull Request**.

**Join us in building a smarter, faster, and more resilient distributed defense.** 🐝

---
[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](../../LICENSE)
[![Status](https://img.shields.io/badge/status-WIP-yellow.svg)]()