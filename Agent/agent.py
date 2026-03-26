import asyncio
import os
import time
import requests
import threading

# --- Configuration ---
HUB_URL = os.getenv("HUB_URL", "http://localhost:8080")
API_SECRET = os.getenv("API_SECRET", "super_secret_key_123")
SENSOR_ID = os.getenv("SENSOR_ID", "alpha-node-01")
SENSOR_IP = os.getenv("SENSOR_IP", "127.0.0.1") # In production, this would be the machine's LAN IP
DECOY_PORTS = [int(p) for p in os.getenv("DECOY_PORTS", "2222,3306").split(",")]

# --- Hub Communication ---
def send_heartbeat():
    """Runs in a background thread, pinging the Hub every 30 seconds."""
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
    """Fires the JSON alert to the Hub."""
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
                "action": "tarpitted",
                "duration_sec": duration,
                "raw_payload": payload_buffer # Send the array of commands
            },
            timeout=5
        )
        print(f"[+] Reported attack from {attacker_ip} to Hub.")
    except Exception as e:
        print(f"[-] Failed to report event: {e}")

# --- The Tarpit Logic ---
async def handle_client(reader, writer, port):
    addr = writer.get_extra_info('peername')
    attacker_ip = addr[0]
    print(f"[!] Incoming connection from {attacker_ip} on port {port}")
    
    start_time = time.time()
    payload_buffer = []
    total_bytes_received = 0
    max_bytes = 1024 * 50 # 50KB safety limit so they can't crash our memory

    try:
        # 1. Send a fake welcome banner to entice the bot to start talking
        writer.write(b"Connected. Please authenticate.\r\n")
        await writer.drain()

        while True:
            try:
                # 2. Wait for them to send data. 
                # If they sit in silence for 60 seconds, we throw a TimeoutError.
                data = await asyncio.wait_for(reader.read(1024), timeout=60.0)
                
                # If data is empty, the attacker closed the connection.
                if not data:
                    break

                # 3. Log what they sent
                decoded_data = data.decode('utf-8', errors='replace').strip()
                if decoded_data:
                    payload_buffer.append(decoded_data)
                    total_bytes_received += len(data)

                # 4. THE ECHO TARPIT
                # We wait 1 second to waste their time, then echo their own garbage back to them
                await asyncio.sleep(1)
                writer.write(data)
                await writer.drain()

                # Safety check
                if total_bytes_received > max_bytes:
                    print(f"[*] {attacker_ip} hit buffer limit. Dropping.")
                    break

            except asyncio.TimeoutError:
                # 5. THE INFINITE HOLD
                # If they are silent, send a single null byte to reset their timeout timer
                # This holds their socket open indefinitely.
                writer.write(b"\0")
                await writer.drain()

    except Exception as e:
        # Connection violently reset by attacker
        pass 
    finally:
        # 6. Cleanup and Reporting
        writer.close()
        await writer.wait_closed()
        
        duration = round(time.time() - start_time, 2)
        print(f"[*] Connection closed with {attacker_ip}. Held for {duration}s.")
        
        # Only report if they actually sent a payload, OR if we held them for more than 5 seconds
        if payload_buffer or duration > 5.0:
            # We use asyncio.to_thread so the HTTP request doesn't pause the async trap
            await asyncio.to_thread(report_event, attacker_ip, port, duration, payload_buffer)

async def start_tarpit(port):
    """Starts an asyncio socket server on a specific port."""
    server = await asyncio.start_server(
        lambda r, w: handle_client(r, w, port), 
        '0.0.0.0', 
        port
    )
    print(f"[*] Tarpit armed and listening on Port {port}")
    async with server:
        await server.serve_forever()

async def main():
    print("=== HoneyWire Agent Starting ===")
    
    # Start the heartbeat in the background
    threading.Thread(target=send_heartbeat, daemon=True).start()
    
    # Start a tarpit listener for every port in our config
    tasks = [start_tarpit(port) for port in DECOY_PORTS]
    await asyncio.gather(*tasks)

if __name__ == "__main__":
    # Standard graceful exit handling for asyncio
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\n[*] Agent shutting down.")