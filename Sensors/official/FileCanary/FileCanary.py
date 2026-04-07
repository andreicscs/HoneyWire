"""
HoneyWire Official Sensor: File Canary
Sensor type: file_canary

This sensor monitors a mounted directory for tampering events such as modifications, deletions, or moves.
"""

import asyncio
import os
import time
from pathlib import Path

from honeywire import HoneyWireSensor
from watchdog.events import FileSystemEventHandler
from watchdog.observers import Observer


class HoneyFileHandler(FileSystemEventHandler):
    def __init__(self, reporter):
        super().__init__()
        self.reporter = reporter

    def _report(self, event, action: str, target: str):
        details = {
            "action": action,
            "timestamp_os": str(time.time()),
        }

        self._send_event(target, details)

    def _send_event(self, target: str, details: dict):
        self.reporter.report_event(
            event_type="file_tampered",
            severity="critical",
            details=details,
            action_taken="logged",
            source="Unknown (Local OS)",
            target=target,
        )

    def on_modified(self, event):
        if event.is_directory:
            return
        self._report(event, "File Modified/Encrypted", str(event.src_path))

    def on_deleted(self, event):
        if event.is_directory:
            return
        self._report(event, "File Deleted", str(event.src_path))

    def on_moved(self, event):
        if event.is_directory:
            return
        self._report(event, "File Moved/Renamed", f"{event.src_path} -> {event.dest_path}")


class FileCanary(HoneyWireSensor):
    def __init__(self):
        super().__init__(sensor_type="file_canary")
        self.honey_dir = os.getenv("HW_HONEY_DIR", "/honey_dir")
        self.polling_interval = int(os.getenv("HW_POLLING_INTERVAL", "1"))

    async def monitor(self):
        watch_path = Path(self.honey_dir)
        if not watch_path.exists() or not watch_path.is_dir():
            print(f"[!] FATAL: Watch directory does not exist: {watch_path}")
            raise SystemExit(1)

        print(f"[*] HoneyWire File Canary | Watching {watch_path} | Severity: {self.severity}")

        event_handler = HoneyFileHandler(self)
        observer = Observer()
        observer.schedule(event_handler, str(watch_path), recursive=True)
        observer.start()

        try:
            while True:
                await asyncio.sleep(self.polling_interval)
        finally:
            observer.stop()
            observer.join()


if __name__ == "__main__":
    sensor = FileCanary()
    try:
        asyncio.run(sensor.start())
    except KeyboardInterrupt:
        print("\n[*] Shutting down File Canary sensor...")
