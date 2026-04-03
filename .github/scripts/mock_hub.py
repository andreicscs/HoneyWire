import http.server
import json
import sys

class MockHubHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        try:
            payload = json.loads(post_data.decode('utf-8'))
            
            # 1. Enforce the V1.0 Contract
            required_keys = [
                "contract_version", "sensor_id", "sensor_type", 
                "event_type", "severity", "timestamp", "metadata"
            ]
            
            for key in required_keys:
                if key not in payload:
                    print(f"❌ FAILED: Missing required key '{key}'")
                    sys.exit(1)
            
            # 2. Enforce severity type
            if not isinstance(payload['severity'], int) and payload['severity'] not in ["info", "low", "medium", "high", "critical"]:
                print("❌ FAILED: 'severity' must be an int or valid enum.")
                sys.exit(1)

            print("✅ SUCCESS: Valid HoneyWire V1.0 payload received.")
            print(json.dumps(payload, indent=2))
            
            self.send_response(200)
            self.end_headers()
            
            # Write a success flag for the CI runner
            with open('/tmp/test_passed', 'w') as f:
                f.write('success')
                
        except json.JSONDecodeError:
            print("❌ FAILED: Payload is not valid JSON.")
            sys.exit(1)
        except Exception as e:
            print(f"❌ FAILED: Unexpected error - {e}")
            sys.exit(1)

    def log_message(self, format, *args):
        # Suppress default HTTP logging to keep CI logs clean
        pass

if __name__ == '__main__':
    print("🛡️ Mock Hub listening on port 8080...")
    server = http.server.HTTPServer(('0.0.0.0', 8080), MockHubHandler)
    server.handle_request() # Process exactly one request then exit