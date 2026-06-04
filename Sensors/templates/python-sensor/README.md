# HoneyWire Python Sensor Template

Welcome to the HoneyWire ecosystem! This template contains everything you need to build a custom, Dockerized security sensor that natively reports to the HoneyWire Hub.

## How to Build Your Sensor

1. **Copy this folder** and rename it to your sensor's name (e.g., `ssh_watcher`).
2. **Write your logic** inside `sensor.py`. The file is heavily commented and shows you exactly where to put your code.
3. **Add dependencies**: If your script needs extra Python libraries (like `scapy` or `paramiko`), add them to `requirements.txt`.
4. **Update `manifest.json`**: Add any custom `HW_` variables your sensor needs to the `env_vars` array so the Hub can configure them.

## Deployment

In the new HoneyWire architecture, sensors are driven by the Hub and Wizard via `manifest.json`. 

1. Define your sensor in `manifest.json`.
2. Build and push your image to a registry.
3. Sync the manifest to your Hub, and deploy it securely using `wizard apply`.

This ensures your sensor runs under strict Distroless sandboxing with dropped capabilities (`cap_drop: ALL`) and read-only file systems out-of-the-box.

## Testing Your Sensor

HoneyWire provides two distinct ways to test your sensor's payload delivery safely:

1. **Local Boot-Time Testing (CI/CD)**: Use the internal Mock Hub to validate your payload format without booting the full server environment.
   ```bash
   # Start the Mock Hub in a separate terminal
   python3 scripts/mock_hub.py
   
   # Run your sensor in test mode
   docker run --rm \
     -e HW_HUB_ENDPOINT=http://<LOCAL_IP>:8080 \
     -e HW_HUB_KEY=test_key -e HW_SENSOR_ID=test_sensor -e HW_TEST_MODE=true \
     your-sensor-image:latest
   ```