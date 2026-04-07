import os
import asyncio
import time

# Import the official HoneyWire SDK
from honeywire import HoneyWireSensor

class MyCustomSensor(HoneyWireSensor):
    def __init__(self):
        # Initialize the base class. 
        # Change 'custom' to whatever category fits your sensor (e.g., 'network', 'file', 'auth')
        super().__init__(sensor_type="custom")
        
        # Add any custom configuration from your .env file here
        # Example: self.target_path = os.getenv("HW_TARGET_PATH", "/tmp/honey")
        self.severity = os.getenv("HW_SEVERITY", "medium")

    async def monitor(self):
        """
        REQUIRED METHOD: This is the main loop of your sensor.
        Write your detection logic here. Do NOT block the async loop.
        """
        print(f"[*] Starting Custom Sensor | Severity: {self.severity}")
        
        try:
            while True:
                # --- YOUR SENSOR LOGIC GOES HERE ---
                # Example: Wait for an event...
                await asyncio.sleep(60) 
                
                # Assume an attack happened! 
                print("[!] Attack detected! Gathering forensics...")
                
                # Format your specific forensic data
                details = {
                    "source_ip": "192.168.1.100", # Replace with actual data
                    "attack_type": "example_probe",
                    "raw_payload": "GET /etc/passwd HTTP/1.1"
                }
                
                # Send the alert to the Hub using the SDK's built-in method.
                # Wrap it in asyncio.to_thread to prevent blocking your async monitor loop.
                await asyncio.to_thread(
                    self.report_event,
                    event_type="custom_anomaly_detected",
                    severity=self.severity,
                    details=details,
                    action_taken="logged"
                )
                
        except asyncio.CancelledError:
            print("[*] Monitor loop cancelled. Shutting down gracefully.")

if __name__ == "__main__":
    sensor = MyCustomSensor()
    try:
        # The SDK's start() method handles connecting to the Hub and running your monitor()
        asyncio.run(sensor.start())
    except KeyboardInterrupt:
        print("\n[*] Shutting down custom sensor...")