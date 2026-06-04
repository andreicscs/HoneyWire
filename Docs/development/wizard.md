# Wizard Extensibility Guide

The Wizard is HoneyWire's ephemeral, host-side orchestration engine. This guide covers how to navigate its architecture to add new commands, modify discovery logic, or debug deployment behavior.

For a conceptual overview of how the Wizard operates, see `Docs/architecture/wizard/overview.md`.

---

## 1. Adding a New CLI Command

The Wizard CLI is strictly organized into operational commands (`apply`, `discover`, `status`, `relink`, `uninstall`) and bootstrap commands (`--link`).

If you need to add a new command (e.g., `wizard verify`):

1.  **Create the Command File:**
    Navigate to `internal/commands/` and create `audit.go`.
2.  **Define the Execution Flow:**
    Use the internal UI library (`internal/cli/`) for all terminal outputs to maintain consistent HoneyWire branding (prompts, spinners, tables).
3.  **Register the Command:**
    Open `cmd/wizard/main.go`.
    Add your new command to the argument parsing switch statement, ensuring the help text is updated accordingly.

---

## 2. Modifying the Scanner (`core/scanner`)

The Scanner is responsible for low-level Linux inspection. It reads `/proc`, maps open sockets to processes, and queries the local Docker daemon for containerized services.

**⚠️ Architectural Rule: Passive Discovery Only**
The Wizard must not tamper with the host. Discovery should rely strictly on passive or minimal interaction mechanisms (e.g., reading read-only `/proc` files, querying the Docker daemon API). Do not introduce active probing, port knocking, or state-mutating scans.

If you need to detect a new type of host artifact (e.g., parsing static Nginx/Traefik configuration files or querying containerd):

1.  **Implement the Inspection Logic:**
    Create a new file in `core/scanner/` (e.g., `nginx_scanner.go`).
    Ensure your logic fails gracefully. If the Wizard is run on an OS where a specific directory or socket does not exist, it should return an empty slice, not a fatal error.
2.  **Update the Service Inventory:**
    Ensure your new scanner correctly maps its findings into the standard `ServiceInventory` struct so the Discovery Engine can interpret it.

---

## 3. Modifying Discovery Logic (`core/discovery`)

The Discovery Engine acts as the bridge between the raw output of the Scanner and the `heuristics` block defined in sensor manifests.

**How to Trace a Recommendation:**
1. The Engine retrieves the target manifest from the Hub.
2. It extracts the `heuristics` block (e.g., `"processes": ["sshd"]`).
3. It iterates over the `ServiceInventory` provided by the Scanner.
4. If a match occurs, it generates a `Recommendation`.

If you want to add a new heuristic type (e.g., matching by file presence rather than active port):
1. Update the `Heuristics` struct in `core/api/models.go` to support your new JSON field.
2. Add the evaluation logic to `core/discovery/evaluator.go`.

---

## 4. Debugging Deployment Generation

The Wizard is an untrusted client; it generates a "Deployment Intent" (a partially transformed manifest) and sends it to the Hub for compilation.

If the `docker-compose.yml` generated on the host is incorrect:
1. Check `internal/deploy/generator.go` to see how the Wizard applies local environment variable overrides.
2. Check the Hub's compiler backend (`internal/services/compiler.go`) — the Hub may be intentionally stripping your volume mounts or configuration fields if they violate the strict security sandboxing rules.