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
            print("[SYNC]       OK | Sensor handshake complete.")
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        """Handles and validates the HoneyWire V1.0 Event Contract."""
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        try:
            payload = json.loads(post_data.decode('utf-8'))
            
            if self.path == '/api/v1/event':
                # 1. Enforce the updated V1.0 Contract
                required_keys = [
                    "contractVersion", "sensorId", "severity", 
                    "eventTrigger", "source", "target", "details"
                ]
                
                for key in required_keys:
                    if key not in payload:
                        print(f"[CONTRACT]   FAIL| Event rejected. Missing required key: '{key}'")
                        sys.exit(1)
                
                # 2. Enforce severity type (String enum or Int)
                valid_severities = ["info", "low", "medium", "high", "critical"]
                if not isinstance(payload['severity'], int) and payload['severity'] not in valid_severities:
                    print(f"[CONTRACT]   FAIL| Event rejected. Invalid severity: '{payload['severity']}'")
                    sys.exit(1)

                print(f"[EVENT]      OK | Trigger: {payload['eventTrigger']} from {payload['sensorId']}")
                print(json.dumps(payload, indent=2))
                
                self.send_response(200)
                self.end_headers()

            elif self.path == '/api/v1/heartbeat':
                self.send_response(200)
                self.end_headers()
                print(f"[HEARTBEAT]  OK | Sensor: {payload.get('sensorId', 'unknown')}")

            elif self.path == '/api/v1/offline':
                self.send_response(200)
                self.end_headers()
                print(f"[OFFLINE]    OK | Sensor: {payload.get('sensorId', 'unknown')}")
            
            # Write success flag for the CI runner
            with open('/tmp/test_passed', 'w') as f:
                f.write('success')
                
        except json.JSONDecodeError:
            print("[ERROR]      FAIL| Payload is not valid JSON.")
            sys.exit(1)
        except Exception as e:
            print(f"[ERROR]      FAIL| Unexpected error: {e}")
            sys.exit(1)

    def log_message(self, format, *args):
        pass

if __name__ == '__main__':
    print("--- HoneyWire Mock Hub (V1.1) ---")
    print("Listening on 0.0.0.0:8080...")
    print("Awaiting sensor connections for contract validation...")
    server = http.server.HTTPServer(('0.0.0.0', 8080), MockHubHandler)
    
    # We need to stay alive for at least two requests: 
    # 1. The GET sync from the SDK
    # 2. The POST alert from the sensor
    while True:
        server.handle_request()