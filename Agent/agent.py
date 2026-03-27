import asyncio
import os
import time
import requests
import threading

# --- Configuration ---
HUB_URL = os.getenv("HUB_URL", "http://localhost:8080")
API_SECRET = os.getenv("API_SECRET", "super_secret_key_123")
SENSOR_ID = os.getenv("SENSOR_ID", "alpha-node-01")
SENSOR_IP = os.getenv("SENSOR_IP", "127.0.0.1")

# Safely parse ports
raw_ports = os.getenv("DECOY_PORTS", "2222,3306")
if not raw_ports:
    raw_ports = "2222,3306"
DECOY_PORTS = [int(p.strip()) for p in raw_ports.split(",") if p.strip()]

TARPIT_MODE = os.getenv("TARPIT_MODE", "hold").lower()

# Safely parse the banner
raw_banner = os.getenv("TARPIT_BANNER", "")
if raw_banner:
    TARPIT_BANNER = raw_banner.encode('utf-8').decode('unicode_escape').encode('utf-8')
else:
    TARPIT_BANNER = b"" # Default to nothing if empty

# --- Hub Communication ---
def send_heartbeat():
    while True:
        try:
            requests.post(
                f"{HUB_URL}/api/v1/heartbeat",
                headers={"x-api-key": API_SECRET},
                json={"sensor_id": SENSOR_ID, "sensor_ip": SENSOR_IP},
                timeout=5
            )
        except Exception as e:
            print(f"[-] Failed to ping Hub: {e}")
        time.sleep(30)

def report_event(attacker_ip, port, duration, payload_buffer):
    try:
        requests.post(
            f"{HUB_URL}/api/v1/event",
            headers={"x-api-key": API_SECRET},
            json={
                "sensor_id": SENSOR_ID,
                "sensor_ip": SENSOR_IP,
                "attacker_ip": attacker_ip,
                "target_port": port,
                "protocol": "TCP",
                "action": TARPIT_MODE, # Log the specific action taken
                "duration_sec": duration,
                "raw_payload": payload_buffer 
            },
            timeout=5
        )
        print(f"[+] Reported attack from {attacker_ip} to Hub.")
    except Exception as e:
        print(f"[-] Failed to report event: {e}")

# --- The Tarpit Logic ---
async def handle_client(reader, writer, port):
    addr = writer.get_extra_info('peername')
    attacker_ip = addr[0] if addr else "Unknown"
    print(f"[!] Incoming connection from {attacker_ip} on port {port}")
    
    start_time = time.time()
    payload_buffer = []
    total_bytes_received = 0
    max_bytes = 1024 * 50 # 50KB safety limit

    try:
        # 1. THE BANNER
        # Only send a banner if one exists AND we aren't immediately closing the door
        if TARPIT_BANNER and TARPIT_MODE != "close":
            writer.write(TARPIT_BANNER)
            await writer.drain()
        
        # 2. THE ACTION
        if TARPIT_MODE == "close":
            # Slam the door immediately
            pass
        
        elif TARPIT_MODE == "echo":
            while True:
                try:
                    data = await asyncio.wait_for(reader.read(1024), timeout=60.0)
                    if not data:
                        break # Attacker closed connection

                    # Record the payload
                    decoded_data = data.decode('utf-8', errors='replace').strip()
                    if decoded_data:
                        payload_buffer.append(decoded_data)
                        total_bytes_received += len(data)

                    # Waste time, then echo back
                    await asyncio.sleep(1)
                    writer.write(data)
                    await writer.drain()

                    # Safety check
                    if total_bytes_received > max_bytes:
                        print(f"[*] {attacker_ip} hit buffer limit. Dropping.")
                        break

                except asyncio.TimeoutError:
                    # Infinite hold trick: send a null byte to reset their timeout clock
                    writer.write(b"\0")
                    await writer.drain()

        elif TARPIT_MODE == "hold":
            while True:
                data = await reader.read(1024)
                if not data:
                    break
                
                # Record the payload even in hold mode
                decoded_data = data.decode('utf-8', errors='replace').strip()
                if decoded_data:
                    payload_buffer.append(decoded_data)
                    total_bytes_received += len(data)

                if total_bytes_received > max_bytes:
                    break

                # Sleep to prevent CPU spinning, holding their thread locked
                await asyncio.sleep(1)
                
    except Exception as e:
        # Connection violently reset by attacker
        pass 
    finally:
        # Cleanup and Reporting
        writer.close()
        await writer.wait_closed()
        
        duration = round(time.time() - start_time, 2)
        print(f"[*] Connection closed with {attacker_ip}. Held for {duration}s.")
        await asyncio.to_thread(report_event, attacker_ip, port, duration, payload_buffer)

async def start_tarpit(port):
    server = await asyncio.start_server(
        lambda r, w: handle_client(r, w, port), 
        '0.0.0.0', 
        port
    )
    print(f"[*] Tarpit armed and listening on Port {port}")
    async with server:
        await server.serve_forever()

async def main():
    print(f"=== HoneyWire Agent Starting (Mode: {TARPIT_MODE.upper()}) ===")
    
    threading.Thread(target=send_heartbeat, daemon=True).start()
    
    tasks = [start_tarpit(port) for port in DECOY_PORTS]
    await asyncio.gather(*tasks)

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\n[*] Agent shutting down.")