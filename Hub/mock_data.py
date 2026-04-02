"""Generate demo sensors + events for HoneyWire UI showcase."""

import os
import requests
import random
import time

HUB_URL = "http://localhost:8080"
API_SECRET = "super_secret_key_123"
HEADERS = {"x-api-key": API_SECRET, "Content-Type": "application/json"}

SENSORS = [
    {"sensor_id": "alpha-node-01", "sensor_type": "tarpit"},
    {"sensor_id": "canary-node-77", "sensor_type": "canary_token"},
    {"sensor_id": "ids-node-03", "sensor_type": "ids_sentinel"},
    {"sensor_id": "honeypot-web", "sensor_type": "web_honeypot"},
    {"sensor_id": "llm-probe-9", "sensor_type": "llm_probe"},
]

EVENT_TYPES = [
    ("tcp_connection", "critical", "10.10.0.5", "Port 2222", "blocked"),
    ("file_access", "high", "192.168.1.55", "/var/www/html/index.php", "alert_only"),
    ("login_attempt", "medium", "172.16.0.4", "/admin", "logged"),
    ("scan_detected", "low", "8.8.8.8", "Subnet 10.0.0.0/24", "logged"),
    ("api_fuzz", "info", "127.0.0.1", "/api/v1/login", "logged"),
]


def get_project_version():
    version = os.getenv("HONEYWIRE_VERSION")
    if version:
        return version
    try:
        with open(os.path.join(os.path.dirname(__file__), "..", "VERSION"), "r") as f:
            return f.read().strip()
    except FileNotFoundError:
        return "1.0.0"


def post_heartbeat(sensor):
    body = {
        "sensor_id": sensor["sensor_id"],
        "sensor_type": sensor["sensor_type"],
        "metadata": {
            "version": get_project_version(),
            "os": random.choice(["Linux", "Windows", "FreeBSD"]),
            "region": random.choice(["us-east-1", "eu-west-2", "ap-southeast-1"]),
        },
    }
    r = requests.post(f"{HUB_URL}/api/v1/heartbeat", json=body, headers=HEADERS, timeout=5)
    r.raise_for_status()


def post_event(sensor, event):
    event_body = {
        "sensor_id": sensor["sensor_id"],
        "sensor_type": sensor["sensor_type"],
        "event_type": event[0],
        "severity": event[1],
        "source": event[2],
        "target": event[3],
        "action_taken": event[4],
        "details": {
            "notes": f"Auto-generated mock event for {sensor['sensor_id']}",
            "payload": ["MALFORMED_REQUEST", "PAYLOAD_STRIP", "AUTH_FAILURE"],
        },
    }
    r = requests.post(f"{HUB_URL}/api/v1/event", json=event_body, headers=HEADERS, timeout=5)
    r.raise_for_status()


def main():
    print("Posting heartbeats + events to HoneyWire Hub...")
    for sensor in SENSORS:
        post_heartbeat(sensor)
        time.sleep(0.1)

    for i in range(30):
        sensor = random.choice(SENSORS)
        event = random.choice(EVENT_TYPES)
        post_event(sensor, event)
        time.sleep(0.05)

    print("Done. Now refresh http://localhost:8080 to view mock UI.")


if __name__ == "__main__":
    main()
