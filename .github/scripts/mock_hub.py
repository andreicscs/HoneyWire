import http.server
import json
import sys
import os

class MockHubHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        """Handles the HoneyWire SDK synchronization handshake."""
        if self.path == '/api/v1/version':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            
            # The SDK expects a version object to confirm compatibility
            response = {"version": "1.0.0", "status": "operational"}
            self.wfile.write(json.dumps(response).encode('utf-8'))
            print("🤝 SUCCESS: Sensor synchronized successfully via GET.")
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        """Handles and validates the HoneyWire V1.0 Event Contract."""
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        try:
            payload = json.loads(post_data.decode('utf-8'))
            
            # 1. Enforce the V1.0 Contract
            required_keys = [
                "contract_version", "sensor_id", "sensor_type", 
                "event_type", "severity", "timestamp", "details"
            ]
            
            for key in required_keys:
                if key not in payload:
                    print(f"❌ FAILED: Missing required key '{key}'")
                    sys.exit(1)
            
            # 2. Enforce severity type (String enum or Int)
            valid_severities = ["info", "low", "medium", "high", "critical"]
            if not isinstance(payload['severity'], int) and payload['severity'] not in valid_severities:
                print(f"❌ FAILED: 'severity' ({payload['severity']}) is invalid.")
                sys.exit(1)

            print(f"🚩 ALERT RECEIVED: {payload['event_type']} from {payload['sensor_id']}")
            print(json.dumps(payload, indent=2))
            
            self.send_response(200)
            self.end_headers()
            
            # Write success flag for the CI runner
            with open('/tmp/test_passed', 'w') as f:
                f.write('success')
                
            print("✅ SUCCESS: Event validated. Waiting for next sensor...")
                
        except json.JSONDecodeError:
            print("❌ FAILED: Payload is not valid JSON.")
            sys.exit(1)
        except Exception as e:
            print(f"❌ FAILED: Unexpected error - {e}")
            sys.exit(1)

    def log_message(self, format, *args):
        pass

if __name__ == '__main__':
    print("🛡️ HoneyWire Mock Hub (V1.1) listening on port 8080...")
    server = http.server.HTTPServer(('0.0.0.0', 8080), MockHubHandler)
    
    # We need to stay alive for at least two requests: 
    # 1. The GET sync from the SDK
    # 2. The POST alert from the sensor
    while True:
        server.handle_request()