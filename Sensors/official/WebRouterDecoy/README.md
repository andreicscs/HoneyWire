# HoneyWire Official Sensor: Web Router Decoy

The Web Router Decoy sensor emulates a consumer router login panel and captures credentials when an attacker (or curious insider) submits the form. It returns an HTTP 401 response to keep the attacker guessing.

## Features
* **Web canary:** Low-resource trap that looks like a router admin page.
* **Credential capture:** Logs attempted username and password.
* **Deceptive response:** Always returns `401 Unauthorized` to preserve the illusion.
* **HoneyWire-native reporting:** Sends standardized JSON events to the Hub.

## Configuration
Copy `.env.example` to `.env` and update the core HoneyWire settings.

| Variable | Description | Default |
|---|---|---|
| `HW_HUB_ENDPOINT` | HoneyWire Hub API endpoint | `http://192.168.1.100:8080` |
| `HW_HUB_KEY` | HoneyWire API key | `super_secret_key_123` |
| `HW_SENSOR_ID` | Unique sensor identifier | `web-router-decoy-01` |
| `HW_SEVERITY` | Alert severity | `critical` |
| `HW_BIND_PORT` | Container port to listen on | `8080` |
| `HW_ROUTER_BRAND` | Display name for the fake router UI | `Netgear` |

## Deployment
This sensor can use host networking for direct access to the network interface. When using host mode, set HW_BIND_PORT to 80 to listen on the standard web port. Alternatively, use standard Docker bridging and map host port 80 to container port 8080.

Example `docker-compose.yml` (host mode):

```yaml
services:
  web-router-decoy:
    build:
      context: ../../../
      dockerfile: Sensors/official/WebRouterDecoy/Dockerfile
    container_name: hw-web-router-decoy
    network_mode: "host"
    env_file:
      - .env
```

Ensure HW_BIND_PORT=80 in your .env for host mode.

## Event Example
```json
{
  "event_type": "web_login_attempt",
  "severity": "critical",
  "source": "192.168.1.15",
  "target": "Web Interface",
  "metadata": {
    "user_agent": "Mozilla/5.0...",
    "attempted_username": "admin",
    "attempted_password": "password123"
  }
}
```
