# THREAT MODEL — HoneyWire

## 1. System Overview

HoneyWire is a decentralized sensor deployment system composed of:

- **Wizard (Untrusted Local Orchestration Layer)**
  Performs:
  - host discovery
  - manifest selection
  - manifest transformation (env/config binding)
  - deployment request construction
  - compose generation request input

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

- **Deployment Intent (Wizard-generated intermediate artifact)**
  Partially transformed manifest produced by the Wizard before Hub validation.
  Includes:
  - selected sensors
  - modified environment variables
  - host-specific configuration bindings

  This artifact is NOT trusted and is treated as attacker-controllable input by the Hub.

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

Hub must validate both:
- raw registry manifests
- Wizard-generated deployment intents (modified manifests)

**Scope limitation:**
Hub validation governs the *content it generates and returns*. It has no enforcement over what the Wizard does with that output after delivery. Post-delivery integrity (writing to disk, executing `docker compose up`) is outside the Hub's control and is instead bounded by host-level trust (see §7.2).

---

### 2.2 Wizard Trust Boundary (UNTRUSTED LOCAL COMPONENT)

The Wizard is NOT part of the security enforcement chain,
but it IS part of the deployment transformation chain.

It:
- observes local system state (processes, ports, services)
- generates sensor recommendations
- submits deployment requests to the Hub
- **receives Hub-generated compose output and writes it to `honeywire-compose.yml`**
- **executes `docker compose up` against that file**

The Wizard:
- may be compromised locally
- may leak environment metadata
- may be manipulated to produce misleading recommendations
- **may discard, modify, or replace Hub-generated compose output before execution**

However:
> Wizard output affects deployment inputs (configuration, sensor selection, environment values), but all outputs are re-validated by the Hub before final compose generation.

**Important limitation:**
> Hub validation protects the integrity of what the Hub *generates*. It does not protect against a compromised Wizard tampering with Hub output *after delivery* discarding it, modifying it, replaying a stale compose, or substituting an attacker-controlled file entirely. This post-delivery gap is explicitly acknowledged and accepted: the precondition for exploiting it (filesystem write access + Docker socket access on the host) is equivalent to full host compromise, which is addressed as a separate threat boundary (§7.2, Threat 3).

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

### Threat 1.5: Wizard-Manipulated Deployment Intent

Attacker capability:
- compromises Wizard OR influences its execution environment
- modifies manifests after retrieval from Hub registry
- injects unsafe environment variables or configuration overrides
- selects or combines sensors in unintended ways

Impact:
- valid manifests become unsafe at deployment time
- policy bypass via "schema-valid but semantically unsafe" configurations
- subtle privilege escalation via configuration injection

Mitigations:
- Hub treats Wizard-generated deployment intents as fully untrusted, regardless of provenance (including registry-derived manifests)
- full re-validation of all manifests and overrides
- capability + mount + env validation enforced at compile stage

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
- sandboxing the hub runs in a Principle of Least Privilege docker container, using distroless image, same as sensors do.
- signed deployment artifacts (planned enforcement expansion)

---

### Threat 3: Wizard Compromise (LOCAL RECON + MISDIRECTION)

**Attacker capability:**
- reads local environment state
- modifies sensor recommendation logic
- submits altered deployment requests
- access to root executable if command execution CVE were to be found
- **discards or modifies Hub-generated compose output after Hub validation**
- **writes and executes an attacker-controlled `honeywire-compose.yml`**
- **replays a previously valid but stale compose file**

**Impact:**
- leakage of local infrastructure metadata
- incorrect sensor recommendations
- operational misconfiguration
- if the Wizard is exploited, it may expose the local host system
- compromise of node identity api key (this is an accepted trust boundary compromise)
- **deployment of attacker-controlled containers regardless of Hub validation result**

**Important scope clarification, post-delivery compose tampering:**
A compromised Wizard can bypass Hub validation entirely by ignoring the Hub's compose output and executing an attacker-controlled file. Hub validation has no post-delivery enforcement capability.

However, this attack is **explicitly accepted as subsumed by host compromise**. The preconditions required, write access to `honeywire-compose.yml` and access to the Docker socket, are equivalent to full root-level host access. An attacker with those capabilities has no need to abuse HoneyWire's deployment pipeline; they already own the host. The relevant threat boundary at that point is host security, not HoneyWire-specific controls.

Hub validation therefore protects against:
- malicious manifests reaching the compile stage
- cross-node compromise (a Wizard on Node A cannot craft input that causes the Hub to generate a dangerous compose for Node B)
- supply chain attacks where the attacker controls manifests but not a host

Hub validation does NOT protect against:
- a locally compromised Wizard choosing not to use Hub output

**Mitigations (IMPLEMENTED):**
- Wizard is fully untrusted by Hub
- Hub re-validates all inputs independently
- no deployment policy logic resides in Wizard
- port range validation prevents local DoS from malformed `/proc` state

**Accepted Risk:**
Post-delivery compose tampering by a compromised Wizard is accepted as out of scope for HoneyWire-specific controls. It is bounded by host-level compromise, which is a prerequisite for the attack and represents a more severe, pre-existing failure condition.

---

### Threat 4: Privilege Escalation via Sensor Deployment Spec Abuse

**Attacker capability:**
- attempts unsafe container configuration via manifest:
  - dangerous capability requests
  - init container misuse
  - host network exploitation
  - unsafe mount paths
  - injection via volume/template fields
  - modifies Wizard-generated deployment intent before Hub validation
  - modifies Hub-generated compose output after Hub validation (see Threat 3 subsumed by host compromise)

**Impact:**
- host compromise via container escape paths
- unauthorized filesystem access
- lateral movement via network exposure

**Mitigations (IMPLEMENTED):**
- capability whitelist enforcement
- forbidden mount path denylist
- validation of:
  - init containers
  - volume mounts
  - command fields
- read-only enforcement for main sensor containers
- security-opt enforcement (`no-new-privileges`)
- validation of dynamic volume paths during generation

**Gaps:**
- no strict command checking for init-provisioner containers
- post-delivery compose tampering is not mitigated at the HoneyWire layer (accepted, see Threat 3)

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

### Threat 6: Supply Chain Compromise (Images + Manifests)

**Image Supply Chain Risks**
- Attacker modifies container image at registry (requires registry compromise or MITM)
- Manifest specifies image without digest pinning
- Wizard deploys attacker's image

**Manifest Supply Chain Risks**
- Attacker modifies manifest at Hub or in manifest registry
- Hub lacks manifest signing verification
- Invalid manifest reaches Wizard

**Mitigations (Planned)**
- Image digest pinning (required in manifests)
- Manifest signature verification at Hub
- Provenance tracking

**Current Gaps**
- No image digest enforcement (images pulled by tag only)
- No manifest cryptographic signature verification

---

### Threat 7: Node Api key compromise

**Attacker capability:**
- deploy sensors on compromised node
- send fake events, DOS attack to the hub.

**Impact:**
- creates confusion in security audits
- takes hub down

**Accepted Risk (Command History Exposure):**
During the initial link, the wizard requires `--link` and `--api-key` as CLI arguments.
`wizard --link https://hub.example.com --api-key eyJhbGc...`
The API key is visible in `ps` output and shell history.

This trade-off (Medium Risk - UX Issue) is formally accepted because:
- It is a one-time setup operation (after that, the config file is used)
- The operator is assumed to be in a controlled environment
- It is standard practice for unattended bootstrap operations

**Mitigations (IMPLEMENTED):**
- **Node API Key Rate Limiting:** The Hub enforces a per-node token bucket rate limit (100 requests/minute) on all API endpoints that use node API keys for authentication. This prevents a compromised key from being used in a high-volume Denial of Service (DoS) attack against the Hub.

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
- Wizard → untrusted local orchestration + transformation layer
- Hub → trusted policy enforcement + compilation engine (defensive against untrusted inputs)
- Sensors → isolated containers in potentially hostile environments

**What Hub validation does and does not guarantee:**
Hub validation ensures the integrity of compose artifacts *as generated*. It does not provide post-delivery enforcement. A compromised Wizard may ignore Hub output entirely. This is accepted: the precondition for doing so is full host compromise, which is a separate and more severe failure condition outside HoneyWire's control plane.

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
- **Wizard has full control over what gets written to `honeywire-compose.yml` and executed on the host — Hub validation does not constrain this**

Wizard compromise does not bypass Hub validation *at the Hub*, but a compromised Wizard can choose to ignore Hub-generated output entirely after delivery. At this point the threat is equivalent to general host compromise: the attacker already has filesystem write access and Docker socket access, which implies full root-level control of the host. HoneyWire-specific controls are no longer the relevant security boundary.

Wizard compromise may still influence operational decisions and cause unsafe deployments if operators approve or automate its output.

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
- dynamic volume path expansion via environment variables
- untrusted Wizard influence on policy *at the Hub*
- schema ambiguity (strict typing enforced)
- port number parsing validation
- Node apikey rate limiting
- UI login rate limiting

### Partially Mitigated:
- init container privilege scope abuse
- Docker runtime escape assumptions

### Not Yet Implemented:
- image digest pinning
- registry signing enforcement
- per-role execution sandbox model
- image supply chain trust
- manifest supply chain trust

### Accepted / Out of Scope:
- **post-delivery compose tampering by compromised Wizard** — requires full host compromise as a precondition; subsumed by host-level threat boundary (see Threat 3, §7.2)


---

## 9. Core Security Principle

HoneyWire security is not based on trust.

It is based on:

- strict validation at Hub boundary
- deterministic compilation
- constrained execution environments
- explicit capability allowlists
- separation of untrusted inputs from trusted policy execution