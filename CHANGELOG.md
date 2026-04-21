# HoneyWire v1.1.1

## Bug Fixes & Resilience Improvements

This release focuses entirely on frontend stability, error handling, and resolving "silent failures" in the UI. No new features or database migrations are included.

**State Synchronization:**
- **Fixed:** UI components now properly await server confirmation before updating their state (Optimistic UI desyncs). Actions like arming the system, archiving events, or purging logs will no longer falsely report success if the backend or network fails.
- **Fixed:** Added explicit error catching and state restoration across 7 core API functions (`toggleSilence`, `markEventRead`, `archiveEvent`, etc.). 

**Performance & Resources:**
- **Fixed:** Added **Exponential Backoff** to the real-time WebSocket connection to prevent the frontend from hammering the Hub with reconnection attempts during server downtime or restarts.
- **Fixed:** Implemented `onUnmounted` lifecycle hooks to properly destroy WebSocket connections and health-sync intervals when leaving the application, preventing browser memory leaks over long sessions.

**Accessibility (a11y) & UX:**
- **Added:** Screen-reader friendly `aria-labels` and keyboard-navigable `tabindex` attributes to 15+ interactive elements and icon-only buttons across the dashboard.
- **Added:** Explicit `type="button"` attributes to prevent accidental form submissions.
- **Improved:** Settings configuration saves and API failures now display persistent, contextual error messages pointing users to server logs, rather than silently failing.

---

*Changelog: v1.1.0 → v1.1.1 | 2026-04-21*
