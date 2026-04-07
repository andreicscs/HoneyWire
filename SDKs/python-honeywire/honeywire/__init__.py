import os
import sys
import time
import threading
import datetime
import requests
from abc import ABC, abstractmethod

SDK_DEFAULT_AGENT_VERSION = "1.0.0"
HONEYWIRE_SCHEMA_VERSION = "1.0"
HEARTBEAT_INTERVAL_SECONDS = 30

class HoneyWireSensor(ABC):
    def __init__(self, sensor_type: str):
        self.sensor_type = sensor_type

        self.hub_endpoint = os.getenv("HW_HUB_ENDPOINT")
        self.hub_key = os.getenv("HW_HUB_KEY")
        self.sensor_id = os.getenv("HW_SENSOR_ID")
        self.test_mode = os.getenv("HW_TEST_MODE", "false").lower() == "true"
        self.agent_version = os.getenv("HONEYWIRE_VERSION", SDK_DEFAULT_AGENT_VERSION)
        self.severity = os.getenv("HW_SEVERITY", "4")

        self._validate_required_env()
        self.headers = self._build_headers()

        self.hub_contract_version = "unknown"

    def _validate_required_env(self):
        if not all([self.hub_endpoint, self.hub_key, self.sensor_id]):
            print("[!] FATAL: Missing required environment variables (HW_HUB_ENDPOINT, HW_HUB_KEY, HW_SENSOR_ID).")
            sys.exit(1)

    def _build_headers(self) -> dict:
        return {
            "Authorization": f"Bearer {self.hub_key}",
            "Content-Type": "application/json"
        }

    def _normalize_severity(self, raw_severity) -> str:
        """Converts 1-5 or strings into the official schema enum."""
        mapping = {
            "1": "info", 
            "2": "low", 
            "3": "medium", 
            "4": "high", 
            "5": "critical"
        }
        
        val = str(raw_severity).lower().strip()
        
        if val in mapping:
            return mapping[val]
        elif val in ["info", "low", "medium", "high", "critical"]:
            return val
        else:
            print(f"[!] Warning: Invalid severity '{raw_severity}'. Defaulting to 'info'.")
            return "info"

    def _sync_hub_version(self) -> None:
        """Fetches the Hub's contract version synchronously on startup."""
        print(f"[*] Synchronizing with Hub at {self.hub_endpoint}...")
        try:
            resp = requests.get(f"{self.hub_endpoint}/api/v1/version", headers=self.headers, timeout=5)
            resp.raise_for_status()
            
            self.hub_contract_version = resp.json().get("version", "unknown")
            
            # Semantic Versioning Check
            hub_major = str(self.hub_contract_version).split('.')[0]
            agent_major = str(self.agent_version).split('.')[0]
            
            if hub_major != agent_major and hub_major != "unknown":
                print(f"[!] FATAL: Version mismatch. Hub (v{self.hub_contract_version}) vs Agent (v{self.agent_version})")
                sys.exit(1)
                
            print(f"[+] Synchronized successfully. Operating on contract v{self.hub_contract_version}")
        except requests.exceptions.RequestException as e:
            print(f"[!] FATAL: Failed to synchronize with Hub. Details: {e}")
            sys.exit(1)

    def _post_to_hub(self, path: str, payload: dict, timeout: int = 5):
        url = f"{self.hub_endpoint}{path}"
        return requests.post(url, headers=self.headers, json=payload, timeout=timeout)

    def _heartbeat_loop(self) -> None:
        """Background thread to ping the Hub every 30 seconds."""
        payload = {
            "sensor_id": self.sensor_id,
            "sensor_type": self.sensor_type,
            "metadata": {
                "agent_version": self.agent_version,
                "contract_version": self.hub_contract_version,
            }
        }
        while True:
            try:
                resp = self._post_to_hub("/api/v1/heartbeat", payload)
                resp.raise_for_status()
            except Exception as e:
                print(f"[-] Heartbeat error: {e}")
            time.sleep(HEARTBEAT_INTERVAL_SECONDS)

    def report_event(
        self,
        event_type: str,
        severity,
        details: dict,
        action_taken: str = "logged",
        source: str = "Unknown",
        target: str = "Unknown",
    ) -> bool:
        """Formats and sends the payload enforcing the HoneyWire JSON Schema."""
        normalized_severity = self._normalize_severity(severity)
        
        payload = {
            "contract_version": "1.0",
            "sensor_id": self.sensor_id,
            "sensor_type": self.sensor_type,
            "event_type": event_type,
            "severity": normalized_severity,
            "timestamp": datetime.datetime.now(datetime.timezone.utc).isoformat(),
            "action_taken": action_taken,
            "source": source,
            "target": target,
            "details": details
        }

        try:
            resp = self._post_to_hub("/api/v1/event", payload)
            resp.raise_for_status()
            print(f"[+] Event sent: {event_type} (Severity: {normalized_severity})")
            return True
        except requests.exceptions.RequestException as e:
            print(f"[-] Event report failed: {e}")
            return False

    def _run_test_mode(self):
        """Used by CI/CD to verify sensor works and exits cleanly."""
        print("🛠️ TEST MODE ACTIVE: Sending synthetic payload...")
        success = self.report_event(
            event_type="test_mode_synthetic_alert",
            severity="info",
            details={"test_message": "Automated CI/CD check."},
            action_taken="ignored"
        )
        if success:
            print("✅ Test mode complete. Exiting gracefully.")
            sys.exit(0)
        else:
            print("❌ Test mode failed to contact Hub.")
            sys.exit(1)

    @abstractmethod
    async def monitor(self):
        """The specific sensor logic to be implemented by the creator."""
        pass

    async def start(self):
        """Initializes the sensor, runs background threads, and starts the async monitor."""
        self._sync_hub_version()
        
        if self.test_mode:
            self._run_test_mode()
            
        # Start heartbeat in a standard daemon thread (avoids blocking the async loop)
        threading.Thread(target=self._heartbeat_loop, daemon=True).start()
        
        # Await the creator's async logic
        await self.monitor()