# Local Environment Setup

This guide covers how to set up HoneyWire for local development. 

For advanced CI/CD simulation (e.g., local Gitea and private Docker registries), see `maintainer-workflow.md`.

## Prerequisites

Ensure the following are installed on your development machine:
- **Go** 1.25+
- **Node.js** (v20+) and **npm**
- **Docker** v5.0+

---

## 1. Quick Start (Minimal Setup)

This is the fastest way to get the Hub and Wizard running locally for standard UI/API development.

### A. Running the Hub (Backend + Frontend)

The Hub requires both the Go backend and the Vue frontend to run simultaneously.

1. **Start the Frontend Dev Server:**
   Navigate to the UI directory and start Vite:
   ```bash
   cd Hub/ui
   npm install
   npm run dev
   ```
   *Note: Vite runs on port 5173 by default, but it is configured to proxy API requests to the Go backend.*

2. **Start the Backend Server:**
   Open a new terminal, navigate to the Hub directory, and run the Go binary:
   ```bash
   cd Hub
   HW_ENV=development HW_PORT=8080 go run cmd/hub/main.go
   ```
   *Note: `HW_ENV=development` is crucial; it disables the `Secure` flag on the authentication cookie so you can log in over `http://localhost`.*

3. **Access the Application:**
   Open your browser and navigate to `http://localhost:5173`. Complete the initial setup to generate your node keys.

### B. Running the Wizard

To test the Wizard against your local Hub:

1. Copy a Node API Key from your local Hub dashboard.
2. Run the Wizard from the repository root:
   ```bash
   go run wizard/cmd/wizard/main.go --link http://localhost:8080 --api-key <YOUR_NODE_KEY>
   ```

---

## 2. Standard Dev Setup (Testing Sensors)

When developing or modifying sensors, you need to verify their interaction with the network.

### Mocking the Hub for Sensor Testing

If you are building a sensor and only want to test its telemetry output without running the full HoneyWire Hub, use the provided Python Mock Hub script.

1. Start the Mock Hub:
   ```bash
   python3 scripts/mock_hub.py
   ```
2. Start your sensor via Docker, passing the Mock Hub's address and enabling test mode:
   ```bash
   docker run --rm \
     -e HW_HUB_ENDPOINT=http://<YOUR_LOCAL_IP>:8080 \
     -e HW_HUB_KEY=test_key \
     -e HW_SENSOR_ID=test_sensor \
     -e HW_TEST_MODE=true \
     your-sensor-image:latest
   ```

If successful, the Mock Hub will print `[EVENT] OK` to the console, proving your sensor's JSON payload adheres to the required data contracts.