# Community Sensor Lab

Welcome to the HoneyWire Community Sensor Lab. This directory provides a space to deploy, share, and experiment with custom deception sensors.

HoneyWire's architecture allows you to build a specialized trap for any protocol and visualize its telemetry in a unified dashboard using the Universal Event Standard.

## Data-Driven Architecture

To add a sensor to the ecosystem, you only need to define it using a `manifest.json` file. This JSON schema is responsible for:
1. Generating interactive UI forms and configuration cards in the HoneyWire Hub.
2. Providing heuristic metadata to the Wizard CLI for host recommendations.
3. Defining the deployment parameters (containers, mounts, variables) required for execution.

Please refer to the official documentation for the complete manifest schema:
**[View the Sensor Manifest Data Contract](../../Docs/architecture/dataContracts.md)**

## Engineering Resources

To ensure structural consistency and security, start with the provided scaffolding:

* **[Sensor Template](../templates/README.md):** The recommended starting point, natively integrated with the HoneyWire Go SDK and pre-configured for Distroless execution.
* **[Contribution Guide](../../CONTRIBUTING.md):** Mandatory reading covering required capability stripping, container best practices, and the JSON Contract.

**Security Standard:** We encourage community submissions to follow the official architecture: statically-linked binaries running as unprivileged users inside `:nonroot` Distroless containers, with all Linux kernel capabilities that are needed explicitly added, all capabilities are dropped by default (`cap_drop: ["ALL"]`).

## Contribution Policy

Community submissions undergo automated validation before review:

1. **Static Analysis:** Scanning for security vulnerabilities and memory leaks (e.g., CodeQL).
2. **Container Security:** Image vulnerability scanning (e.g., Trivy).
3. **Functional Testing:** Automated verification against a mock network contract in `HW_TEST_MODE`.

## Submitting a Sensor

1. Fork the repository.
2. Create your sensor directory under `Sensors/community/your-sensor-name`.
3. Implement your sensor logic using the [HoneyWire Go SDK](../../SDKs/go-honeywire) or equivalent language SDK.
4. Define your `manifest.json` according to the official schema.
5. Open a Pull Request for review.