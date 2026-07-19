# HoneyWire Operations Guide

This guide covers the day-to-day operations and lifecycle management of your HoneyWire ecosystem. The UI is designed to be intuitive, but the concepts below clarify exactly how state is managed across your distributed nodes.

## 1. Node Creation & Provisioning

A **Node** is any physical or virtual machine where you want to deploy deceptive sensors. 

1. **Create the Node:** In the Sentinel Hub UI, navigate to the **Nodes** page and click "New Node". Give it a descriptive name (e.g., `DMZ-Web-Server`).
2. **Provisioning:** Once created, the Hub will generate a unique Node API Key and a setup command. 
3. **Link the Node:** SSH into your target machine and run the provided command. This command is an all-in-one script (`curl -fsSL ... | bash`) that installs the Setup Wizard on the host and immediately links the node to the Hub, establishing trust for its first deployment and all future deployment updates.

## 2. Sensor Deployment

HoneyWire uses a declarative deployment model. You tell the Hub what you *want* the node to run, and the Hub compiles that desired state.

* **Assigning Sensors:** From the Hub UI, select a Node and browse the Registry. Choose the sensors you want to deploy (e.g., *TCP Tarpit*, *File Canary*) and configure their environment variables (like ports or trigger thresholds).
* **Desired State:** Once you assign a sensor, the Hub updates the Node's "Desired State." The sensor is not actively running on the host yet it is merely queued for deployment.

## 3. Node Synchronization (`apply`)

Because the Setup Wizard is an ephemeral, on-demand CLI tool rather than a constant 24/7 background agent, changes made in the UI must be explicitly pushed to the host.

* **Reconciliation:** To sync a Node, SSH into the target host and run:
  ```bash
  honeywire apply
  ```
* The Wizard will pull the latest compiled desired state from the Hub, compare it against the currently running Docker containers, and automatically spin up, reconfigure, or tear down sensors to match the Hub.

> [!IMPORTANT]
> **Strict State Enforcement:** The `honeywire apply` command strictly enforces the configuration defined in the Hub. It does **not** perform local conflict resolution or template evaluation. If you manually configure a sensor to use port 8080 in the Hub UI, the node will blindly attempt to bind to 8080 even if that port is already in use by another application, resulting in a crash. It is the operator's responsibility to ensure manually configured settings are valid.
> Conversely, running the automated `honeywire discover` command directly on the host allows the wizard to proactively scan the environment, resolve port conflicts, and dynamically calculate a safe configuration before uploading it to the Hub.

## 4. Sensor Updates and Rollbacks

When community maintainers or official developers push a new version of a sensor to the Registry, you can manage the upgrade lifecycle directly from the UI.

1. **Updates:** If a newer version of an installed sensor is available, an "Update Available" badge will appear next to it on the Node page. Click the update button to queue the new version in the desired state. Run `honeywire apply` on the Node to execute the upgrade.
2. **Rollbacks:** The operator cannot perform a rollback independently. Rollbacks are managed by the core maintainers when a published update goes wrong. If a rollback is issued by the developers, it appears in the UI identically to an upgrade: an "Update Available" badge will appear, and clicking "Update" will actually revert the sensor to the older, stable version. The next `honeywire apply` will then seamlessly roll the container back.

## 5. Node Teardown

To cleanly remove HoneyWire from a host:

1. **Uninstall Sensors:** On the target host, run the teardown command:
   ```bash
   honeywire uninstall
   ```
   This action strictly targets HoneyWire resources. It will cleanly stop and remove the isolated Docker containers, delete the `/opt/honeywire` configuration directory, remove the node identity file at `/etc/honeywire/config.json`, and delete the CLI binary itself. It will **not** blindly wipe files or perform wide-ranging host cleanups, ensuring no important host data is ever accidentally deleted.
2. **Delete from Hub:** Return to the Hub UI and delete the Node. This permanently revokes the Node's API Key, ensuring it can never authenticate again.

## 6. Updating the Setup Wizard

The Setup Wizard itself is updated independently from the sensors. The same curl script provided by the Hub to install the wizard can also be used to safely update it in place:

```bash
curl -fsSL https://get.honeywire.dev | bash
```

This will download the latest binary for your architecture and replace the existing installation without affecting any running sensors or node configurations.

## 7. Updating the Sentinel Hub

Because the Hub is deployed as a standard Docker container, updating it is incredibly straightforward. You just need to pull the latest image and restart your compose stack. On the server hosting the Hub, run:

```bash
docker compose pull
docker compose up -d
```

This will cleanly restart the Hub with the newest release while preserving all of your persistent data, node identities, and configurations (which are safely stored in the mapped volumes).

> [!WARNING]
> **Major Version Upgrades:** While minor and patch updates (e.g., `v2.0.1` to `v2.0.4`) are safe to pull automatically, **Major** version bumps (e.g., `v2.x` to `v3.0`) often contain breaking changes, database schema migrations, or new environment variables. Always read the [GitHub Release Notes](https://github.com/AndReicscs/HoneyWire/releases) before pulling a major version to ensure you do not corrupt your Hub instance.
