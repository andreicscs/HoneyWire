# Maintainer Workflow & Advanced Setup

**⚠️ NOTE: This document is for core maintainers. It is NOT required for standard development or OSS contributions.** 
If you just want to build a sensor or fix a UI bug, please refer to `setup.md` instead.

This guide outlines how to set up an advanced local environment. Because HoneyWire is a distributed orchestration platform, testing the full lifecycle from pushing a sensor image to the Hub compiling a manifest, to the Wizard pulling and deploying that image requires simulating a real infrastructure ecosystem.

---

## 1. Why an Advanced Setup?

In standard development, testing a sensor involves running it directly with Docker and pointing it at a Mock Hub. 
However, to test changes to the Hub Compiler or the Wizard's Deployment Engine, you need:
1. A real Docker Registry so the Wizard can execute `docker compose pull` on your test images.
2. A local Hub running to generate the `honeywire-compose.yml`.
3. The Wizard connecting to the local Hub to fetch intents and apply them to the Docker daemon.

---

## 2. Setting Up a Local Registry and Gitea

To simulate the production supply chain, we run a local Docker registry (for images) and optionally a local Gitea instance (if you are testing CI automation hooks).

Create a `docker-compose.test-infra.yml` file outside the HoneyWire repository:

```yaml
services:
  # Local Docker Registry
  registry:
    image: registry:2
    ports:
      - "5000:5000"
    restart: always

  # Local Gitea (Optional: for testing webhooks/CI)
  gitea:
    image: gitea/gitea:latest
    ports:
      - "3000:3000"
      - "222:22"
    environment:
      - USER_UID=1000
      - USER_GID=1000
    volumes:
      - ./gitea_data:/data
    restart: always
```

Run `docker compose -f docker-compose.test-infra.yml up -d`.

---

## 3. The End-to-End Testing Flow

### Step A: Push your Test Sensor to the Local Registry
1. Build your custom sensor or experimental branch locally:
   ```bash
   docker build -t localhost:5000/honeywire/test-sensor:latest ./Sensors/official/TcpTarpit
   ```
2. Push it to your local registry:
   ```bash
   docker push localhost:5000/honeywire/test-sensor:latest
   ```
3. *Crucial:* Modify your test manifest (`manifest.dev.json`) so the `deployment.image` points to `localhost:5000/honeywire/test-sensor:latest` and upload/sync this manifest to your local Hub.

### Step B: Run the Local Hub
Start the Hub in development mode. Leave this running in a dedicated terminal pane so you can monitor the logs for HTTP requests and compilation errors.
```bash
cd Hub
HW_ENV=development HW_PORT=8080 go run cmd/hub/main.go
```
*(Ensure the UI is also running via `npm run dev` in `Hub/ui` if you are working on the ui, you can also use the hub's embedded ui at `http://localhost:8080`if you need to use the dashboard to provision API keys).*

### Step C: Test the Wizard Locally Against the Hub
You do not need to compile the Wizard to test it. You can run it directly via Go, passing your local Hub endpoint.

1. **Link the Node:**
   Provision a new Node in your local Hub dashboard, copy the API key, and link your local machine:
   ```bash
   go run wizard/cmd/wizard/main.go --link http://localhost:8080 --api-key <YOUR_NODE_KEY>
   ```

2. **Test Discovery:**
   Run the discovery engine to ensure it correctly evaluates your local host's `/proc` and ports:
   ```bash
   go run wizard/cmd/wizard/main.go discover
   ```

3. **Test Apply (Full Cycle):**
   Trigger the deployment. The Wizard will fetch the deployment intent from the local Hub, generate the compose file, pull the image from your `localhost:5000` registry, and apply it to your local Docker daemon:
   ```bash
   go run wizard/cmd/wizard/main.go apply
   ```

---

## 4. Releasing a Sensor Version

Sensor releases are automated via the registry pipeline. **You never create versioned manifest files manually.**

### Step 1: Edit the Source Manifest
The authoring surface requires you to add the `manifest.json` file directly inside the individual sensor's root directory:
- `Sensors/official/tcp-tarpit/manifest.json`
- `Sensors/official/file-canary/manifest.json`
- `Sensors/official/web-router-decoy/manifest.json`

This decoupled architecture allows the Hub to dynamically index sensors without monolithic configuration files. Edit the file, update documentation, env vars, heuristics, etc. Commit to `main`.

### Step 2: Push a Namespaced Git Tag
```bash
git tag sensor/file-canary/v1.2.0
git push origin sensor/file-canary/v1.2.0
```

> [!WARNING]
> **Immutability Rule:** In standard GitOps workflows, release tags are immutable. If you push `v1.2.0`, realize it's broken, and rollback your fleet, you **must not** overwrite the `v1.2.0` tag with a fix. Doing so causes cache poisoning in the Hub backend and Docker registries. Always bump the version (e.g. `v1.2.1`) and leave the rolled-back tag untouched.

### Step 3: CI Handles the Rest
The `publish-sensor-registry` Gitea Action will:
1. Read `Sensors/official/FileCanary/file-canary.json` at the tagged commit
2. Inject `"version": "1.2.0"` and update the Docker `image_tag` to `"1.2.0"`
3. Write `file-canary-v1.2.0.json` to the `registry-pages` branch
4. Regenerate `index.json`

### Step 4: Verify
Check the `registry-pages` branch to confirm:
- The versioned JSON file exists
- `index.json` lists the new version
- The `latest` field points to the new version

### Step 5: Dashboard Sync & Manual Upgrades
Refresh your HoneyWire dashboard or wait for the automatic UI sync. The Hub's event-driven catalog hook will instantly detect the registry mutation and compare it against deployed sensors.

Instead of forcefully upgrading production edge nodes automatically, the Hub will flag nodes with an **"Update Available"** indicator. Users must manually trigger the `/api/v1/nodes/{id}/upgrade` endpoint (via the UI) to instruct the node to pull the new version schema and execute a compose restart.
