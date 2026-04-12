import os
import sys
import asyncio
import traceback

# Import the official HoneyWire SDK
from honeywire import HoneyWireSensor

class MyCustomSensor(HoneyWireSensor):
    def __init__(self):
        # Initialize the base class. 
        super().__init__(sensor_type="custom")
        
        # Custom config overrides
        self.severity = os.getenv("HW_SEVERITY", "medium")
        self.target = os.getenv("HW_CUSTOM_TARGET", "/tmp/honey")

    async def monitor(self):
        """
        REQUIRED METHOD: This is the main loop of your sensor.
        """
        print(f"[*] Starting Custom Sensor | Target: {self.target} | Severity: {self.severity}")
        
        try:
            while True:
                # --- YOUR SENSOR LOGIC GOES HERE ---
                # Example: Wait for an event...
                await asyncio.sleep(60) 
                
                # Assume an attack happened! 
                print("[!] Attack detected! Gathering forensics...")
                
                # Format your specific forensic data
                details = {
                    "attack_type": "example_probe",
                    "raw_payload": "GET /etc/passwd HTTP/1.1"
                }
                
                # Send the alert to the Hub using the SDK's built-in method.
                # Wrap it in asyncio.to_thread to prevent blocking your async monitor loop.
                await asyncio.to_thread(
                    self.report_event,
                    event_trigger="custom_anomaly_detected",
                    severity=self.severity,
                    source="192.168.1.100", # Replace with actual IP
                    target=self.target,
                    details=details
                )
                
        except asyncio.CancelledError:
            print("[*] Monitor loop cancelled. Shutting down gracefully.")

if __name__ == "__main__":
    # 1. Initialize SDK safely
    try:
        sensor = MyCustomSensor()
    except ValueError as e:
        print(f"[!] FATAL: {e}")
        sys.exit(1)

    # 2. Handle Test Mode
    if sensor.test_mode:
        if sensor.run_test_mode():
            print("✅ Test mode complete. Exiting gracefully.")
            sys.exit(0)
        else:
            print("❌ Test mode failed to contact Hub.")
            sys.exit(1)

    # 3. Start the Async Loop
    try:
        asyncio.run(sensor.start())
    except KeyboardInterrupt:
        print("\n[*] Shutting down custom sensor...")
    except Exception as e:
        print(f"[!] Fatal error: {e}")
        traceback.print_exc()
    finally:
        sensor.stop() # Prevents dangling heartbeat threads!