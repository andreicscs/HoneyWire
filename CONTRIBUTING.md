[![License](https://img.shields.io/badge/license-AGPLv3-blue.svg)](LICENSE)

# Contributing to HoneyWire

Welcome to HoneyWire! We are building a centralized, high-fidelity security and deception ecosystem for homelabs and SMBs. 

Whether you want to build a new decoy sensor, improve the Vue.js frontend, or optimize the Go backend, your contributions are highly welcome.

---

## Development Documentation

To keep this guide concise, we have moved the in-depth technical guides to the **[Development Docs](./Docs/development/README.md)** directory. 

Before contributing, please review the relevant documentation:
- **[Local Setup & Environment](./Docs/development/setup.md)**: How to spin up the Hub, Frontend, and Mock Hub.
- **[Contribution Rules](./Docs/development/contributionRules.md)**: Essential coding standards, branching strategies, and PR guidelines.
- **[Building Sensors](./Docs/development/sensors.md)**: A complete guide to creating, testing, and submitting new deception sensors.
- **[Wizard Development](./Docs/development/wizard.md)**: Guidelines for contributing to the Wizard CLI engine.
- **[Maintainer Workflow](./Docs/development/maintainer-workflow.md)**: Internal workflows for tagging, releasing, and updating the manifest registry.

---

## Project Structure

Here is a high-level overview of the HoneyWire repository:

```text
honeywire/
├── Docs/            # Comprehensive documentation (Architecture, Development, Security)
├── Hub/             # Central brain (Go backend + Vue 3 frontend)
│   ├── cmd/hub/     # Main Go entrypoint
│   ├── internal/    # Go packages (api, store, registry)
│   └── ui/          # Vue 3 Frontend (Vite + TailwindCSS)
├── SDKs/            # Language SDKs (Go, Python) for building sensors
├── Sensors/         # Decoy nodes
│   ├── official/    # First-party maintained sensors
│   ├── community/   # Community-submitted sensors
│   └── templates/   # Boilerplate templates (go-sensor, python-sensor)
├── wizard/          # The Wizard CLI (Intelligent Deception Deployment)
└── scripts/         # Utility scripts (e.g., mock_hub.py for testing)
```

---

## Contributing a New Sensor

To keep the ecosystem stable, all community-submitted sensors must adhere to our DevSecOps rules. We treat sensors as **isolated, unprivileged microservices**.

1. **Use the Official Templates:** Start by copying either the `go-sensor` or `python-sensor` folder from [`Sensors/templates/`](./Sensors/templates/) into [`Sensors/community/`](./Sensors/community/). 
2. **Follow the Data Contract:** Your sensor must POST a payload matching the Universal Event Standard. Using the provided SDKs handles this for you.
3. **Strict Sandboxing:** We strongly enforce the use of minimal, hardened base images (like Distroless) running as non-root users (`UID 65532`) with all Linux kernel capabilities dropped (`cap_drop: ALL`).
4. **Implement Test Mode:** Our CI pipeline tests your sensor by passing `HW_TEST_MODE=true`. Ensure your code handles this by securely sending a heartbeat and mock event before exiting gracefully.

For step-by-step instructions, please read the **[Sensor Development Guide](./Docs/development/sensors.md)**.

---

## Review Process

Once you open a Pull Request:
1. **Automated Security Scanning:** GitHub Actions/Gitea will run **Trivy** to scan your Docker image for vulnerabilities, and **CodeQL** to perform static code analysis.
2. **Functional Testing:** Our CI will automatically build your Docker container and test it against a Mock Hub.
3. **Manual Review:** A core maintainer will manually review the code, specifically checking for malicious intent and proper capability stripping.

Join us in building a smarter, faster, and more resilient distributed defense.