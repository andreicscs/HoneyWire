# App Store Architecture Guide

This document describes the architectural design, state segregation, and session management strategies used in the `app.js` Pinia store.

The `app.js` store serves as the global application orchestrator. It manages the boot sequence, identity session, and critical system flags. It has been strictly engineered to separate frontend-owned UI state from backend-owned truths, utilizing defensive data-fetching, explicit error boundaries, and a strict Session Transition Authority.

---

# Architecture Overview

## The 4-Layer State Model

State within `app.js` is strictly divided into four semantic categories to prevent UI drift and side effects.

### 1. Backend-Synced State (Server Truth)
The absolute source of truth for global backend settings. These are **never** optimistically mutated. 
```javascript
const isArmed = ref(true)
const version = ref('1.0.0')
```

### 2. Pure UI State (Frontend Owned)
Ephemeral state owned entirely by the client. Fully synchronous and never requires server validation.
```javascript
const viewingArchive = ref(false)
const sidebarOpen = ref(true)
const currentView = ref('dashboard')
```

### 3. Session State Machine & Bootstrap
Utilizes a formal three-state session model rather than a naive boolean, allowing the UI to differentiate between "checking identity" and "not logged in."
```javascript
const sessionState = ref('unknown') // 'unknown' | 'authenticated' | 'unauthenticated'
const isAuthenticated = computed(() => sessionState.value === 'authenticated')
const requiresSetup = ref(false)
const isInitialized = ref(false)
```

### 4. Explicit Error State
Errors are managed state-first. UI components react to these variables rather than parsing ephemeral function return values.
```javascript
const bootstrapError = ref(null) 
const authError = ref(null)
const setupError = ref(null)
```

---

# 1. The Session Transition Authority

Traditional applications often use a simple `isAuthenticated: false` flag, which causes UI layout pop-in because the app assumes the user is logged out before the initial network request finishes, or scatters auth logic across many API fetchers.

HoneyWire uses a **Session State Machine** governed by a strict Gatekeeper (`transitionSession`):
- `sessionState` begins as `'unknown'`.
- The app stalls the router/UI initialization while `sessionState` is `'unknown'`.
- **All** session changes (login, logout, 401s during background polling) must pass through `transitionSession`. This prevents race conditions, such as a lagging bootstrap overwriting a successful concurrent login.

```javascript
const transitionSession = (nextState, reason = 'Implicit') => {
  if (sessionState.value === nextState) return

  // Prevent a lagging bootstrap from overwriting a successful login
  if (sessionState.value === 'authenticated' && nextState === 'unknown') {
    return
  }

  console.info(`[AppStore] Session Transition: ${sessionState.value} -> ${nextState} (Reason: ${reason})`)
  sessionState.value = nextState
}
```

---

# 2. Decoupled System State Fetching

`fetchSystemState` is a pure data-fetching function. It does not dictate application flow or own authentication logic. If it encounters a `401 Unauthorized` or `403 Forbidden` from the backend, it simply reports this to the Gatekeeper. 

This ensures that any API failure drops the user to a safe, unauthenticated state immediately, without crashing the frontend.

```javascript
const fetchSystemState = async () => {
  try {
    // ... fetch and commit ...
  } catch (err) {
    if (err.status === 401 || err.status === 403) {
      transitionSession('unauthenticated', 'System fetch received 401/403')
    }
  }
}
```

---

# 3. The "Reconciled Update" Pattern

For critical system flags (like toggling the `isArmed` status of the entire security ecosystem), the store completely avoids Optimistic UI updates. 

If the UI pretends the system is armed but the backend fails to apply it, the user is left with a false sense of security. Instead, `app.js` relies on **Reconciled Updates**:

1. **Dispatch Intent:** Send the user's intended state to the server.
2. **Reconcile Reality:** Regardless of success or failure, fetch the *actual* resulting state from the server.
3. **UI Update:** The UI only updates when the newly fetched snapshot replaces the local state.

```javascript
const toggleArmed = async () => {
  const targetState = !isArmed.value
  
  try {
    await api.patch('/api/v1/system/state', { is_armed: targetState })
    // Reconcile reality
    await fetchSystemState()
  } catch (err) {
    // Reality check fallback
    await fetchSystemState()
  }
}
```

---

# 4. State-First Error Boundaries

Authentication and Setup routines do not return complex error strings directly to components. Instead, they act as mutators for dedicated error states.

This decouples the form submission logic from the error rendering logic:

```javascript
// In the Store:
const login = async (password) => {
  authError.value = null // Clear previous
  try {
    // ...
    transitionSession('authenticated', 'Login successful')
    return { success: true }
  } catch (err) {
    transitionSession('unauthenticated', 'Login failed')
    authError.value = 'Invalid credentials'
    return { success: false }
  }
}
```
*Rule:* Views bind directly to `appStore.authError`. The component's `onSubmit` handler only needs to check `{ success: boolean }`.

---

# 5. Bootstrap Orchestration

Initialization is managed by a single orchestrator: `initAppStore()`. 

It fetches multiple independent system requirements concurrently using `Promise.allSettled`. This ensures that a failure in a non-critical endpoint (e.g., a version check) does not fatally crash the critical system state ingestion. 

The bootstrap explicitly delegates its findings to the Gatekeeper, allowing the application to safely stall, authenticate, or fall back to the setup screen.