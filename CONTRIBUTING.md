# Contributing to HoneyWire

Welcome to HoneyWire! We are building a centralized, high-fidelity security ecosystem for homelabs and SMBs. 

To keep the ecosystem stable, all community-submitted sensors must adhere to a strict set of rules. We treat sensors as **isolated microservices**. 

## The Golden Rules of Sensors

1. **Self-Contained (Docker Only):** Every sensor must include a `Dockerfile`. Users should not need to install Python, Node, Go, or Rust on their host machine to run your sensor.
2. **Zero Blast Radius:** Your sensor must not crash the main Hub. All communication must happen via HTTP POST requests containing JSON.
3. **No Hardcoding:** All configurations (Ports, API keys, file paths) must be handled via environment variables inside a `.env` file.

## How to Submit a New Sensor

### 1. Use the Template
Copy the [`Sensors/templates/python-sensor/`](./Sensors/templates/python-sensor/) folder and rename it to your sensor's name inside the [`Sensors/community/`](./Sensors/community/) directory.
You can also build a custom sensor in **any language**, the templates are only there to make it easier for people to start contributing!

### 2. Follow the JSON Contract (v1.0)
Your sensor must POST a payload to the Hub (`HW_HUB_ENDPOINT`) matching this exact schema:

```json
{
  "contract_version": "1.0",
  "sensor_id": "provided-by-env",
  "sensor_type": "your_sensor_category",
  "event_type": "what_just_happened",
  "severity": "critical", 
  "timestamp": "2026-04-03T01:24:18Z",
  "action_taken": "logged",
  "metadata": {
    "ip": "192.168.1.5",
    "custom_data": "anything you want"
    .
    .
    .
  }
}
```
*(Note: If you use the official HoneyWire SDK provided in the template, this formatting is handled for you automatically).*

### 3. Implement Test Mode (Required for CI/CD)
To ensure your code works before merging, our GitHub Actions will build your Docker container and pass `HW_TEST_MODE=true` as an environment variable. 

If this variable is present, your sensor **must immediately send a dummy payload to the Hub and exit**. (The Python SDK handles this out-of-the-box).

### 4. Provide Documentation
Provide a README.md within your sensor directory containing:
  - Technical Overview: Purpose of the sensor and the "lure" or monitoring it provides.
  - Environment Reference: A table of all HW_ variables.
  - Deployment Example: A docker-compose.yaml and a .env.example snippet.

## Review Process
Once you open a Pull Request:
1. **Functional Testing:** GitHub Actions will automatically build your Docker container and test it against a Mock Hub using `HW_TEST_MODE=true`.
2. **Automated Security Scanning:** GitHub Actions will run **Trivy** to scan your Docker image for OS and library vulnerabilities, and **CodeQL** to perform static code analysis for insecure patterns and security flaws.
3. **Manual Review:** A core maintainer will manually review the code for malicious intent or blast-radius risks before merging. PRs that fail automated testing or scanning will not be reviewed.