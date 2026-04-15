[![License](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

# Contributing to HoneyWire

Welcome to HoneyWire! We are building a centralized, high-fidelity security and deception ecosystem for homelabs and SMBs. 

Whether you want to build a new decoy sensor, improve the Vue.js frontend, or optimize the Go backend, your contributions are highly welcome.

---

## Project Structure

To help you navigate the repository, here is a high-level overview of the architecture:

```text
honeywire/
├── .github/         # CI/CD workflows
├── Docs/            # Documentation (API.md)
├── Hub/             # Central brain (Go backend + Vue frontend)
│   ├── cmd/hub/     # Main Go entrypoint (main.go)
│   ├── internal/    # Go packages (api, auth, config, models, notify, store)
│   ├── ui/          # Vue 3 Frontend (src, public, tailwind config)
│   ├── docker-compose.yml
│   └── Dockerfile
├── SDKs/            # Libraries for writing custom sensors
└── Sensors/         # Decoy nodes (Official and Community)
```

---

## Contributing to the Hub (Core)

The Hub is a unified monolith containing both the Go API and the embedded Vue 3 frontend. 

### Frontend (Vue 3 + TailwindCSS)
1. Navigate to the `Hub/ui/` directory.
2. Run `npm install` to install dependencies.
3. Run `npm run dev` to start the Vite development server.
   * *Note: You will need the Go backend running concurrently to serve the `/api/v1` routes.*
4. When your UI changes are complete, run `npm run build`. This compiles the assets into `Hub/ui/dist/`, which the Go binary automatically embeds at compile time.

### Backend (Go 1.25 + SQLite)
1. Navigate to the `Hub/` directory.
2. The backend uses `modernc.org/sqlite` (a pure Go port of SQLite) to ensure the binary remains statically linked and cross-platform without requiring CGO.
3. Run `go run cmd/hub/main.go` to start the Hub.
4. **Database Migrations:** If you need to alter the database schema, **do not** modify the `baselineSchema` string in `Hub/internal/store/store.go`. Instead, append your `ALTER TABLE` SQL command to the `migrations` array within that file. The Hub will safely backup the database and apply the migration automatically on boot.

---

## Contributing a New Sensor

To keep the ecosystem stable, all community-submitted sensors must adhere to a strict set of DevSecOps rules. We treat sensors as **isolated, unprivileged microservices**.

### The Golden Rules of Sensors
1. **Strict Sandboxing (Docker Only):** Every sensor must include a `Dockerfile`. We strongly enforce the use of minimal, hardened base images (like Distroless) running as non-root users (`UID 65532`) with all Linux kernel capabilities dropped (`cap_drop: ALL`).
2. **Zero Blast Radius:** Your sensor must not crash or overwhelm the main Hub. All communication must happen asynchronously via HTTP POST requests containing JSON.
3. **No Hardcoding:** All configurations (Ports, API keys, file paths, thresholds) must be handled dynamically via environment variables.

### How to Submit
1. **Use the Official Template:** Copy the [`Sensors/templates/go-sensor-template/`](./Sensors/templates/go-sensor-template/) folder and rename it to your sensor's name inside the [`Sensors/community/`](./Sensors/community/) directory. 
   *While you can technically build a custom sensor in any language, **pure Go is the official standard** for HoneyWire due to its minimal footprint, concurrency models, and ability to compile statically.*
2. **Follow the JSON Contract:** Your sensor must POST a payload to the Hub matching the schema outlined in the main `README.md`. *(The official HoneyWire Go SDK handles this formatting for you).*
3. **Implement Test Mode:** To ensure your code works before merging, our GitHub Actions will pass `HW_TEST_MODE=true` to your container. Your sensor must immediately send a synthetic payload to the Hub and exit gracefully.
4. **Documentation:** Provide a `README.md` within your sensor directory containing:
   * **Technical Overview:** Purpose of the sensor.
   * **Environment Reference:** A table of all `HW_` configuration variables.
   * **Deployment Example:** A secure `docker-compose.yml` snippet.
   * **Security Architecture:** An explicit breakdown of the capability drops and isolation techniques utilized.

### Review Process
Once you open a Pull Request:
1. **Automated Security Scanning:** GitHub Actions will run **Trivy** to scan your Docker image for vulnerabilities, and **CodeQL** to perform static code analysis for memory leaks.
2. **Functional Testing:** GitHub Actions will automatically build your Docker container and test it against a Mock Hub using `HW_TEST_MODE=true`.
3. **Manual Review:** A core maintainer will manually review the code, specifically checking for malicious intent and proper capability stripping.