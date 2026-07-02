# App Store Architecture (`app.ts`)

## 4-Layer State Model

State in `app.ts` is divided into four distinct categories:

1. **Backend-Synced State (Server Truth):** Absolute source of truth for backend settings (e.g., `isArmed`, `version`). Never optimistically mutated.
2. **Pure UI State (Frontend Owned):** Ephemeral client-only state (e.g., `viewingArchive`, `sidebarOpen`, `currentView`).
3. **Session State Machine:** A three-state model (`'unknown'`, `'authenticated'`, `'unauthenticated'`) to manage identity before UI renders.
4. **Explicit Error State:** Dedicated reactive variables for errors (`bootstrapError`, `authError`, `setupError`).

## Session Transition Authority

All session changes (login, logout, 401s) must pass through a strict `transitionSession` gatekeeper that blocks invalid state transitions. 

## Decoupled System State Fetching

`fetchSystemState` only fetches data. If it encounters a `401 Unauthorized` or `403 Forbidden`, it delegates to the Gatekeeper by calling `transitionSession('unauthenticated')`.

## Reconciled Update Pattern

For critical system flags (like toggling `isArmed`), optimistic UI updates are forbidden:
1. **Dispatch Intent:** Send intended state to server.
2. **Reconcile Reality:** Fetch actual resulting state from server.
3. **UI Update:** Update UI only when the new snapshot replaces local state.

## State-First Error Boundaries

Auth and Setup methods do not return error strings. They act as mutators for dedicated error states (`authError`, `setupError`). Views bind directly to these state variables.

## Bootstrap Orchestration

Initialization (`initAppStore()`) fetches independent system requirements concurrently using `Promise.allSettled`. It delegates findings to the Gatekeeper to safely stall, authenticate, or fall back to setup.
