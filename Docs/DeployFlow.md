# Deployment Flow Summary

## User Flow

1. **User clicks "Deploy New Node"** in the Fleet UI
2. **User enters** a node alias and optional tags
3. **Frontend sends** `POST /api/v1/nodes`
4. **Backend creates** a new node row and returns the generated node API key
5. **Frontend refreshes** fleet list and can show the new API key to the operator
6. **User opens** the new node detail screen and chooses a sensor from the catalog
7. **User configures** env vars and clicks "Add to Node"
8. **Frontend sends** `POST /api/v1/nodes/{id}/sensors`
9. **Backend inserts** the sensor into the node and marks the node as pending sync
10. **The node detail UI** shows "Pending Sync"
11. **When the user clicks "Sync Node"** or the node agent polls the compose endpoint, the backend generates a new compose YAML. (this doesn't clear synced status!)

---

## What Gets Created on "Deploy New Node"

### Frontend
- **File:** `FleetView.vue`
- **Function:** `handleDeploySubmit()`
- **Sends:** `alias`, `tags`

### Backend
- **Files:** `nodes.go` → `CreateNode()` / `CreateNode(alias, tagsJSON)`
- **Inserts into `nodes` table:**
  - `id` = node-...
  - `alias`
  - `api_key` = generated hw_key_...
  - `tags`
  - `pending_config` = 0
  - `created_at`, `updated_at`
- **Returns:**
  - `node_id`
  - `apiKey`
  - `alias`

### Result
A new logical node is created with:
- a unique node ID
- a generated node API key
- no sensors yet
- `pending_config = false`

---

## What Gets Created on "Add Sensor to Node"

### Frontend
- **File:** `NodeDetailView.vue`
- **Function:** `handleAddSensorToNode()`
- **Sends:** `sensor_id`, `custom_name`, `config_values`

### Backend
- **Files:** `nodes.go` → `AddNodeSensor()` / `AddSensorToNode(nodeID, sensorID, customName, configValues)`
- **Inserts into `node_sensors` table:**
  - `node_id`
  - `sensor_id`
  - `custom_name`
  - `config_values` # Should the whole manifest / compose file be stored? how will the Node compose generator be able to distinguish sensors and associate them to the catalog's sensors, with the exact same svg icon and configs the /api/v1/compose/generate endpoint "previews" for the user? 
  - `is_silenced` = 0
  - `created_at`, `updated_at`
- **Updates the parent node:** `pending_config = 1`

### Result
- the sensor is now assigned to the node
- the node is flagged as needing a sync

---

## Revision Logic

### Pending Sync Tracking
Every sensor add/edit/remove sets: `nodes.pending_config = 1`

### Active Revision
When compose is applied successfully to the server (phisical node) to make sure the state doesn't drift out of sync (how can that be done?), the backend writes:
- `nodes.active_revision = <new hash>`
- `nodes.pending_config = 0`

### How the Hash Is Created
`generateRevisionHash()` returns: `rev_` + 8-byte hex
This becomes `HW_CONFIG_REV` in generated YAML.

### Summary
- `pending_config` = desired config changed
- `active_revision` = last synced effective config version

---

## Deployment Logic

### Node List and Detail Behavior

**GetNodes()** loads nodes and their installed sensors, including:
- `pending_config`
- derives node status from last heartbeat

**GetNodeDetails()** loads one node plus:
- sensor metadata
- 24h event count

### Sync Action

The UI shows a **"Pending Sync"** banner when `hasPendingConfig` is true.

Clicking **Sync Node** executes:
`
fetch('/api/v1/nodes/compose', { Authorization: Bearer <node api key> })`


*This is the actual node sync / compose generation call.*

---

## How GetNodeCompose Works

### Endpoint
Endpoint
GET /api/v1/nodes/compose

Implementation

Authentication
Uses Authorization: Bearer <api_key>
Looks up node_id by matching api_key in nodes table
If the key is invalid, returns 401

What It Does
Authenticates the node via GetNodeByKey(token)

SHOULD:
have the same exact logic as POST /api/v1/compose/generate, metter of fact it should use that endpoint internally for each sensor, using the config values found in the db (which were set by the user before they added the sensor to the node), the only thing different should be adding the HW_CONFIG_REV env variable.
in doing so it will build the whole services list for the node basically automatically.

what is being done right now for this endpoint completely breaks the node deployment logic.

Result

SHOULD:
The endpoint produces the canonical node deployment bundle
It should not mark the node as "synced", this causes config state drifting, which is why we implemented HW_CONFIG_REV, it should only generate the final node compose file, the "synced" will be set once the config actually gets deployed (there will need to be a check on that, like waiting for heartbeats to show HW_CONFIG_REV and check it is the same for all sensors in the node.)
That's the backend's main manual "deploy" operation.


---

## Notes on Preview vs. Actual Compose

| | **UI Preview** | **Actual Deployment / Sync** |
|---|---|---|
| **Endpoint** | `POST /api/v1/compose/generate` | `GET /api/v1/nodes/compose` |
| **Purpose** | Separate compose generator for manifests and preview, used while configuring a sensor | Authenticated by node API key, persisted revision state and pending flag are updated |

---

## Short Technical Summary

### Deploy New Node
- creates `nodes` row
- generates `api\_key`
- no sensors yet

### Add Sensor to Node
- creates `node\_sensors` row
- sets `nodes.pending\_config = 1`

### Revision Logic
- `pending\_config` tracks unsynced changes
- `active\_revision` tracks last generated config version
- sync clears pending and stores new revision

### Deployment Logic
- node compose endpoint builds YAML from installed sensors