import os
import sys
import time
import threading
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
        
        # Updated default to medium to prevent misconfigurations from flooding critical alerts
        self.severity = os.getenv("HW_SEVERITY", "medium") 

        self._validate_required_env()
        self.headers = self._build_headers()

        self._hub_contract_version = "unknown"
        self._stop_event = threading.Event() # For graceful heartbeat shutdown

    def _validate_required_env(self):
        if not all([self.hub_endpoint, self.hub_key, self.sensor_id]):
            raise ValueError("Missing required environment variables (HW_HUB_ENDPOINT, HW_HUB_KEY, HW_SENSOR_ID).")

    def _build_headers(self) -> dict:
        return {
            "Authorization": f"Bearer {self.hub_key}",
            "Content-Type": "application/json"
        }

    def _normalize_severity(self, raw_severity) -> str:
        mapping = {
            "1": "info", "2": "low", "3": "medium", "4": "high", "5": "critical"
        }
        val = str(raw_severity).lower().strip()
        if val in mapping: return mapping[val]
        if val in ["info", "low", "medium", "high", "critical"]: return val
        print(f"[!] Warning: Invalid severity '{raw_severity}'. Defaulting to 'info'.")
        return "info"

    def _sync_hub_version(self) -> None:
        """Fetches the Hub's contract version synchronously on startup with backoff."""
        print(f"[*] Synchronizing with Hub at {self.hub_endpoint}...")
        backoff = [2, 5, 15]

        for wait in backoff:
            try:
                resp = requests.get(f"{self.hub_endpoint}/api/v1/version", headers=self.headers, timeout=5)
                resp.raise_for_status()
                
                self._hub_contract_version = resp.json().get("version", "unknown")
                
                hub_major = str(self._hub_contract_version).split('.')[0]
                agent_major = str(self.agent_version).split('.')[0]
                
                if hub_major != agent_major and hub_major != "unknown":
                    raise RuntimeError(f"Version mismatch. Hub (v{self._hub_contract_version}) vs Agent (v{self.agent_version})")
                    
                print(f"[+] Synchronized successfully. Operating on contract v{self._hub_contract_version}")
                return
            except requests.exceptions.RequestException as e:
                print(f"[!] Sync attempt failed: {e}. Retrying in {wait}s...")
                time.sleep(wait)
                
        raise ConnectionError("Failed to synchronize with Hub after multiple attempts.")

    def _post_to_hub(self, path: str, payload: dict, timeout: int = 10):
        url = f"{self.hub_endpoint}{path}"
        return requests.post(url, headers=self.headers, json=payload, timeout=timeout)

    def _heartbeat_loop(self) -> None:
        """Background thread to ping the Hub."""
        # Fire an immediate heartbeat on startup
        self._send_heartbeat()
        
        # .wait() acts like sleep, but instantly returns True if the stop event is triggered
        while not self._stop_event.wait(HEARTBEAT_INTERVAL_SECONDS):
            self._send_heartbeat()

    def _send_heartbeat(self) -> None:
        """Constructs and sends the payload to prevent stale metadata."""
        payload = {
            "sensor_id": self.sensor_id,
            "details": {
                "agent_version": self.agent_version,
                "contract_version": self._hub_contract_version,
                "sensor_type": self.sensor_type
            }
        }
        try:
            resp = self._post_to_hub("/api/v1/heartbeat", payload)
            resp.raise_for_status()
        except Exception as e:
            print(f"[-] Heartbeat error: {e}")

    def report_event(self, event_trigger: str, severity: str, source: str = "Unknown", target: str = "Unknown", details: dict = None) -> bool:
        if details is None: details = {}
        normalized_severity = self._normalize_severity(severity)
        
        payload = {
            "contract_version": self._hub_contract_version, # Send synced version
            "severity": normalized_severity,
            "event_trigger": event_trigger,
            "source": source,
            "target": target,
            "sensor_id": self.sensor_id,
            "details": details
        }

        try:
            resp = self._post_to_hub("/api/v1/event", payload)
            resp.raise_for_status()
            print(f"[+] Event sent: {event_trigger} (Severity: {normalized_severity})")
            return True
        except requests.exceptions.RequestException as e:
            print(f"[-] Event report failed: {e}")
            return False

    def run_test_mode(self) -> bool:
        """Used by CI/CD to verify sensor works and exits cleanly."""
        print("🛠️ TEST MODE ACTIVE: Sending synthetic payload...")
        success = self.report_event(
            event_trigger="test_mode_synthetic_alert",
            severity="info",
            source="CI/CD Runner",
            target="Mock Hub",
            details={
                "test_message": "Automated CI/CD check.",
                "action_taken": "ignored"
            }
        )
        return success

    @abstractmethod
    async def monitor(self):
        """The specific sensor logic to be implemented by the creator."""
        pass

    async def start(self):
        """Initializes the sensor, runs background threads, and starts the async monitor."""
        self._sync_hub_version()
        threading.Thread(target=self._heartbeat_loop, daemon=True).start()
        await self.monitor()
        
    def stop(self):
        """Gracefully shuts down background tasks."""
        self._stop_event.set()