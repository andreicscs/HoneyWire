"""
HoneyWire Official Sensor: Network Scan Detector
Sensor type: network

This sensor silently watches raw SYN packets and detects horizontal port scans against closed/unused ports.
"""

import asyncio
import os
import time
from collections import defaultdict
from typing import Dict, List

from honeywire import HoneyWireSensor
from scapy.all import IP, TCP, conf, sniff

DEFAULT_THRESHOLD = int(os.getenv("HW_SCAN_THRESHOLD", "5"))
DEFAULT_WINDOW = int(os.getenv("HW_SCAN_WINDOW", "5"))
DEFAULT_IGNORE_PORTS = os.getenv("HW_IGNORE_PORTS", "80,443")
DEFAULT_COOLDOWN = 60.0  # Seconds to wait before alerting on the same IP again

class NetworkScanDetector(HoneyWireSensor):
    def __init__(self):
        super().__init__(sensor_type="network")

        self.threshold = int(os.getenv("HW_SCAN_THRESHOLD", DEFAULT_THRESHOLD))
        self.window = int(os.getenv("HW_SCAN_WINDOW", DEFAULT_WINDOW))
        self.ignore_ports = self._parse_ports(os.getenv("HW_IGNORE_PORTS", DEFAULT_IGNORE_PORTS))
        self.cooldown = DEFAULT_COOLDOWN

        self.scan_history: Dict[str, List[tuple[float, int]]] = defaultdict(list)
        
        # Changed from set() to dict to track timestamps for the cooldown
        self.alerted_sources: Dict[str, float] = {}

        conf.use_pcap = False

    @staticmethod
    def _parse_ports(raw: str):
        ports = set()
        for item in raw.split(","):
            item = item.strip()
            if not item:
                continue
            try:
                ports.add(int(item))
            except ValueError:
                print(f"[!] Invalid port listed in HW_IGNORE_PORTS: {item}")
        return ports

    def _handle_syn(self, packet) -> None:
        if not packet.haslayer(TCP) or not packet.haslayer(IP):
            return

        tcp = packet[TCP]
        
        # STRICT FILTER: 0x02 is the exact hex value for a pure SYN packet.
        # This drops ACK (0x10), SYN-ACK (0x12), etc., eliminating background noise.
        if tcp.flags != 0x02:
            return

        dst_port = tcp.dport
        source_ip = packet[IP].src

        # Optional: Ignore traffic from your Windows host gateway to reduce WSL noise
        # if source_ip == "192.168.218.1": 
        #     return

        if dst_port in self.ignore_ports:
            return

        now = time.time()
        
        # Clean up old history outside the time window
        self.scan_history[source_ip] = [entry for entry in self.scan_history[source_ip] if now - entry[0] <= self.window]
        self.scan_history[source_ip].append((now, dst_port))

        unique_ports = sorted({entry[1] for entry in self.scan_history[source_ip]})
        
        if len(unique_ports) >= self.threshold:
            last_alert_time = self.alerted_sources.get(source_ip, 0.0)
            
            # Only fire the alert if this IP is out of the cooldown penalty box
            if now - last_alert_time > self.cooldown:
                self.alerted_sources[source_ip] = now
                print(f"[!] Port scan detected from {source_ip}: {unique_ports}")
                
                # Synchronous call to the HoneyWire SDK
                self.report_event(
                    event_type="network_scan_detected",
                    severity="high",
                    details={
                        "ports_hit": unique_ports, 
                        "count": len(unique_ports),
                        "window_sec": self.window
                    },
                    source=source_ip,
                    target="Multiple Ports"
                )
            
            # Always clear the queue after crossing the threshold to save memory
            self.scan_history[source_ip] = []

    async def monitor(self):
        print(f"[*] HoneyWire Network Scan Detector | Threshold: {self.threshold} ports | Severity: {self.severity}")
        
        # We removed iface="any". Scapy will default to the primary interface.
        await asyncio.to_thread(
            sniff,
            store=False,
            prn=self._handle_syn,
            filter="tcp"
        )


if __name__ == "__main__":
    sensor = NetworkScanDetector()
    try:
        asyncio.run(sensor.start())
    except KeyboardInterrupt:
        print("\n[*] Shutting down Network Scan Detector...")