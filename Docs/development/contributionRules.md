# Contribution Rules

To maintain a stable, secure, and cohesive platform, all contributions to HoneyWire must adhere to the following rules. Pull Requests that violate these guidelines will not be merged.

## 1. Code Style and Architecture

### General
*   **No Global State:** Dependency injection must be used across all Go components. Do not use `init()` functions to instantiate external dependencies or global variables.
*   **Error Handling:** Never swallow errors silently. Always return errors up the call stack or handle them explicitly with appropriate logging.
*   **Variable Naming Conventions:** Variable names that go on disk (e.g., JSON payloads, database columns, config files) must use `snake_case`. Variable names that stay in memory (e.g., Go structs, TypeScript interfaces, local variables) must use `camelCase`.

### Go (Backend, Wizard, SDKs)
*   **Version:** Pure Go 1.25. No CGO is permitted.
*   **Formatting:** All code must be formatted using `go fmt`.
*   **Separation of Concerns (Hub):** HTTP handlers (`internal/api/`) must contain *zero* business logic. All logic must reside in `internal/services/`.

### Vue 3 (Frontend)
*   **Composition API:** Use `<script setup>` and Vue 3 Composition API exclusively. Do not use the Options API.
*   **State Mutability:** Never reassign arrays or objects in Pinia stores (e.g., `events.value = newEvents`). Mutate them in-place (e.g., `events.value.push(...)`) to preserve Vue's reactive identity.
*   **Styling:** Use TailwindCSS utility classes. Avoid custom CSS files unless absolutely necessary.

## 2. Schema and Data Contracts

**⚠️ WARNING: Do NOT break schema contracts.**

HoneyWire relies on strict JSON contracts to communicate across distributed components. Any changes to API payloads must be backward compatible.

*   Before modifying telemetry shapes, review the [Data Contracts](/Docs/architecture/dataContracts.md).
*   Sensors and the Hub must agree on the standard. If you introduce a new field to an event, ensure the frontend can render it without crashing if the field is missing from older events.

## 3. Testing Requirements

All submissions must pass automated CI checks.

*   **Unit Tests:** New business logic in the Go backend (`internal/services/`) requires accompanying `_test.go` files.
*   **Test Mode Compliance:** If you submit a new sensor, it must support `HW_TEST_MODE=true`. When this flag is passed, the container must immediately fire a synthetic alert to the Hub and exit with code 0.
*   **Security Scanning:** Your PR must pass automated CodeQL (or Semgrep) static analysis and Trivy container vulnerability scans. 

## 4. Pull Request Expectations

1.  **Draft Early:** Open a Draft PR if you want early feedback on an architectural approach.
2.  **Scope:** Keep PRs focused. Do not combine unrelated refactoring with feature additions.
3.  **Naming Convention:** Use descriptive titles (e.g., `fix(hub): resolve websocket memory leak on disconnect`).
4.  **Description:** Your PR description must include:
    *   What the PR does.
    *   Why it is necessary.
    *   How it was tested locally.

## 5. Security Contributions

If you discover a vulnerability in HoneyWire, **do not open a public issue or PR.** 

Please use GitHub Private Vulnerability Reporting via the Security tab on the repository, as outlined in [SECURITY.md](/SECURITY.md).