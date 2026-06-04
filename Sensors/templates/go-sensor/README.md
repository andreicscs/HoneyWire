# HoneyWire Go Sensor Template

Welcome to the HoneyWire ecosystem! This template provides the boilerplate to build a custom, Dockerized Go security sensor.

## How to Build Your Sensor

1. **Copy this folder** and rename it to your sensor's name (e.g., `ssh-watcher`).
2. **Write your logic** inside `main.go`. It comes pre-wired with the `sdk-go` event loop.
3. **Update `manifest.json`**: Add any custom `HW_` variables your sensor needs to the `env_vars` array so the Hub can dynamically configure them.

## Deployment

In the HoneyWire architecture, sensors are driven by the Hub and Wizard via `manifest.json`. 

1. Define your sensor in `manifest.json`.
2. Build and push your image to a registry.
3. Sync the manifest to your Hub, and deploy it securely using `wizard apply`.

## Testing Your Sensor

HoneyWire provides two distinct ways to test your sensor's payload delivery safely:

1. **Local Boot-Time Testing (CI/CD)**: Use the internal Mock Hub to validate your payload format without booting the full server environment.
   ```bash
   python3 scripts/mock_hub.py &
   docker run --rm -e HW_HUB_ENDPOINT=http://<LOCAL_IP>:8080 -e HW_HUB_KEY=test_key -e HW_SENSOR_ID=test_sensor -e HW_TEST_MODE=true your-sensor-image:latest
   ```