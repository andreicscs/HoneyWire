# HoneyWire Operations Guide

This guide covers the day-to-day operations and lifecycle management of your HoneyWire ecosystem. The UI is designed to be intuitive, but the concepts below clarify exactly how state is managed across your distributed nodes.

## 1. Node Creation & Provisioning

A **Node** is any physical or virtual machine where you want to deploy deceptive sensors. 

1. **Create the Node:** In the Sentinel Hub UI, navigate to the **Nodes** page and click "New Node". Give it a descriptive name (e.g., `DMZ-Web-Server`).
2. **Provisioning:** Once created, the Hub will generate a unique Node API Key and a pairing command. 
3. **Link the Node:** SSH into your target machine and run the provided `honeywire link` command. This securely pairs the host's Setup Wizard to the Hub, establishing trust for future deployments.

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
2. **Rollbacks:** If an update breaks your environment, you can easily revert it. **Note:** Rollbacks are initiated using the exact same interface as updates. You simply click the version dropdown, select the older, stable version, and click the "Update" button. The Hub will treat the downgrade as a state change, and the next `honeywire apply` will seamlessly roll the container back to the older image.

## 5. Node Teardown

To cleanly remove HoneyWire from a host:

1. **Uninstall Sensors:** On the target host, run the teardown command:
   ```bash
   honeywire uninstall
   ```
   This will immediately tear down all running decoy containers, networks, and persistent volumes associated with HoneyWire on that specific host.
2. **Delete from Hub:** Return to the Hub UI and delete the Node. This permanently revokes the Node's API Key, ensuring it can never authenticate again.
