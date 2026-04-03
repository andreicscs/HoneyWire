# HoneyWire Python Sensor Template

Welcome to the HoneyWire ecosystem! This template contains everything you need to build a custom, Dockerized security sensor that natively reports to the HoneyWire Hub.

## How to Build Your Sensor

1. **Copy this folder** and rename it to your sensor's name (e.g., `ssh_watcher`).
2. **Write your logic** inside `sensor.py`. The file is heavily commented and shows you exactly where to put your code.
3. **Add dependencies**: If your script needs extra Python libraries (like `scapy` or `paramiko`), add them to `requirements.txt`.
4. **Update `.env.example`**: Add any custom `HW_` variables your sensor needs.

## Testing Your Sensor

You can test your sensor locally by running:

```bash
# Copy the example environment file and fill in your Hub details
cp .env.example .env

# Build and run the Distroless Docker container
docker build -t honeywire-custom-sensor .
docker run --rm --env-file .env honeywire-custom-sensor
```

## CI/CD Requirement
To ensure your sensor works, our GitHub Actions will run it with HW_TEST_MODE=true. The base SDK handles this automatically, so you don't need to write any test logic! Just ensure you don't override the start() method in the base class.