import os
import sys
import time
import math
import random
import threading
import queue
import requests
from abc import ABC, abstractmethod
from enum import Enum
from dataclasses import dataclass

SDK_DEFAULT_AGENT_VERSION = "1.0.0"

# ============================================================================
# 1. SHARED CLASSIFIER (Pure Truth Layer)
# ============================================================================

@dataclass
class ResponseFact:
    is_error: bool
    is_transient: bool
    status_code: int
    retry_after: float

def classify(exception: Exception, response: requests.Response = None) -> ResponseFact:
    if exception is not None or response is None:
        return ResponseFact(is_error=True, is_transient=True, status_code=0, retry_after=0.0)

    fact = ResponseFact(
        status_code=response.status_code,
        is_error=response.status_code >= 400,
        is_transient=False,
        retry_after=0.0
    )

    if fact.is_error:
        if response.status_code in (400, 401, 403, 404):
            fact.is_transient = False
        else:
            fact.is_transient = True
            
            # Extract explicit wait instructions if the server provides them
            retry_after_str = response.headers.get("Retry-After")
            if retry_after_str and retry_after_str.isdigit():
                fact.retry_after = float(retry_after_str)

    return fact

# ============================================================================
# 2. POLICY INTERPRETERS (Domain-Specific Rules)
# ============================================================================

class EventAction(Enum):
    SUCCESS = "success"
    RETRY = "retry"
    DROP = "drop"

MAX_RETRIES_PER_EVENT = 7
BASE_HEARTBEAT_INTERVAL = 30.0
TERMINAL_SLEEP_INTERVAL = 300.0  # 5 minutes

class PolicyEngine:
    @staticmethod
    def event_policy(fact: ResponseFact, attempt: int) -> tuple[EventAction, float]:
        if not fact.is_error:
            return EventAction.SUCCESS, 0.0
        if not fact.is_transient:
            return EventAction.DROP, 0.0

        delay = fact.retry_after
        if delay <= 0.0:
            delay = PolicyEngine.calculate_backoff(attempt)
        return EventAction.RETRY, delay

    @staticmethod
    def heartbeat_policy(fact: ResponseFact) -> float:
        if not fact.is_error:
            return BASE_HEARTBEAT_INTERVAL
        if not fact.is_transient:
            return TERMINAL_SLEEP_INTERVAL
        
        if fact.retry_after > BASE_HEARTBEAT_INTERVAL:
            return fact.retry_after
        return BASE_HEARTBEAT_INTERVAL

    @staticmethod
    def calculate_backoff(attempt: int) -> float:
        base = 2.0
        max_delay = 60.0
        delay = base * math.pow(2, attempt)
        if delay > max_delay:
            delay = max_delay
            
        jitter = (random.random() * 0.2) - 0.1
        return delay + (delay * jitter)


# ============================================================================
# SENSOR STRUCT & INIT
# ============================================================================

class HoneyWireSensor(ABC):
    def __init__(self, sensor_type: str):
        self.sensor_type = sensor_type

        self.hub_endpoint = os.getenv("HW_HUB_ENDPOINT")
        self.hub_key = os.getenv("HW_HUB_KEY")
        self.sensor_id = os.getenv("HW_SENSOR_ID")
        self.config_rev = os.getenv("HW_CONFIG_REV", "")
        self.test_mode = os.getenv("HW_TEST_MODE", "false").lower() == "true"
        self.agent_version = os.getenv("HONEYWIRE_VERSION", SDK_DEFAULT_AGENT_VERSION)
        
        self.severity = os.getenv("HW_SEVERITY", "medium") 

        self._validate_required_env()
        
        # Connection pooling for high-performance background workers
        self.client = requests.Session()
        self.client.headers.update({
            "Authorization": f"Bearer {self.hub_key}",
            "Content-Type": "application/json"
        })

        self._hub_contract_version = "unknown"
        self._stop_event = threading.Event()
        self._event_queue = queue.Queue(maxsize=1000)

    def _validate_required_env(self):
        if not all([self.hub_endpoint, self.hub_key, self.sensor_id]):
            raise ValueError("Missing required environment variables (HW_HUB_ENDPOINT, HW_HUB_KEY, HW_SENSOR_ID).")

    def _normalize_severity(self, raw_severity) -> str:
        mapping = {"1": "info", "2": "low", "3": "medium", "4": "high", "5": "critical"}
        val = str(raw_severity).lower().strip()
        if val in mapping: return mapping[val]
        if val in ["info", "low", "medium", "high", "critical"]: return val
        print(f"[!] Warning: Invalid severity '{raw_severity}'. Defaulting to 'info'.")
        return "info"

    async def start(self):
        self._sync_hub_version()
        
        threading.Thread(target=self._event_loop, daemon=True).start()
        threading.Thread(target=self._heartbeat_loop, daemon=True).start()
        
        await self.monitor()
        
    def stop(self):
        self._stop_event.set()
        self.go_offline("graceful_shutdown")

    def run_test_mode(self) -> bool:
        print("[*] Test mode: sending synthetic payload...")
        return self.report_event(
            event_trigger="test_mode_synthetic_alert",
            severity="info",
            source="CI/CD Runner",
            target="Mock Hub",
            details={"test_message": "Automated CI/CD check."}
        )

    # ==========================================
    # PIPELINE A: EVENT WORKER
    # ==========================================

    def report_event(self, event_trigger: str, source: str = "Unknown", target: str = "Unknown", details: dict = None) -> bool:
        if details is None: details = {}
        
        # Uses the environment variable loaded in __init__
        normalized_severity = self._normalize_severity(self.severity)
        
        payload = {
            "contract_version": self._hub_contract_version,
            "severity": normalized_severity,
            "event_trigger": event_trigger,
            "source": source,
            "target": target,
            "sensor_id": self.sensor_id,
            "details": details
        }

        try:
            self._event_queue.put_nowait(payload)
            return True
        except queue.Full:
            print("[-] Event buffer full. Dropping event.")
            return False

    def _event_loop(self):
        while not self._stop_event.is_set():
            try:
                # 1s timeout allows loop to check _stop_event cleanly
                event = self._event_queue.get(timeout=1.0)
            except queue.Empty:
                continue

            self._process_event(event)
            self._event_queue.task_done()
            
        self._drain_queue()

    def _process_event(self, event: dict):
        for attempt in range(MAX_RETRIES_PER_EVENT):
            resp, exc = None, None
            try:
                resp = self.client.post(f"{self.hub_endpoint}/api/v1/event", json=event, timeout=10)
            except Exception as e:
                exc = e

            fact = classify(exc, resp)
            action, delay = PolicyEngine.event_policy(fact, attempt)

            if action == EventAction.SUCCESS:
                print("[+] Event reported successfully.")
                return
            elif action == EventAction.DROP:
                print(f"[-] Terminal failure (HTTP {fact.status_code}). Dropping poison event.")
                return
            elif action == EventAction.RETRY:
                print(f"[!] Transient issue. Retrying event ({attempt+1}/{MAX_RETRIES_PER_EVENT}) in {delay:.1f}s...")
                
                # wait() blocks for 'delay' seconds, but instantly returns True if shutdown is signaled
                if self._stop_event.wait(delay):
                    return

        print(f"[-] Event exceeded MaxRetriesPerEvent ({MAX_RETRIES_PER_EVENT}). Dropped.")

    def _drain_queue(self):
        print("[*] Draining remaining event queue before shutdown...")
        while not self._event_queue.empty():
            try:
                event = self._event_queue.get_nowait()
                self.client.post(f"{self.hub_endpoint}/api/v1/event", json=event, timeout=2)
            except Exception:
                pass
        print("[*] Event queue gracefully drained.")

    # ==========================================
    # PIPELINE B: HEARTBEAT WORKER
    # ==========================================

    def _heartbeat_loop(self) -> None:
        sleep_duration = 0.0

        while True:
            if sleep_duration > 0:
                if self._stop_event.wait(sleep_duration):
                    return

            payload = {
                "sensor_id": self.sensor_id,
                "metadata": {
                    "agent_version": self.agent_version,
                    "contract_version": self._hub_contract_version,
                    "sensor_type": self.sensor_type,
                    "HW_CONFIG_REV": self.config_rev
                }
            }

            resp, exc = None, None
            try:
                resp = self.client.post(f"{self.hub_endpoint}/api/v1/heartbeat", json=payload, timeout=10)
            except Exception as e:
                exc = e

            fact = classify(exc, resp)
            sleep_duration = PolicyEngine.heartbeat_policy(fact)

            if fact.is_error:
                print(f"[!] Heartbeat degraded. Next pulse in {sleep_duration}s")

    # ==========================================
    # UTILITIES
    # ==========================================

    def _sync_hub_version(self) -> None:
        for i in range(3):
            resp, exc = None, None
            try:
                resp = self.client.get(f"{self.hub_endpoint}/api/v1/version", timeout=5)
            except Exception as e:
                exc = e

            fact = classify(exc, resp)

            if not fact.is_error:
                try:
                    data = resp.json()
                    self._hub_contract_version = data.get("version", "unknown")
                    
                    hub_major = str(self._hub_contract_version).split('.')[0]
                    agent_major = str(self.agent_version).split('.')[0]
                    if hub_major != agent_major and hub_major != "unknown":
                        raise RuntimeError(f"Version mismatch. Hub (v{self._hub_contract_version}) vs Agent (v{self.agent_version})")
                    
                    return
                except Exception as e:
                    print(f"[!] Sync decode failed: {e}")
                    fact.is_transient = True 

            if not fact.is_transient:
                raise ConnectionError(f"Fatal synchronization failure (HTTP {fact.status_code})")

            wait = PolicyEngine.calculate_backoff(i)
            time.sleep(wait) # Safe blocking sleep during startup

        raise ConnectionError("Failed to synchronize with Hub after backoff limits.")

    def go_offline(self, reason: str):
        payload = {
            "sensor_id": self.sensor_id,
            "reason": reason
        }
        try:
            self.client.post(f"{self.hub_endpoint}/api/v1/offline", json=payload, timeout=2)
        except Exception:
            pass

    @abstractmethod
    async def monitor(self):
        pass