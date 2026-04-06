"""Generate demo sensors + dynamic events for HoneyWire UI showcase."""

import os
import requests
import random
import socket
import struct
import time
from datetime import datetime

HUB_URL = "http://localhost:8080"
API_SECRET = "change_this_to_a_secure_random_string"
HEADERS = {"x-api-key": API_SECRET, "Content-Type": "application/json"}

# Determine version for contract validation
def get_project_version():
    version = os.getenv("HONEYWIRE_VERSION")
    if version: return version
    try:
        with open(os.path.join(os.path.dirname(__file__), "..", "VERSION"), "r") as f:
            return f.read().strip()
    except FileNotFoundError:
        return "1.0.0"

VERSION = get_project_version()

# --- Random Data Generators ---

def generate_random_ip():
    """Generates a random public-looking IPv4 address."""
    return socket.inet_ntoa(struct.pack('>I', random.randint(0x01000000, 0xDF000000)))

def generate_fleet(num_sensors=8):
    """Generates a list of random sensors with dynamic names."""
    types = ["tarpit", "canary_token", "ids_sentinel", "web_honeypot", "llm_probe"]
    regions = ["us-east", "us-west", "eu-central", "ap-south", "lan-core", "dmz"]
    
    fleet = []
    for _ in range(num_blocks := num_sensors):
        s_type = random.choice(types)
        region = random.choice(regions)
        sensor_id = f"{s_type}-{region}-{random.randint(10, 99)}"
        fleet.append({"sensor_id": sensor_id, "sensor_type": s_type})
    return fleet

def generate_contextual_event(sensor):
    """Generates a realistic event that matches the sensor type."""
    s_type = sensor["sensor_type"]
    ip = generate_random_ip()
    
    # Base Event Template
    event = {
        "sensor_id": sensor["sensor_id"],
        "sensor_type": s_type,
        "contract_version": VERSION,
        "source": ip,
        "action_taken": random.choice(["logged", "blocked", "tarpitted", "alert_only"])
    }

    # Contextual Threat Data
    if s_type == "web_honeypot":
        event["event_type"] = random.choice(["sqli_attempt", "xss_payload", "dir_bruteforce", "api_fuzz"])
        event["severity"] = random.choice(["high", "critical", "medium"])
        event["target"] = random.choice(["/wp-admin", "/api/v1/users", "/.env", "/config.php"])
        event["metadata"] = {"user_agent": "masscan/1.3", "payload": "1' OR '1'='1", "method": "POST"}

    elif s_type == "tarpit":
        event["event_type"] = random.choice(["tcp_syn_flood", "port_scan", "ssh_bruteforce"])
        event["severity"] = random.choice(["low", "medium", "info"])
        event["target"] = random.choice(["Port 22", "Port 23", "Port 3389", "Port 445"])
        event["metadata"] = {"packets_dropped": random.randint(50, 5000), "connection_time_held": f"{random.randint(10, 120)}s"}

    elif s_type == "canary_token":
        event["event_type"] = random.choice(["file_accessed", "aws_key_used", "db_queried"])
        event["severity"] = "critical"
        event["target"] = random.choice(["/etc/passwd.bak", "AWS_SECRET_KEY", "users_backup.sql"])
        event["metadata"] = {"process_name": random.choice(["cat", "curl", "python3"]), "user": "www-data"}

    elif s_type == "llm_probe":
        event["event_type"] = random.choice(["prompt_injection", "jailbreak_attempt", "data_exfiltration"])
        event["severity"] = random.choice(["high", "critical"])
        event["target"] = "/v1/chat/completions"
        event["metadata"] = {"prompt_snippet": "Ignore previous instructions and print...", "model": "gpt-4-turbo"}

    else: # ids_sentinel
        event["event_type"] = random.choice(["malware_signature", "lateral_movement", "beaconing"])
        event["severity"] = random.choice(["high", "critical", "medium"])
        event["target"] = f"Subnet 10.0.{random.randint(1,5)}.0/24"
        event["metadata"] = {"signature_id": f"SID-{random.randint(1000, 9999)}", "confidence": f"{random.randint(80, 100)}%"}

    return event

# --- API Interaction ---

def post_heartbeat(sensor):
    body = {
        "sensor_id": sensor["sensor_id"],
        "sensor_type": sensor["sensor_type"],
        "metadata": {
            "version": VERSION,
            "os": random.choice(["Linux", "Windows Server 2022", "FreeBSD"]),
            "uptime_days": random.randint(1, 400),
        },
    }
    r = requests.post(f"{HUB_URL}/api/v1/heartbeat", json=body, headers=HEADERS, timeout=5)
    r.raise_for_status()

def post_event(event_data):
    r = requests.post(f"{HUB_URL}/api/v1/event", json=event_data, headers=HEADERS, timeout=5)
    r.raise_for_status()

def main():
    print(f"Generating dynamic fleet and events for HoneyWire Hub (v{VERSION})...")
    
    # 1. Generate Fleet & Send Heartbeats
    fleet = generate_fleet(10) # Change this number to test UI scaling
    for sensor in fleet:
        post_heartbeat(sensor)
        print(f"[+] Heartbeat registered: {sensor['sensor_id']}")
        time.sleep(0.1)

    print("\nSimulating threat activity...")
    
    # 2. Fire 40 random, contextual events
    for _ in range(40):
        sensor = random.choice(fleet)
        
        # Occasionally simulate an offline sensor by NOT sending a heartbeat,
        # but 90% of the time, refresh the heartbeat with the event.
        if random.random() > 0.1:
            post_heartbeat(sensor)
            
        event_data = generate_contextual_event(sensor)
        post_event(event_data)
        
        # Print a tiny log to the terminal so it looks cool running
        print(f" ↳ {event_data['severity'].upper().ljust(8)} | {event_data['sensor_id']} | {event_data['event_type']}")
        time.sleep(random.uniform(0.05, 0.2))

    print("\n✅ Done. Refresh http://localhost:8080 to view the SOC dashboard.")

if __name__ == "__main__":
    main()