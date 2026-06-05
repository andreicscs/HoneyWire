# App Store Architecture Guide

This document describes the architectural design, state segregation, and session management strategies used in the `app.ts` Pinia store.

The `app.ts` store serves as the global application orchestrator. It manages the boot sequence, identity session, and critical system flags. It has been strictly engineered to separate frontend-owned UI state from backend-owned truths, utilizing defensive data-fetching, explicit error boundaries, and a strict Session Transition Authority.

---

# Architecture Overview

## The 4-Layer State Model

State within `app.ts` is managed in a single reactive `state` object, but is conceptually divided into categories. It is exposed to components via `computed` getters to prevent direct mutation and encapsulate the store's internal structure.

### 1. Backend-Synced State (Server Truth)
The absolute source of truth for global backend settings. These are **never** optimistically mutated. 

const state = ref<AppState>({
  isArmed: true,
  version: '1.0.0',
  // ...
})

### 2. Pure UI State (Frontend Owned)
Ephemeral state owned entirely by the client. Fully synchronous and never requires server validation.

const state = ref<AppState>({
  viewingArchive: false,
  sidebarOpen: true,
  currentView: 'dashboard', // 'dashboard' | 'fleet' | 'settings' | 'node-details'
  // ...
})

### 3. Session State Machine & Bootstrap
Utilizes a formal three-state session model rather than a naive boolean, allowing the UI to differentiate between "checking identity" and "not logged in."

const state = ref<AppState>({
  sessionState: 'unknown', // 'unknown' | 'authenticated' | 'unauthenticated'
  requiresSetup: false,
  isInitialized: false,
  // ...
})

### 4. Explicit Error State
Errors are managed state-first. UI components react to these variables rather than parsing ephemeral function return values.

const state = ref<AppState>({
  bootstrapError: null,
  authError: null,
  setupError: null,
})

---

# 1. The Session Transition Authority

Traditional applications often use a simple `isAuthenticated: false` flag, which causes UI layout pop-in because the app assumes the user is logged out before the initial network request finishes, or scatters auth logic across many API fetchers.

HoneyWire uses a **Session State Machine** governed by a strict Gatekeeper (`transitionSession`):
- `sessionState` begins as `'unknown'`.
- The app stalls the router/UI initialization while `sessionState` is `'unknown'`.
- **All** session changes (login, logout, 401s during background polling) must pass through `transitionSession`. This prevents race conditions, such as a lagging bootstrap overwriting a successful concurrent login.
- The transition function uses an explicit transition matrix to block invalid state changes (e.g., authenticated -> authenticated).

const transitionSession = (nextState: SessionState): void => {
  if (state.value.sessionState === nextState) return

  const validTransitions: Record<SessionState, SessionState[]> = {
    unknown: ['authenticated', 'unauthenticated'],
    authenticated: ['unauthenticated'],
    unauthenticated: ['authenticated']
  }

  if (!validTransitions[state.value.sessionState].includes(nextState)) return
  state.value.sessionState = nextState
}

---

# 2. Decoupled System State Fetching

`fetchSystemState` is a pure data-fetching function. It does not dictate application flow or own authentication logic. If it encounters a `401 Unauthorized` or `403 Forbidden` from the backend, it simply reports this to the Gatekeeper. 

This ensures that any API failure drops the user to a safe, unauthenticated state immediately, without crashing the frontend.

const fetchSystemState = async () => {
  try {
    // ... fetch and commit ...
  } catch (err) {
    if (err.status === 401 || err.status === 403) {
      transitionSession('unauthenticated')
    }
  }
}

---

# 3. The "Reconciled Update" Pattern

For critical system flags (like toggling the `isArmed` status of the entire security ecosystem), the store completely avoids Optimistic UI updates. 

If the UI pretends the system is armed but the backend fails to apply it, the user is left with a false sense of security. Instead, `app.ts` relies on **Reconciled Updates**:

1. **Dispatch Intent:** Send the user's intended state to the server.
2. **Reconcile Reality:** Regardless of success or failure, fetch the *actual* resulting state from the server.
3. **UI Update:** The UI only updates when the newly fetched snapshot replaces the local state.

const toggleArmed = async () => {
  const targetState = !state.value.isArmed
  
  try {
    await api.patch('/api/v1/system/state', { isArmed: targetState })
    // Reconcile reality
    await fetchSystemState()
  } catch (err) {
    // Reality check fallback
    await fetchSystemState()
  }
}

---

# 4. State-First Error Boundaries

Authentication and Setup routines do not return complex error strings directly to components. Instead, they act as mutators for dedicated error states.

This decouples the form submission logic from the error rendering logic:

// In the Store:
const login = async (password) => {
  state.value.authError = null // Clear previous
  try {
    // ...
    transitionSession('authenticated')
    return { success: true }
  } catch (err) {
    transitionSession('unauthenticated')
    state.value.authError = 'Invalid credentials'
    return { success: false }
  }
}

*Rule:* Views bind directly to `appStore.authError`. The component's `onSubmit` handler only needs to check `{ success: boolean }`.

---

# 5. Bootstrap Orchestration

Initialization is managed by a single orchestrator: `initAppStore()`. 

It fetches multiple independent system requirements concurrently using `Promise.allSettled`. This ensures that a failure in a non-critical endpoint (e.g., a version check) does not fatally crash the critical system state ingestion. 

The bootstrap explicitly delegates its findings to the Gatekeeper, allowing the application to safely stall, authenticate, or fall back to the setup screen.