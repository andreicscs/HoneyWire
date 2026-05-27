TODO

# THREAT MODEL — HoneyWire

## 1. System Overview

HoneyWire is a decentralized sensor deployment system composed of:

- **Wizard (Untrusted Local Decision Tool)**
  Performs local environment discovery and submits deployment recommendations.

- **Hub (Trust Anchor: Policy + Compiler + Signer)**
  Central authority responsible for:
  - validating manifests (strict schema + security rules)
  - enforcing capability / mount / runtime policies
  - compiling Docker Compose artifacts
  - signing deployment outputs

- **Sensor Manifests (Untrusted Declarative DSL)**
  External JSON definitions describing:
  - container configuration
  - init containers
  - volumes / mounts
  - capabilities
  - heuristics and triggers

- **Deployed Sensors (Untrusted Runtime Containers)**
  Containers running in Docker with enforced restrictions and sandboxing assumptions.

---

## 2. Trust Boundaries

### 2.1 Hub Trust Boundary (ROOT SECURITY AUTHORITY)

The Hub is the only enforcement point for deployment safety.

It is responsible for:
- rejecting unsafe or non-compliant manifests
- enforcing deterministic compilation rules
- applying security hardening defaults
- producing signed deployment artifacts

**Important:**
- Hub is a trust anchor, not an invulnerable system
- compromise affects future deployments, not existing runtime execution directly

---

### 2.2 Wizard Trust Boundary (UNTRUSTED LOCAL COMPONENT)

The Wizard is NOT part of the security enforcement chain.

It:
- observes local system state (processes, ports, services)
- generates sensor recommendations
- submits deployment requests to the Hub

The Wizard:
- may be compromised locally
- may leak environment metadata
- may be manipulated to produce misleading recommendations

However:
> Wizard output never affects security policy directly — only Hub validation does.

---

### 2.3 Manifest Trust Boundary (HIGH-RISK INPUT DSL)

Sensor manifests are fully untrusted external input.

They define:
- container images
- init containers
- environment variables
- volume mounts
- capability requests

A malicious manifest may attempt to:
- pull malicious images
- request privilege escalation
- inject unsafe mount paths
- exploit interpolation / template injection attempts
- misuse init containers for host interaction

All such behavior must be blocked by Hub validation.

---

### 2.4 Compilation Boundary (HUB BUILD SYSTEM)

The Compose builder is part of the trusted Hub pipeline.

It MUST:
- only operate on validated typed models
- NOT perform unsafe string interpolation
- enforce deterministic output generation

**Important constraint (current state):**
- init containers are still partially mutable execution surfaces
- they are NOT fully sandboxed like main sensor containers

---

### 2.5 Execution Boundary (DOCKER RUNTIME)

Sensors run in Docker with:
- read-only root filesystem (main containers)
- dropped capabilities by default
- optional capability allowlist

However:
- Docker is NOT treated as a strong security boundary
- container escape is considered possible in adversarial environments

---

## 3. Assets to Protect

- Hub signing keys
- Deployment artifacts (Compose YAML)
- Node authentication keys
- Manifest registry integrity
- Host systems running sensors
- Local environment metadata (Wizard-collected)
- Volume-mounted host paths

---

## 4. Threats

### Threat 1: Malicious Manifest Injection (Manifest supply chain compromise)

**Attacker capability:**
- modifies or supplies malicious sensor manifests

**Impact:**
- unsafe privilege requests (cap_add abuse, mounts, init abuse)
- invalid or dangerous deployment configurations
- potential host-level exposure via container misconfiguration

**Mitigations (IMPLEMENTED):**
- strict typed schema enforcement
- DisallowUnknownFields JSON decoding
- capability allowlist enforcement
- forbidden mount path validation
- interpolation rejection (`${`, `{{`, and `$` patterns)
- Hub-side normalization before compile stage

---

### Threat 2: Hub Compromise (ROOT TRUST FAILURE)

**Attacker capability:**
- modifies validation logic or compiler behavior
- alters signing process or policy rules

**Impact:**
- malicious future deployment artifacts
- weakened or bypassed validation rules
- compromised system-wide trust

**Clarification:**
- Hub does NOT execute workloads directly
- compromise affects *generation*, not immediate runtime execution, meaning older deployments will not be compromised, but it can generate malicious deployment artifacts that will be executed when applied by the operator.

**Mitigations (PARTIAL / FUTURE HARDENING):**
- sandboxing the hub runs in a Pinciple of Least Privilege docker container, using distroless image, same as sensors do.
- signed deployment artifacts (planned enforcement expansion)

---

### Threat 3: Wizard Compromise (LOCAL RECON + MISDIRECTION)

**Attacker capability:**
- reads local environment state
- modifies sensor recommendation logic
- submits altered deployment requests
- access to root executable if command execution cve were to be found

**Impact:**
- leakage of local infrastructure metadata
- incorrect sensor recommendations
- operational misconfiguration
- if the Wizard is exploited, it may expose the local host system
- compromise of node identity api key (this is an accepted trust boundary compromise)

**Mitigations (IMPLEMENTED):**
- Wizard is fully untrusted by Hub
- Hub re-validates all inputs independently
- Wizard cannot override Hub-generated output
- no deployment policy logic resides in Wizard

---

### Threat 4: Privilege Escalation via Sensor Deployment Spec Abuse

**Attacker capability:**
- attempts unsafe container configuration via manifest:
  - dangerous capability requests
  - init container misuse
  - host network exploitation
  - unsafe mount paths
  - injection via volume/template fields

**Impact:**
- host compromise via container escape paths
- unauthorized filesystem access
- lateral movement via network exposure

**Mitigations (IMPLEMENTED):**
- capability whitelist enforcement
- forbidden mount path denylist
- strict validation of:
  - init containers
  - volume mounts
  - command fields
- read-only enforcement for main sensor containers
- security-opt enforcement (`no-new-privileges`)

---

### Threat 5: Init Container Abuse (PARTIALLY MITIGATED / CURRENT GAP)

**Attacker capability:**
- uses init containers as privileged bootstrap execution
- writes to mounted host volumes

**Impact:**
- unintended host file modifications (expected behavior for FIM use case)
- potential privilege misuse if mounts are overly broad
- sandbox bypass if mounts are incorrectly restricted

**Current Reality:**
- init containers are intentionally more permissive than sensor runtime
- they are required for provisioning / decoy generation

**Mitigations (PARTIAL):**
- controlled volume mounts only
- strict source path validation
- no arbitrary host filesystem access outside declared mounts

**Future Improvement:**
- explicit role separation:
  - `sensor-runtime` (hardened)
  - `init-provisioner` (writable but scoped)

---

### Threat 6: Supply Chain / Image Compromise

**Attacker capability:**
- modifies container images referenced in manifests

**Impact:**
- malicious runtime behavior inside sensor containers
- silent data exfiltration
- backdoored sensors

**Mitigations (FUTURE):**
- image digest pinning (required improvement)
- registry allowlisting
- image signature verification

---

### Threat 7: Node Api key compromise

**Attacker capability:**
- deploy sensors on compromised node
- send fake events, DOS attack to the hub.

**Impact:**
- creates confusion in security audits
- takes hub down

**Mitigations (FUTURE):**
- Node api key rate limiting

---

## 5. Security Principles

- Hub is the ONLY security enforcement authority
- All inputs are untrusted (Wizard + Manifest registry)
- Security is enforced at compile-time, not runtime
- Deterministic builds are required for auditability
- Capability-based security model (allowlist > denylist where possible)
- Containers are treated as potentially hostile execution units

---

## 6. Security Model Summary

- Manifest → untrusted declarative DSL
- Wizard → untrusted local recommender
- Hub → trusted compiler + policy enforcement
- Sensors → isolated containers in potentially hostile environments

---

## 7. Trust Assumptions (EXPLICIT NON-GUARANTEES)

### 7.1 Hub Assumptions

- Hub is assumed secure at time of operation
- Hub runs in controlled environment
- Hub is protected by authentication and OS controls
- Hub is behind reverse proxy running on HTTPS

If Hub is compromised:
- future deployments MAY be malicious
- validation MAY be bypassed
- signing keys MAY be exposed

Security relies on:
- detection
- auditing
- containment
- recovery

---

### 7.2 Wizard Assumptions

- Wizard is untrusted
- Wizard may be compromised
- Wizard may leak environment metadata

Wizard compromise does not bypass Hub validation, but it may still influence operational decisions and cause unsafe deployments if operators approve or automate its output.

---

### 7.3 Sensor Runtime Assumptions

- Docker isolation is not absolute security
- host systems may be compromised
- containers are potentially hostile

Therefore:
- least privilege is mandatory
- read-only rootfs is default (except init containers, which may require write access depending on provisioning tasks)
- sensors run in distroless docker images

---

### 7.4 Manifest Assumptions

- manifests are fully attacker-controllable
- schema validity does NOT imply safety

Therefore:
- all manifests MUST be validated by Hub
- no trust is granted to structure alone

---

## 8. Current Security Posture

### Strongly Mitigated:
- unsafe capability injection (allowlist enforced)
- basic mount path escape attempts
- interpolation-based injection attempts
- untrusted Wizard influence on policy
- schema ambiguity (strict typing enforced)

### Partially Mitigated:
- init container privilege scope abuse
- Docker runtime escape assumptions

### Not Yet Fully Implemented:
- image digest pinning
- registry signing enforcement
- per-role execution sandbox model
- image supply chain trust
- manifest supply chain trust
- Node apikey rate limiting

---

## 9. Core Security Principle

HoneyWire security is not based on trust.

It is based on:

- strict validation at Hub boundary
- deterministic compilation
- constrained execution environments
- explicit capability allowlists
- separation of untrusted inputs from trusted policy execution