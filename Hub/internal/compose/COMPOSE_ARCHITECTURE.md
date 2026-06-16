# Hub v2.0 Secure Compiler Architecture

## Overview
The `internal/compose` package handles the generation of deterministic, hardened `honeywire-compose.yml` payloads sent to the remote sensor nodes.

The architecture is driven by the principle of **Secure Defaults by Inversion**. Instead of filtering bad things out of an arbitrary map, the compiler explicitly maps specific, allowed schema primitives into a locked-down Compose base.

## The Pipeline

1. **Strict Ingestion** 
   All catalog fetch operations and API preview payloads use `json.NewDecoder` equipped with `DisallowUnknownFields()`. This creates a hard edge boundary—if a payload possesses attributes outside the v2 schema, it is rejected immediately. This enforces structural schema validity, not semantic safety.
2. **The Validator (`security/validate.go`)**
   We assume malicious intent. This module acts as the "bouncer." 
   - Strictly enforces capabilities via an explicit allowlist (e.g. `NET_RAW`, `NET_BIND_SERVICE`, `NET_ADMIN`).
   - Normalizes path structures (`filepath.Clean()`) and checks them against an absolute denylist (e.g., `/proc`, `/sys`, `/var/run/docker.sock`).
   - Recursively verifies volume mounts and init-containers.
   - Blocks untrusted interpolation patterns in fields that are not executed through a safe templating engine.
3. **Secure Environment Composition (`env.go`)**
   The `BuildEnv` pipeline ensures isolated priority overrides. Manifest defaults are loaded, overridden safely by user vars (dropping any attempting to modify forbidden environment fields), and ultimately superseded by statically defined, system-injected constants (e.g., `HW_HUB_KEY`, `HW_SENSOR_ID`).
4. **Versioning and Dual-Manifest Resolution**
   Because nodes now support manual rather than automatic upgrades, a node may run a deprecated legacy version of a sensor. To prevent breaking changes during compose regeneration, the compiler relies on dual-manifest resolution:
   - **Latest Catalog:** Warmed into cache on startup via `index.json`.
   - **Historical Schemas:** Lazy-loaded and permanently cached via `FetchSpecificManifest` (e.g. `hw-sensor-tarpit-v1.0.0.json`) when requested by a legacy node.
5. **The Builder (`builder.go`)**
   The `BuildService` routine converts the strictly-validated models into `ComposeFile` structs.
   - **Immutable Sandboxing**: Unconditionally forces `ReadOnly: true`, `CapDrop: ["ALL"]`, and `SecurityOpt: ["no-new-privileges:true"]`.
   - Structural templating handles file-bind expansions inherently rather than executing string replacement interpolations on manifest strings.
   - Output lists (environment pairs, volume mount paths, initialization chains) are explicitly alphabetically sorted. This guarantees bit-for-bit deterministic deployment manifests every time.