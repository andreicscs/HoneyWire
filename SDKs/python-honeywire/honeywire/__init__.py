import os
import sys
import time
import math
import random
import threading
import queue
import asyncio
import signal
import requests
from abc import ABC, abstractmethod
from enum import Enum
from dataclasses import dataclass



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
TERMINAL_SLEEP_INTERVAL = 3600.0  # 1 hour

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
    def __init__(self):
        self.hub_endpoint = os.getenv("HW_HUB_ENDPOINT")
        self.hub_key = os.getenv("HW_HUB_KEY")
        self.sensor_id = os.getenv("HW_SENSOR_ID")
        self.config_rev = os.getenv("HW_CONFIG_REV", "")
        self.test_mode = os.getenv("HW_TEST_MODE", "false").lower() == "true"
        
        self.severity = os.getenv("HW_SEVERITY", "medium") 

        self._validate_required_env()
        
        # Connection pooling for high-performance background workers
        self.client = requests.Session()
        self.client.headers.update({
            "Authorization": f"Bearer {self.hub_key}",
            "Content-Type": "application/json"
        })

        self._stop_event = threading.Event()
        self._event_queue = queue.Queue(maxsize=1000)
        
        self.test_trigger = ""
        self.test_source = ""
        self.test_target = ""
        self.test_details = {}

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

    def set_test_payload(self, trigger: str, source: str, target: str, details: dict):
        """Set a custom payload for test mode and SIGUSR1 live testing."""
        self.test_trigger = trigger
        self.test_source = source
        self.test_target = target
        self.test_details = details

    async def start(self):
        threading.Thread(target=self._event_loop, daemon=True).start()
        threading.Thread(target=self._heartbeat_loop, daemon=True).start()
        
        try:
            loop = asyncio.get_running_loop()
            loop.add_signal_handler(signal.SIGUSR1, self._handle_sigusr1)
        except (NotImplementedError, AttributeError):
            pass

        await self.monitor()
        
    def _handle_sigusr1(self):
        print("[*] SIGUSR1 received: injecting test event into queue...")
        trigger = "test_mode_synthetic_alert"
        source = "Wizard Live Test"
        target = "Mock Hub"
        details = {"test_message": "Wizard triggered a live test event firedrill."}

        if self.test_trigger:
            trigger = self.test_trigger
            source = self.test_source
            target = self.test_target
            details = self.test_details

        self.report_event(trigger, source, target, details)

    def stop(self):
        self._stop_event.set()
        self.go_offline("graceful_shutdown")

    def run_test_mode(self) -> bool:
        print("[*] Test mode: sending synthetic payload...")

        # 2. Synchronously send the payload to guarantee delivery before the program exits
        normalized_severity = self._normalize_severity(self.severity)
        
        trigger = "test_mode_synthetic_alert"
        source = "CI/CD Runner"
        target = "Mock Hub"
        details = {"test_message": "Automated CI/CD check."}

        if self.test_trigger:
            trigger = self.test_trigger
            source = self.test_source
            target = self.test_target
            details = self.test_details

        payload = {
            "sensorId": self.sensor_id,
            "severity": normalized_severity,
            "eventTrigger": trigger,
            "source": source,
            "target": target,
            "details": details
        }

        try:
            resp = self.client.post(f"{self.hub_endpoint}/api/v1/event", json=payload, timeout=10)
            if resp.status_code >= 400:
                print(f"[-] Test mode failed to send event: HTTP {resp.status_code}")
                return False
            return True
        except Exception as e:
            print(f"[-] Test mode failed to send event: {e}")
            return False

    # ==========================================
    # PIPELINE A: EVENT WORKER
    # ==========================================

    def report_event(self, event_trigger: str, source: str = "Unknown", target: str = "Unknown", details: dict = None) -> bool:
        if details is None: details = {}
        
        # Uses the environment variable loaded in __init__
        normalized_severity = self._normalize_severity(self.severity)
        
        payload = {
            "sensorId": self.sensor_id,
            "severity": normalized_severity,
            "eventTrigger": event_trigger,
            "source": source,
            "target": target,
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
                "sensorId": self.sensor_id,
                "configRev": self.config_rev
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

    def go_offline(self, reason: str):
        payload = {
            "sensorId": self.sensor_id,
            "reason": reason
        }
        try:
            self.client.post(f"{self.hub_endpoint}/api/v1/offline", json=payload, timeout=2)
        except Exception:
            pass

    @abstractmethod
    async def monitor(self):
        pass