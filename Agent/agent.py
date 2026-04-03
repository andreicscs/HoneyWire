"""
HoneyWire Tarpit Agent (Standard Port Tripwire)
Sensor type: tarpit

Environment variables:
    HUB_URL         Hub base URL              (default: http://localhost:8080)
    API_SECRET      Shared API key            (default: super_secret_key_123)
    SENSOR_ID       Unique name for this node (default: alpha-node-01)
    DECOY_PORTS     Comma-separated ports     (default: 2222,3306)
    SEVERITY        info|low|medium|high|critical (default: high)
    TARPIT_MODE     hold | echo | close       (default: hold)
"""

import asyncio
import os
import time
import threading
import requests

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
HUB_URL     = os.getenv("HUB_URL", "http://localhost:8080")
API_SECRET  = os.getenv("API_SECRET", "super_secret_key_123")
SENSOR_ID   = os.getenv("SENSOR_ID", "alpha-node-01")
SEVERITY    = os.getenv("SEVERITY", "high").lower()

raw_ports   = os.getenv("DECOY_PORTS", "2222,3306") or "2222,3306"
DECOY_PORTS = [int(p.strip()) for p in raw_ports.split(",") if p.strip()]

TARPIT_MODE = os.getenv("TARPIT_MODE", "hold").lower()

raw_banner    = os.getenv("TARPIT_BANNER", "")
TARPIT_BANNER = (
    raw_banner.encode("utf-8").decode("unicode_escape").encode("utf-8")
    if raw_banner else b""
)

DEFAULT_AGENT_VERSION = "1.0.0"
VERSION_FILE_PATH = os.path.join(os.path.dirname(__file__), "..", "VERSION")
try:
    with open(os.path.abspath(VERSION_FILE_PATH), "r") as f:
        file_version = f.read().strip()
except FileNotFoundError:
    file_version = DEFAULT_AGENT_VERSION

AGENT_VERSION = os.getenv("HONEYWIRE_VERSION", file_version)
MAX_BYTES     = 1024 * 50  # 50 KB safety cap per connection
MAX_LINES     = 10         # Max lines to store in memory for forensics
MAX_DURATION  = 3600       # Force disconnect after 1 hour to free File Descriptors
CONCURRENCY_LIMIT = 1000   # Max simultaneous connections to prevent FD exhaustion

_HEADERS      = {"x-api-key": API_SECRET}

# Global semaphore to limit concurrent TCP sockets
connection_semaphore = asyncio.Semaphore(CONCURRENCY_LIMIT)

# ---------------------------------------------------------------------------
# Hub Communication
# ---------------------------------------------------------------------------
def send_heartbeat() -> None:
    """Pings the Hub every 30 seconds to stay 'online'."""
    payload = {
        "sensor_id": SENSOR_ID,
        "sensor_type": "tarpit",
        "metadata": {
            "version": AGENT_VERSION,
            "mode": TARPIT_MODE,
            "ports": DECOY_PORTS,
            "severity_config": SEVERITY
        }
    }
    while True:
        try:
            requests.post(f"{HUB_URL}/api/v1/heartbeat", headers=_HEADERS, json=payload, timeout=5)
        except Exception as e:
            print(f"[-] Heartbeat error: {e}")
        time.sleep(30)


def report_event(source_ip: str, port: int, duration: float, payload: list[str]) -> None:
    """Reports connection to Hub using the Universal Event Standard."""
    event = {
        "contract_version": "1.0",
        "sensor_id": SENSOR_ID,
        "sensor_type": "tarpit",
        "event_type": "tcp_connection",
        "severity": SEVERITY,
        "source": source_ip,
        "target": f"Port {port}",
        "action_taken": TARPIT_MODE,
        "details": {
            "duration_sec": round(duration, 2),
            "payload": payload,
        }
    }
    try:
        requests.post(f"{HUB_URL}/api/v1/event", headers=_HEADERS, json=event, timeout=5)
        print(f"[+] Alert sent: {source_ip} -> Port {port} ({SEVERITY.upper()})")
    except Exception as e:
        print(f"[-] Event report failed: {e}")

# ---------------------------------------------------------------------------
# Core Logic
# ---------------------------------------------------------------------------
async def handle_client(reader: asyncio.StreamReader, writer: asyncio.StreamWriter, port: int) -> None:
    addr = writer.get_extra_info("peername")
    source_ip = addr[0] if addr else "Unknown"
    
    start_time = time.time()
    payload = []
    total_bytes = 0

    try:
        if TARPIT_BANNER and TARPIT_MODE != "close":
            writer.write(TARPIT_BANNER)
            await writer.drain()

        if TARPIT_MODE != "close":
            while total_bytes < MAX_BYTES and (time.time() - start_time) < MAX_DURATION:
                try:
                    # Added a default timeout even for "hold" to allow TTL checks
                    timeout = 60.0 if TARPIT_MODE == "echo" else 300.0 
                    data = await asyncio.wait_for(reader.read(1024), timeout=timeout)
                    
                    if not data: break
                    
                    decoded = data.decode("utf-8", errors="replace").strip()
                    if decoded: 
                        # 🔒 SECURITY: Only store the first 10 lines to prevent memory bloat
                        if len(payload) < MAX_LINES:
                            payload.append(decoded)
                            
                    total_bytes += len(data)

                    if TARPIT_MODE == "echo":
                        writer.write(data)
                        await writer.drain()
                    
                    await asyncio.sleep(0.5) # Anti-spam delay
                    
                except asyncio.TimeoutError:
                    if not writer.is_closing():
                        writer.write(b"\0") # Heartbeat byte to keep connection active
                        await writer.drain()
                    continue
                    
    except Exception:
        pass # Expected when attackers forcefully close the socket
    finally:
        # 🔒 SECURITY: Try/Except prevents Ghost Bypass on TCP RST
        try:
            writer.close()
            await writer.wait_closed()
        except Exception:
            pass 
        
        duration = time.time() - start_time
        await asyncio.to_thread(report_event, source_ip, port, duration, payload)


async def connection_wrapper(reader: asyncio.StreamReader, writer: asyncio.StreamWriter, port: int) -> None:
    """Wraps handle_client in a Semaphore to prevent File Descriptor exhaustion."""
    async with connection_semaphore:
        await handle_client(reader, writer, port)


async def main() -> None:
    print(f"[*] HoneyWire Agent v{AGENT_VERSION} | Severity: {SEVERITY.upper()}")
    threading.Thread(target=send_heartbeat, daemon=True).start()
    
    servers = []
    for port in DECOY_PORTS:
        server = await asyncio.start_server(lambda r, w, p=port: connection_wrapper(r, w, p), "0.0.0.0", port)
        servers.append(server.serve_forever())
        print(f"[+] Listening on port {port}")
    
    await asyncio.gather(*servers)

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        pass