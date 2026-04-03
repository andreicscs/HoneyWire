"""
HoneyWire Tarpit Agent (Standard Port Tripwire)
Sensor type: tarpit

Custom Environment variables:
    HW_DECOY_PORTS   Comma-separated ports (default: 2222,3306)
    HW_TARPIT_MODE   hold | echo | close (default: hold)
    HW_TARPIT_BANNER Optional service banner
"""

import asyncio
import os
import time
from typing import List, Optional

from honeywire import HoneyWireSensor


DEFAULT_DECOY_PORTS = [2222, 3306]
DEFAULT_TARPIT_MODE = "hold"
DEFAULT_MAX_BYTES = 1024 * 50
DEFAULT_MAX_LINES = 10
DEFAULT_MAX_DURATION = 3600
DEFAULT_CONCURRENCY = 1000


class TcpTarpitSensor(HoneyWireSensor):
    def __init__(self):
        super().__init__(sensor_type="tarpit")

        self.tarpit_mode = os.getenv("HW_TARPIT_MODE", DEFAULT_TARPIT_MODE).lower()
        self.decoy_ports = self._parse_ports(os.getenv("HW_DECOY_PORTS", ",".join(map(str, DEFAULT_DECOY_PORTS))))
        self.tarpit_banner = self._load_banner(os.getenv("HW_TARPIT_BANNER", ""))

        self.max_bytes = DEFAULT_MAX_BYTES
        self.max_lines = DEFAULT_MAX_LINES
        self.max_duration = DEFAULT_MAX_DURATION
        self.semaphore = asyncio.Semaphore(DEFAULT_CONCURRENCY)

    @staticmethod
    def _parse_ports(raw_ports: str) -> List[int]:
        ports = []
        for item in raw_ports.split(","):
            item = item.strip()
            if not item:
                continue
            try:
                ports.append(int(item))
            except ValueError:
                print(f"[!] Invalid port in HW_DECOY_PORTS: {item}")
        return ports or DEFAULT_DECOY_PORTS

    @staticmethod
    def _load_banner(raw_banner: str) -> bytes:
        if not raw_banner:
            return b""
        return raw_banner.encode("utf-8").decode("unicode_escape").encode("utf-8")

    async def _capture_connection(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter, port: int):
        peer = writer.get_extra_info("peername")
        source_ip = peer[0] if peer else "Unknown"

        start_time = time.time()
        captured_payload = []
        consumed_bytes = 0

        try:
            if self.tarpit_banner and self.tarpit_mode != "close":
                writer.write(self.tarpit_banner)
                await writer.drain()

            if self.tarpit_mode != "close":
                while consumed_bytes < self.max_bytes and (time.time() - start_time) < self.max_duration:
                    try:
                        timeout = 300.0
                        data = await asyncio.wait_for(reader.read(1024), timeout=timeout)
                        if not data:
                            break

                        decoded = data.decode("utf-8", errors="replace").strip()
                        if decoded and len(captured_payload) < self.max_lines:
                            captured_payload.append(decoded)

                        consumed_bytes += len(data)

                        if self.tarpit_mode == "echo":
                            writer.write(data)
                            await writer.drain()

                        await asyncio.sleep(0.5)

                    except asyncio.TimeoutError:
                        if not writer.is_closing():
                            writer.write(b"\0")
                            await writer.drain()

        except Exception as exc:
            print(f"[!] Connection processing error from {source_ip}:{port} - {exc}")

        finally:
            if not writer.is_closing():
                try:
                    writer.close()
                    await writer.wait_closed()
                except Exception:
                    pass

            duration = time.time() - start_time
            event_payload = {
                "source_ip": source_ip,
                "target_port": port,
                "duration_sec": round(duration, 2),
                "payload": captured_payload,
            }

            await asyncio.to_thread(
                self.report_event,
                event_type="tcp_connection",
                severity=self.severity,
                metadata=event_payload,
                action_taken=self.tarpit_mode,
                source=source_ip,
                target=f"Port {port}",
            )

    async def _serve_connection(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter, port: int):
        async with self.semaphore:
            await self._capture_connection(reader, writer, port)

    async def monitor(self):
        print(f"[*] HoneyWire Agent | Mode: {self.tarpit_mode.upper()} | Severity: {self.severity}")

        listeners = []
        for port in self.decoy_ports:
            listener = await asyncio.start_server(
                lambda r, w, p=port: self._serve_connection(r, w, p),
                "0.0.0.0",
                port,
            )
            listeners.append(listener.serve_forever())
            print(f"[+] Tarpit listening on port {port}")

        await asyncio.gather(*listeners)


if __name__ == "__main__":
    sensor = TcpTarpitSensor()
    try:
        asyncio.run(sensor.start())
    except KeyboardInterrupt:
        print("\n[*] Shutting down HoneyWire Tarpit...")