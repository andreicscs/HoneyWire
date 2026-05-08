# HoneyWire Frontend Architecture & State Management Guide

This document outlines the architecture, data flow, and debugging protocols for the HoneyWire v2.0.0 Vue 3 frontend. The application utilizes a decoupled, three-layer architecture utilizing Pinia for state management and native JavaScript classes for network services.

## 1. Architectural Overview

The frontend is strictly separated into three layers to ensure maintainability, testability, and separation of concerns.

* **Layer 1: Presentation (Vue Components)**
Located in `src/components/` and `src/views/`. These files contain zero business logic and make no direct API calls. They exist solely to read reactive state from Pinia stores and render HTML/CSS. User interactions (clicks, inputs) trigger Pinia store actions.
* **Layer 2: State Management (Pinia Stores)**
Located in `src/stores/`. This is the brain of the application. Stores hold global variables, execute business logic, and handle HTTP requests.
* **Layer 3: Services (Network Layer)**
Located in `src/services/`. These are framework-agnostic classes that handle persistent connections (e.g., WebSockets). They contain no reactive Vue variables and communicate with the State layer via callbacks.

## 2. Unidirectional Data Flow

To prevent race conditions and layout thrashing, HoneyWire enforces a strict unidirectional data flow:

1. **Trigger:** A user clicks a UI element (e.g., selecting a node).
2. **Action:** The component calls a Pinia action (e.g., `fleetStore.selectTarget(nodeId)`).
3. **Mutation/Request:** The Pinia action updates its local state or makes an HTTP request.
4. **Reaction:** Vue Watchers in the main orchestrator (`App.vue`) detect the state change and trigger subsequent data fetches if necessary.
5. **Render:** The stores update their data arrays, and Vue automatically repaints the dependent components.

## 3. The State Domains (Pinia Stores)

State is divided into three domain-specific stores.

### A. App Store (`src/stores/app.js`)

Manages the global user interface and application-level configuration.

* **State:** `currentView`, `sidebarOpen`, `viewingArchive`, `isArmed`, `version`.
* **Actions:** `toggleArmed()`, `logout()`.

### B. Fleet Store (`src/stores/fleet.js`)

Manages infrastructure assets (Nodes and Sensors) and their uptime metrics.

* **State:** `sensors`, `uptimeData`, `selectedNode`, `selectedSensor`, `activeTimeframe`.
* **Key Concept (Composite Keys):** Sensors are uniquely identified by a composite key consisting of both `node_id` AND `sensor_id`. All array filtering, finding, and deletion logic within this store strictly enforces this composite check to prevent cross-node collisions.
* **Actions:** `fetchFleet()`, `fetchUptime()`, `selectTarget()`, `forgetSensor()`, `toggleSilence()`, `handleWsUpdate()`.

### C. Events Store (`src/stores/events.js`)

Manages telemetry, threat alerts, and event archiving.

* **State:** `events`, `unreadCount`, `isFetching`.
* **Getters:** `filteredEvents` (Evaluates the base `events` array against the `viewingArchive` state from the App Store and the `selectedNode`/`selectedSensor` state from the Fleet Store).
* **Actions:** `fetchEvents()`, `markAllRead()`, `archiveEvent()`, `archiveAll()`, `purgeEvents()`, `handleWsEvent()`.

## 4. The Orchestrator (`App.vue`)

`App.vue` serves as the application's central nervous system. It handles initialization and wires the Service layer to the State layer.

* **Initialization:** On mount, it checks authentication. If valid, it triggers the initial `fetchFleet`, `fetchUptime`, and `fetchEvents` actions.
* **Service Routing:** It instantiates the `HoneyWireWS` class and maps its event emitters directly to the appropriate Pinia store actions (e.g., routing 'NEW_EVENT' payloads to `eventsStore.handleWsEvent`).
* **Cross-Store Reactivity:** It contains the root-level Vue `watch` functions. For example, it watches `fleetStore.selectedNode` and calls `eventsStore.fetchEvents()` when a change is detected.

## 5. Debugging Guide

When encountering UI or state issues, follow this structured troubleshooting protocol.

### Scenario A: A button click does not update the UI

1. **Check the Component:** Verify the component is calling the correct store action without attempting to mutate the state directly.
2. **Check the Store Action:** Open the respective store and ensure the action is properly updating the state variable (remembering to use `.value` for refs in composition API stores).
3. **Check the Watcher:** If the UI update requires an API call (e.g., clicking the "7D" timeframe), verify that `App.vue` has an active watcher on that specific state variable to trigger the fetch.

### Scenario B: Filtering or selecting sensors behaves erratically

1. **Verify Composite Keys:** Ensure the UI is passing both `node_id` and `sensor_id` to the store action.
2. **Check Action Logic:** Review `selectTarget` in `fleet.js`. Ensure it explicitly handles the difference between clicking a specific sensor (requires clearing both node and sensor on toggle) versus clicking a node group.
3. **Check the Getter:** Review `filteredEvents` in `events.js`. Ensure it is pulling the correct selection state from the Fleet Store before filtering the events array.

### Scenario C: Real-time events are not appearing

1. **Check Network Tab:** Verify the WebSocket connection (`/api/v1/ws`) is returning a `101 Switching Protocols` status.
2. **Check the Service Router:** In `App.vue`, ensure `wsService.on('onNewEvent', ...)` is actively passing payloads to `eventsStore.handleWsEvent`.
3. **Check Filter State:** A new event will only appear in the active queue if it matches the current UI filters. Verify that `handleWsEvent` in `events.js` evaluates the incoming payload's `node_id` against the currently selected filters before unshifting it into the array.