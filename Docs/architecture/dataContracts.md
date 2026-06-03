# HoneyWire Data Contracts

This document serves as the **single source of truth** for all cross-component data shapes in the HoneyWire platform. To prevent documentation drift, these contracts are defined here and referenced by other subsystems (Hub, Wizard, Sensors, SDKs).

---

## 1. The Universal Event Standard

This is the standard telemetry payload that all sensors (official or community) must POST to the Hub when an intrusion is detected. The Hub's frontend dynamically parses, syntax-highlights, and renders this JSON.

**Endpoint:** `POST /api/v1/event`
**Authentication:** `Authorization: Bearer <HW_NODE_KEY>`

```json
{
  "contractVersion": "1.0",
  "sensorId": "core-dpi-engine",
  "eventTrigger": "malformed_jwt_detected",
  "severity": "critical",
  "source": "104.28.19.12",
  "target": "Auth Gateway",
  "details": {
    "protocol": "TCP",
    "headers_stripped": true,
    "payload_sample": [
      "Authorization: Bearer eyJhbG... [TRUNCATED]",
      "User-Agent: curl/7.64.1"
    ]
  }
}
```

### Field Definitions
- `contractVersion` (string, required): Must match the Hub's supported major version.
- `sensorId` (string, required): The unique identifier of the sensor emitting the event.
- `eventTrigger` (string, required): A brief, machine-readable reason for the event.
- `severity` (string, required): Must be one of `info`, `low`, `medium`, `high`, `critical`.
- `source` (string, required): The attacker's IP, hostname, or identifier.
- `target` (string, required): The decoy service or port that was attacked.
- `details` (object, optional): A flexible JSON object containing forensic artifacts. Primitive values are rendered as tags; arrays are rendered as syntax-highlighted code blocks in the UI.

---

## 2. Heartbeat Payload

This payload is emitted continuously (typically every 30 seconds) by sensors to prove they are alive and running the expected configuration.

**Endpoint:** `POST /api/v1/heartbeat`
**Authentication:** `Authorization: Bearer <HW_NODE_KEY>`

```json
{
  "sensorId": "alpha-node-01",
  "metadata": {
    "agent_version": "1.0.0",
    "contract_version": "1.0",
    "HW_CONFIG_REV": "rev_abc123"
  }
}
```

### Field Definitions
- `sensorId` (string, required): The unique identifier of the sensor.
- `metadata` (object, required):
  - `agent_version` (string): The version of the language SDK or agent.
  - `contract_version` (string): The API contract version in use.
  - `HW_CONFIG_REV` (string): The current configuration revision the sensor is running. Used by the Hub to determine if the node has synchronized its desired state.

---

## 3. Sensor Manifest

The Sensor Manifest is the declarative JSON schema used to describe a decoy. It is consumed by the Wizard to evaluate host heuristics and generate Docker Compose deployments, and by the Hub to render UI metadata.

```json
{
  "id": "hw-tcp-tarpit",
  "version": "1.1.0",
  "schema_version": "1.0",
  "min_wizard_version": "1.0.0",
  "name": "TCP Tarpit",
  "category": "network",
  "osi_layer": "L4",
  "icon_svg": "<svg>...</svg>",
  "description": "Slows down automated network scanners by holding TCP connections open indefinitely.",
  "documentation": {
    "summary": "Deploy this on unused ports to trap automated scanners.",
    "sections": [
      {
        "title": "Operation",
        "type": "markdown",
        "content": ["Holds connections open until timeout."]
      }
    ]
  },
  "heuristics": {
    "triggers": {
      "ports": [22, 23, 3389, 5900]
    },
    "recommendation_reason": "High-value administrative ports are exposed."
  },
  "deployment": {
    "image": "ghcr.io/andreicscs/honeywire-tcp-tarpit:latest",
    "network_mode": "host",
    "user": "65532:65532",
    "cap_add": ["NET_BIND_SERVICE"],
    "port_assignments": [
      {
        "env_var_name": "HW_TARPIT_PORT",
        "default": 2222,
        "auto_shift": true
      }
    ],
    "env_vars": [
      {
        "name": "HW_TARPIT_MODE",
        "description": "Behavior mode: hold, echo, or banner",
        "default": "hold",
        "type": "string",
        "required": true
      }
    ]
  }
}
```

### Key Subsystems
- `heuristics.triggers`: Used by the Wizard Discovery Engine. If the Wizard observes matching `processes`, `ports`, or `file_patterns` on the host, it will recommend this sensor.
- `deployment`: Used by the Wizard Deployment Engine to generate the Intermediate Representation (IR) and final `docker-compose.yml`.
- `deployment.env_vars`: Rendered in the Hub UI so users can configure the sensor dynamically.
 Rendered in the Hub UI so users can configure the sensor dynamically.
