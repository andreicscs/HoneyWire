# HoneyWire Documentation

Welcome to the official HoneyWire documentation! This directory contains everything you need to understand, operate, and develop for the HoneyWire ecosystem.

## 📚 Directory Structure

To help you find what you're looking for, our documentation is split into several focused areas:

### [1. Operations Guide](/Docs/operations.md)
**Who it's for:** System Administrators and Operators.
**What you'll learn:** The day-to-day lifecycle management of your HoneyWire ecosystem. This guide covers how to provision new nodes, how the declarative state deployment works, how to sync nodes using the Setup Wizard (`apply`), and how to handle updates, rollbacks, and node teardowns safely.

### [2. Architecture (`/architecture`)](/Docs/architecture/README.md)
**Who it's for:** System Architects, Developers, and Security Researchers.
**What you'll learn:** The underlying technical design of HoneyWire. This section dives deep into how the Sentinel Hub, Setup Wizard, and Sensors communicate. It includes detailed data contracts and component-specific breakdowns to help you understand the magic behind the high-signal tripwires.

### [3. Development (`/development`)](/Docs/development/README.md)
**Who it's for:** Contributors, Maintainers, and Custom Sensor Authors.
**What you'll learn:** How to get your hands dirty with the HoneyWire codebase. This directory contains contribution rules, local setup guides, maintainer workflows, and comprehensive tutorials on how to build and publish your own custom HoneyWire sensors.


---

> [!WARNING]
> **HTTPS & Reverse Proxy Required**
> The HoneyWire Hub **does not** natively terminate TLS/HTTPS. It is critical that you deploy the Hub behind a secure reverse proxy (such as Nginx, Caddy, or Traefik) configured with SSL/TLS. Exposing the Hub over raw HTTP will expose your API keys, deployment manifests, and event telemetry to network interception. For more details on the system's trust boundaries and security posture, please review the [Threat Model](/THREATMODEL.md) and [Security Policy](/SECURITY.md).

---

**Tip:** If you're just getting started and want to deploy your first sensor, start by reading the **[Operations Guide](/Docs/operations.md)**!
