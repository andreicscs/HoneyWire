# THREAT MODEL - HoneyWire

## 1. System Overview

HoneyWire is a decentralized sensor deployment system. For a deep dive into the architecture and data flows, see the [Architecture Overview](Docs/architecture/README.md). From a security perspective, the system is composed of:

- **Hub:** The central orchestrator, policy engine, and trust anchor.
- **Wizard:** An untrusted local CLI tool that discovers host services and requests deployment intents.
- **Manifests:** Untrusted, external JSON definitions that describe sensor behaviors.
- **Sensors:** Untrusted runtime containers executing within a sandboxed Docker environment.

---

## 2. Trust Boundaries & Assumptions

### 2.1 Hub (Trust Anchor & Root Authority)
The Hub is the ONLY enforcement point for deployment safety. It is responsible for rejecting unsafe manifests, enforcing deterministic compilation, and producing the final deployment artifacts.
* **Assumption:** The Hub is secure at the time of operation, runs in a controlled environment, and is protected by OS controls and reverse proxies. If compromised, future generated deployments may be malicious, though existing runtime executions are not directly affected.

### 2.2 Wizard (Untrusted Local Component)
The Wizard operates on the target host to observe state and generate deployment intents (modified manifests bindings).
* **Assumption:** The Wizard is fundamentally untrusted and may be locally compromised. It may leak environment metadata or submit altered deployment requests. Therefore, all Wizard outputs are treated as attacker-controllable input by the Hub.

### 2.3 Manifests (High-Risk Input DSL)
External sensor definitions dictate container images, capabilities, and mounts.
* **Assumption:** Manifests are fully attacker-controllable. Schema validity does *not* imply safety. Every field must be explicitly validated by the Hub.

### 2.4 Execution Boundary (Docker Runtime)
Sensors execute as isolated containers.
* **Assumption:** Docker isolation is not absolute security. Host systems may be compromised, and containers are treated as potentially hostile execution units requiring least-privilege configurations. To minimize post-compromise attacker capabilities, official components and sensors use minimal, shell-less "Distroless" base images.

---

## 3. Threat Analysis

### T1: Manifest & Supply Chain Compromise
**Attacker Capability:** Modifies sensor manifests at the registry, supplies malicious definitions, or alters container images in the registry.

**Impact:** Unsafe privilege requests, malicious configurations, or execution of compromised binaries within the sensor runtime.

* **Mitigations (IMPLEMENTED):**
  * Strict typed schema enforcement (`DisallowUnknownFields` JSON decoding).
  * Hub-side normalization prior to compilation.
  * Interpolation rejection (blocks `${`, `{{`, and `$` patterns).
* **Gaps (NOT IMPLEMENTED):**
  * Image digest pinning (currently relying on image tags).
  * Cryptographic manifest signature verification.
  * Provenance tracking.

### T2: Wizard Misdirection & Deployment Intent Manipulation
**Attacker Capability:** Compromises the local Wizard to alter deployment requests, inject unsafe environment variables, or submit misleading local infrastructure state.

**Impact:** Leakage of local metadata, incorrect sensor recommendations, or subtle privilege escalation via configuration injection.

* **Mitigations (IMPLEMENTED):**
  * Hub treats *all* Wizard-generated deployment intents as fully untrusted.
  * Full re-validation of all manifests and overrides at the Hub compile stage.
  * Port range validation prevents local DoS from malformed `/proc` state.

### T3: Privilege Escalation via Sensor Specification Abuse
**Attacker Capability:** Attempts unsafe container configurations via manifest injection (e.g., requesting dangerous capabilities, exploiting host network namespaces, defining unsafe mounts, or misusing init containers).

**Impact:** Host compromise via container escape paths, unauthorized filesystem access, or lateral network movement.

* **Mitigations (IMPLEMENTED):**
  * Capability allowlist enforcement (all others dropped).
  * Forbidden mount path denylist validation.
  * Read-only root filesystem enforcement for main sensor containers.
  * Security-opt enforcement (`no-new-privileges`).
  * Distroless execution environments (no shell, package manager, or standard utilities) for all official Hub and Sensor images.
  * Dynamic volume path validation during Compose generation.
  * Explicit sandboxing role separation (e.g., heavily hardened `sensor-runtime` vs. scoped `init-provisioner`).

* **Mitigations (PARTIAL):**
  * Controlled volume mounts for init containers (init containers are intentionally more permissive than main sensors to allow for provisioning/FIM tasks).
* **Gaps (NOT IMPLEMENTED):**
  * Strict command checking for `init-provisioner` containers.

### T4: Hub Compromise (Root Trust Failure)
**Attacker Capability:** Modifies the central validation logic, Compose compiler behavior, or signing policies.

**Impact:** Malicious generation of future deployment artifacts and systemic bypass of validation rules.

* **Mitigations (IMPLEMENTED):**
  * Hub runs in a least-privilege, distroless Docker container (sandboxed execution).
* **Gaps (NOT IMPLEMENTED):**
  * Signed deployment artifact enforcement (planned).

### T5: Node Authentication Compromise
**Attacker Capability:** Steals node identity API keys to deploy unapproved sensors on compromised nodes or flood the Hub with fake telemetry.

**Impact:** Audit log confusion, false positive alerts, or Hub Denial of Service (DoS).

* **Mitigations (IMPLEMENTED):**
  * Node API Key Rate Limiting: Token bucket rate limit (100 req/min) enforced on all API endpoints to prevent high-volume DoS. UI login rate limiting is also active.

---

## 4. Risk Register (Accepted Risks)

The following risks represent known architectural trade-offs or constraints that are explicitly formally accepted:

### Risk-01: Post-Delivery Compose Tampering
A compromised local Wizard could bypass Hub validation entirely by ignoring the `honeywire-compose.yml` returned by the Hub, instead executing a locally crafted, malicious compose file.
* **Why it is accepted:** Hub validation guarantees the safety of what it *generates*, not what the host *chooses to run*. Exploiting this requires write access to the filesystem and access to the Docker socket. An attacker with these privileges already possesses equivalent root-level control of the host. Thus, HoneyWire-specific controls are no longer the relevant security boundary; it is a general host-security failure.

### Risk-02: CLI Command History Exposure
During the initial host provisioning link, the Wizard requires the API key as a CLI argument (e.g., `wizard --link <url> --api-key <key>`). This exposes the credential briefly in `ps` output and shell history.
* **Why it is accepted:** This is a one-time bootstrap operation, standard for unattended deployments, and assumed to occur within a controlled environment prior to active hostile presence. After initialization, credentials are read safely from configuration files.

---

## 5. Core Security Principles

HoneyWire security is not based on trust. It is predicated on strict verification and boundary enforcement:

1. **Hub is the ONLY Authority:** No security or policy logic resides in the Wizard.
2. **All Inputs are Untrusted:** External manifests, local Wizard requests and deployed sensors telemetry are treated as hostile input.
3. **Compile-Time Security:** Security policies (capabilities, mounts, read-only constraints) are structurally enforced during Compose generation, before runtime execution begins.
4. **Deterministic Generation:** The compiler must produce predictable, auditable artifacts.
5. **Least Privilege:** Containers are assumed to be potentially hostile units, defaulting to read-only environments with explicitly allowed capabilities, and executed via Distroless base images to eliminate secondary payload tooling.