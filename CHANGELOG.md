# [v2.0.0] - "The Security and UX Update" (2026-06-22)

## This is a massive, breaking architectural update that transitions HoneyWire from a passive event listener into a comprehensive, distributed fleet management platform.

### ⚠️ Breaking Changes
* **Global API Path Bump:** All internal and external routing endpoints have been migrated to `/api/v2/`.
* **Deprecation of Standalone Sensors:** Sensors are no longer deployed as floating, unmanaged containers. All sensors must now be attached to a Node and managed via the new Hub Fleet Architecture. Existing `v1.1.1` deployments will need to be re-provisioned using the new Node structure.
* **Registry Migration:** Official sensor images are now indexed in the `registry-pages` branch to allow for better sensor lifecycle management and strict manifest validation.

---

### Major Features
* **Introduced The Wizard CLI:** A brand new, cross-platform CLI tool built to automate the operator lifecycle. The Wizard handles environment discovery, automated node linking, sensor deployment, and synthetic firedrill testing. *The Wizard is not a background daemon agent; it is a point-in-time CLI tool designed to automate operator tasks and exit.*
* **Nodes & Fleet Management:** The Hub has been completely overhauled to support a Node-based architecture. You can now track, manage, and monitor the health of multiple host machines (Nodes) from a single centralized Fleet dashboard.
* **Sensor Catalog & Deployment Engine:** You no longer need to manually write `docker-compose` files for sensors. The Hub now features a built-in Fleet Management view and Sensor Catalog pulling from a live Registry, allowing users to select and configure sensors directly from the Hub. Operators can generate copy-pasteable deployment commands leveraging the Wizard.
* **State Synchronization:** Added a real-time sync engine between the Hub and deployed Nodes. The Hub now tracks `Pending` vs `Deployed` states and automatically resolves them when a Node checks in.
* **60-Second Cyber Canary Deployments:** Thanks to the improved UX of the new Wizard and Fleet architecture, you can now seamlessly turn any standard Linux box into a fully provisioned Cyber Canary in under 60 seconds.
* **Event Source Whitelisting:** Added native support for event source whitelisting via the Hub settings to effortlessly filter out noise from known internal security scanners.

---

### Security Enhancements
* **Threat Model Documentation:** We have formally published our [Threat Model](./THREATMODEL.md) detailing trust boundaries, attack surfaces, and mitigations.
* **Supply Chain & Manifest Hardening:** To mitigate supply chain attacks, the Hub now enforces strict typed schema decoding, blocks variable interpolation, and enforces cryptographic image digest pinning for immutable sensor pulls.
* **Rate Limiting & DoS Protection:** Implemented a strict token-bucket rate limiter (100 req/min) across all Node API endpoints to prevent telemetry flooding, alongside hardened UI login limits.
* **Runtime Sandboxing & Distroless Execution:** Enforced distroless base images (no shells or package managers) across all official Hub and Sensor containers. Added explicit validation for dangerous mounts and strict capability allowlists to prevent container privilege escalation.

---

*Changelog: v1.1.1 → v2.0.0 | 2026-06-22*
