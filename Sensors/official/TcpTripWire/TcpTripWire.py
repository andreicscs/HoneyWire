"""
HoneyWire Tarpit Agent (Standard Port Tripwire)
Sensor type: tarpit

Custom Environment variables:
    DECOY_PORTS     Comma-separated ports     (default: 2222,3306)
    TARPIT_MODE     hold | echo | close       (default: hold)
"""
import asyncio
import os
import sys
import time

# Import the SDK
from honeywire import HoneyWireSensor


class TcpTarpitSensor(HoneyWireSensor):
    def __init__(self):
        # Initialize the base class with the sensor type
        super().__init__(sensor_type="tarpit")
        
        # Sensor-specific configuration
        self.tarpit_mode = os.getenv("HW_TARPIT_MODE", "hold").lower()
        
        raw_ports = os.getenv("HW_DECOY_PORTS", "2222,3306")
        self.decoy_ports = [int(p.strip()) for p in raw_ports.split(",") if p.strip()]
        
        raw_banner = os.getenv("HW_TARPIT_BANNER", "")
        self.tarpit_banner = raw_banner.encode("utf-8").decode("unicode_escape").encode("utf-8") if raw_banner else b""

        self.max_bytes = 1024 * 50
        self.max_lines = 10
        self.max_duration = 3600
        self.semaphore = asyncio.Semaphore(1000)

    async def handle_client(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter, port: int) -> None:
        addr = writer.get_extra_info("peername")
        source_ip = addr[0] if addr else "Unknown"
        
        start_time = time.time()
        payload_data = []
        total_bytes = 0

        try:
            if self.tarpit_banner and self.tarpit_mode != "close":
                writer.write(self.tarpit_banner)
                await writer.drain()

            if self.tarpit_mode != "close":
                while total_bytes < self.max_bytes and (time.time() - start_time) < self.max_duration:
                    try:
                        timeout = 300.0
                        data = await asyncio.wait_for(reader.read(1024), timeout=timeout)
                        
                        if not data: break
                        
                        decoded = data.decode("utf-8", errors="replace").strip()
                        if decoded and len(payload_data) < self.max_lines:
                            payload_data.append(decoded)
                                
                        total_bytes += len(data)

                        if self.tarpit_mode == "echo":
                            writer.write(data)
                            await writer.drain()
                        
                        await asyncio.sleep(0.5)
                        
                    except asyncio.TimeoutError:
                        if not writer.is_closing():
                            writer.write(b"\0")
                            await writer.drain()
                        continue
                        
        except Exception:
            pass 
        finally:
            try:
                writer.close()
                await writer.wait_closed()
            except Exception:
                pass 
            
            duration = time.time() - start_time
            
            metadata = {
                "source_ip": source_ip,
                "target_port": port,
                "duration_sec": round(duration, 2),
                "payload": payload_data
            }
            
            # Send event!
            # Use asyncio.to_thread because report_event uses blocking requests
            await asyncio.to_thread(
                self.report_event, 
                event_type="tcp_connection",
                severity=self.severity,
                metadata=metadata,
                action_taken=self.tarpit_mode
            )

    async def connection_wrapper(self, reader, writer, port):
        async with self.semaphore:
            await self.handle_client(reader, writer, port)

    async def monitor(self):
        """This is the required method from the SDK."""
        print(f"[*] HoneyWire Agent | Mode: {self.tarpit_mode.upper()} | Severity: {self.severity}")
        
        servers = []
        for port in self.decoy_ports:
            server = await asyncio.start_server(
                lambda r, w, p=port: self.connection_wrapper(r, w, p), 
                "0.0.0.0", 
                port
            )
            servers.append(server.serve_forever())
            print(f"[+] Tarpit listening on port {port}")
        
        await asyncio.gather(*servers)


if __name__ == "__main__":
    sensor = TcpTarpitSensor()
    try:
        # SDK's start() handles heartbeat, version sync, and calls monitor()
        asyncio.run(sensor.start()) 
    except KeyboardInterrupt:
        print("\n[*] Shutting down HoneyWire Tarpit...")