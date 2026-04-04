"""
HoneyWire Official Sensor: Ping Canary
Sensor type: network

This Dark Node sensor listens for ICMP Echo Requests directed at the container's own IP address.
It is designed to be deployed on an isolated IP where absolutely zero legitimate traffic should occur.
"""

import asyncio
import os
from typing import Dict

from honeywire import HoneyWireSensor
from scapy.all import ICMP, IP, conf, sniff


class PingCanary(HoneyWireSensor):
    def __init__(self):
        super().__init__(sensor_type="network")
        self.severity = os.getenv("HW_SEVERITY", "high")
        self.iface = os.getenv("HW_PING_CANARY_IFACE", "")

        if self.iface:
            conf.iface = self.iface
            print(f"[*] Ping Canary using interface: {self.iface}")
        else:
            print("[*] Ping Canary using default Scapy interface")

        conf.use_pcap = False

    def _handle_ping(self, packet) -> None:
        if not packet.haslayer(ICMP) or not packet.haslayer(IP):
            return

        icmp = packet[ICMP]
        if icmp.type != 8:
            return

        source_ip = packet[IP].src
        packet_size = len(packet)
        ttl = packet[IP].ttl

        metadata: Dict[str, object] = {
            "packet_size": packet_size,
            "ttl": ttl,
        }

        print(f"[+] ICMP Echo Request detected from {source_ip} (size={packet_size}, ttl={ttl})")
        self._report_ping(source_ip, metadata)

    def _report_ping(self, source_ip: str, metadata: Dict) -> None:
        self.report_event(
            event_type="icmp_ping_received",
            severity="high",
            metadata=metadata,
            action_taken="logged",
            source=source_ip,
            target="ICMP Listener",
        )

    async def monitor(self):
        print(f"[*] HoneyWire Ping Canary | Severity: {self.severity}")
        print("[*] Listening for ICMP Echo Requests on the configured interface...")

        await asyncio.to_thread(
            sniff,
            store=False,
            prn=self._handle_ping,
            filter="icmp[icmptype] == icmp-echo",
        )


if __name__ == "__main__":
    sensor = PingCanary()
    try:
        asyncio.run(sensor.start())
    except KeyboardInterrupt:
        print("\n[*] Shutting down Ping Canary sensor...")
