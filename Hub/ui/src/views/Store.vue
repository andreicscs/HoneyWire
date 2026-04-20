<script setup>
import { ref } from 'vue'
import { useConfig } from '../api/useConfig'

// Pull the global reactive config state
const { config } = useConfig()

const selectedSensor = ref(null)
const activeTab = ref('readme') 
const editableCompose = ref('') 

const openSensor = (sensor) => {
    selectedSensor.value = sensor
    activeTab.value = 'readme'
    // This now dynamically grabs the DB settings the moment you click the card
    editableCompose.value = getComposeFile(sensor.compose)
    document.body.style.overflow = 'hidden' 
}

const closeSensor = () => {
    selectedSensor.value = null
    document.body.style.overflow = ''
}

const getComposeFile = (composeString) => {
    // Read from the backend config FIRST, fallback to the browser URL if empty
    const endpoint = config.hubEndpoint || window.location.origin
    const key = config.hubKey || '<YOUR_HW_HUB_KEY>'
    
    return composeString
        .replace(/__HUB_ENDPOINT__/g, endpoint)
        .replace(/__HUB_KEY__/g, key)
}

const copyToClipboard = () => {
    if (!selectedSensor.value) return
    navigator.clipboard.writeText(editableCompose.value)
    
    const btn = document.getElementById('copy-btn')
    const originalText = btn.innerHTML
    btn.innerHTML = 'Copied!'
    setTimeout(() => { btn.innerHTML = originalText }, 2000)
}

const sensors = [
    {
        id: 'file-canary',
        name: 'File Canary (FIM)',
        osi: 'Host Level',
        shortDesc: 'Honeypot and File Integrity Monitor. Watches files/directories for unauthorized modifications or drops.',
        icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
        compose: `services:
  # 1. THE FIM SETTER: Only runs if the path exists. Simply grants the read-only ACL pass.
  permission-fixer:
    image: alpine:latest
    command: sh -c "apk add --no-cache acl && setfacl -R -m u:65532:rx /honey_dir"
    volumes:
      # LONG SYNTAX: This forces Docker to throw an error if \${TRAP_PATH} does not exist!
      # Define TRAP_PATH in your .env file (e.g., TRAP_PATH=/opt/fake_secrets)
      - type: bind
        source: \${TRAP_PATH}
        target: /honey_dir

  # 2. THE WATCHER
  file-canary:
    image: ghcr.io/andreicscs/honeywire-filecanary:latest
    container_name: hw-file-canary
    restart: unless-stopped
    
    # Ensures uniform communication with a locally hosted Hub
    network_mode: "host"
    
    depends_on:
      permission-fixer:
        condition: service_completed_successfully

    # --- SECURITY SANDBOX ---
    user: "65532:65532"
    read_only: true
    cap_drop: ["ALL"]
    security_opt: ["no-new-privileges:true"]
    # ------------------------

    environment:
      - HW_HUB_ENDPOINT=__HUB_ENDPOINT__
      - HW_HUB_KEY=__HUB_KEY__
      - HW_SENSOR_ID=\${HW_SENSOR_ID:-file-canary-01}
      - HW_TEST_MODE=false
      - HW_HONEY_DIR=/honey_dir
      - HW_SEVERITY=\${HW_SEVERITY:-critical}

    volumes:
      # LONG SYNTAX + READ ONLY
      - type: bind
        source: \${TRAP_PATH}
        target: /honey_dir
        read_only: true`,
        readme: `
            <p>The File Canary acts as both a Honeypot and a File Integrity Monitor (FIM). It watches a specified directory or file on the host machine. If an attacker modifies, deletes, or drops a file into the watched area, the sensor immediately fires an alert to the HoneyWire Hub.</p>
            
            <h3>Features</h3>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Zero-Setup SDK Integration:</strong> Natively built on the HoneyWire Go SDK.</li>
                <li><strong>Dual-Mode Operation:</strong> Can monitor highly sensitive, real system files (FIM) or act as a standalone honeypot directory (Trap).</li>
                <li><strong>Safe Permissions Handling:</strong> Uses Access Control Lists (ACLs) to securely read target directories without altering their original host ownership.</li>
                <li><strong>Failsafe Mounts:</strong> Designed to halt deployment if the target directory doesn't exist, preventing false-positive monitoring.</li>
            </ul>

            <h3>Configuration</h3>
            <p class="mb-2">Configuration is managed through an <code>.env</code> file located in the same directory as the <code>docker-compose.yml</code>.</p>
            
            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Core Ecosystem Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Example</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">HW_HUB_ENDPOINT</td><td class="p-3">The URL of your central HoneyWire Hub.</td><td class="p-3 mono text-xs">http://127.0.0.1:8080</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_HUB_KEY</td><td class="p-3">The shared secret API key to authenticate with the Hub.</td><td class="p-3 mono text-xs">super_secret_key_123</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SENSOR_ID</td><td class="p-3">A unique identifier for this specific trap.</td><td class="p-3 mono text-xs">file-canary-01</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SEVERITY</td><td class="p-3">Alert severity sent to the Hub.</td><td class="p-3 mono text-xs">critical</td></tr>
                    </tbody>
                </table>
            </div>

            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Sensor-Specific Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Default</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">TRAP_PATH</td><td class="p-3">The physical path on the host machine to monitor.</td><td class="p-3 mono text-xs">./trap_directory</td></tr>
                    </tbody>
                </table>
            </div>

            <h3>Security Architecture</h3>
            <p>This sensor is architected for extreme resilience against exploits by utilizing a minimal attack surface and enforcing strict container sandboxing, ensuring the host filesystem remains protected.</p>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Unprivileged Execution:</strong> Runs entirely as a non-root user (<code>UID 65532</code>), preventing system-level modifications even in the event of a container breach.</li>
                <li><strong>Read-Only Mounts:</strong> The target directory is mounted with strict <code>read_only: true</code> flags, ensuring the container cannot write to or modify the host files.</li>
                <li><strong>ACL Integration:</strong> Instead of changing host file ownership, a temporary initialization container uses <code>setfacl</code> to grant the non-root user specific, read-only traverse rights, keeping your original host permissions completely intact.</li>
                <li><strong>Kernel Capability Stripping:</strong> Drops all default Linux kernel capabilities (<code>cap_drop: ALL</code>) via the Docker Compose configuration, neutralizing advanced kernel exploitation techniques.</li>
                <li><strong>Distroless Isolation:</strong> Built on a statically-linked Distroless image. It completely lacks a shell (<code>/bin/sh</code>), package managers, or standard Linux utilities, leaving attackers with zero tools to pivot to the host.</li>
            </ul>
        `
    },
    {
        id: 'icmp-canary',
        name: 'ICMP Canary (Ping)',
        osi: 'L3 Network',
        shortDesc: 'A network tripwire. Listens for ICMP Echo Requests directed at isolated IPs or unused subnets.',
        icon: 'M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z', 
        compose: `services:
  icmp-canary:
    image: ghcr.io/andreicscs/honeywire-icmpcanary:latest
    container_name: hw-icmp-canary
    restart: unless-stopped
    
    # Preserves the real Source IP of the attacker.
    network_mode: "host"
    # Root user is required by the Linux kernel to utilize NET_RAW for ICMP
    user: "0:0"

    # --- SECURITY SANDBOX ---
    read_only: true
    cap_drop: ["ALL"]
    cap_add: ["NET_RAW"]
    security_opt: ["no-new-privileges:true"]
    # ------------------------
    
    environment:
      - HW_HUB_ENDPOINT=__HUB_ENDPOINT__
      - HW_HUB_KEY=__HUB_KEY__
      - HW_SENSOR_ID=\${HW_SENSOR_ID:-ping-canary-01}
      - HW_SEVERITY=\${HW_SEVERITY:-high}
      - HW_TEST_MODE=false`,
        readme: `
            <p>The ICMP Canary (Ping Canary) is a simple, highly effective network tripwire. It listens for ICMP Echo Requests (pings) directed at the host machine. It is best deployed on isolated IPs, darknets, or unused subnets where any inbound ICMP traffic is inherently suspicious.</p>
            
            <h3>Features</h3>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Zero-Setup SDK Integration:</strong> Natively built on the HoneyWire Go SDK.</li>
                <li><strong>Raw Socket Listening:</strong> Uses pure Go to listen directly for protocol 1 (ICMP) packets without external C-dependencies.</li>
                <li><strong>Low Overhead:</strong> Requires minimal CPU and RAM to operate, making it ideal for widespread deployment.</li>
                <li><strong>Distroless Container:</strong> Compiled as a statically-linked binary running inside a minimal Docker image.</li>
            </ul>

            <h3>Configuration</h3>
            <p class="mb-2">Configuration is managed through Environment Variables.</p>
            
            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Core Ecosystem Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Example</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">HW_HUB_ENDPOINT</td><td class="p-3">The URL of your central HoneyWire Hub.</td><td class="p-3 mono text-xs">http://127.0.0.1:8080</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_HUB_KEY</td><td class="p-3">The shared secret API key to authenticate with the Hub.</td><td class="p-3 mono text-xs">super_secret_key_123</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SENSOR_ID</td><td class="p-3">A unique identifier for this specific trap.</td><td class="p-3 mono text-xs">ping-canary-01</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SEVERITY</td><td class="p-3">Alert severity sent to the Hub.</td><td class="p-3 mono text-xs">high</td></tr>
                    </tbody>
                </table>
            </div>

            <h3>Security Architecture</h3>
            <p>This sensor is architected for extreme resilience against exploits by utilizing a minimal attack surface and enforcing strict container sandboxing to safely intercept raw ICMP traffic.</p>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Raw Socket Isolation:</strong> Bypasses heavy NIDS frameworks by interacting directly with network packets in pure Go, eliminating external C-library vulnerabilities.</li>
                <li><strong>Least Privilege Execution:</strong> Runs as container root strictly to bind the raw socket, relying on container boundaries to limit system access.</li>
                <li><strong>Kernel Capability Stripping:</strong> Drops all default Linux kernel capabilities (<code>cap_drop: ALL</code>) and only adds back <code>NET_RAW</code>, ensuring the sensor can intercept pings but cannot modify the host filesystem or OS.</li>
                <li><strong>Distroless Isolation:</strong> Built on a statically-linked Distroless image. It completely lacks a shell (<code>/bin/sh</code>), package managers, or standard Linux utilities, leaving attackers with zero tools to execute secondary payloads.</li>
                <li><strong>In-Memory Operation:</strong> Processes all packet data exclusively in memory, ensuring zero malicious disk I/O operations occur on the host system.</li>
            </ul>
        `
    },
    {
        id: 'scan-detector',
        name: 'Network Scan Detector',
        osi: 'L4 Transport',
        shortDesc: 'Low-overhead network sensor to silently detect horizontal port scans via raw SYN packets.',
        icon: 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z M10 7v3m0 0v3m0-3h3m-3 0H7', 
        compose: `services:
  scan-detector:
    image: ghcr.io/andreicscs/honeywire-networkscandetector:latest
    container_name: hw-scan-detector
    restart: unless-stopped
    
    # Preserves the real Source IP of the attacker.
    network_mode: "host"
    # Root user is required by the kernel to utilize NET_RAW
    user: "0:0"

    # --- SECURITY SANDBOX ---
    read_only: true
    cap_drop: ["ALL"]
    cap_add: ["NET_RAW"]
    security_opt: ["no-new-privileges:true"]
    # ------------------------
    
    environment:
      - HW_HUB_ENDPOINT=__HUB_ENDPOINT__
      - HW_HUB_KEY=__HUB_KEY__
      - HW_SENSOR_ID=\${HW_SENSOR_ID:-scan-detector-01}
      - HW_SEVERITY=\${HW_SEVERITY:-critical}
      - HW_TEST_MODE=false

      # Scan Detector Configuration
      - HW_SCAN_THRESHOLD=5
      - HW_SCAN_WINDOW=5
      - HW_IGNORE_PORTS=80,443`,
        readme: `
            <p>The Network Scan Detector is a low-overhead network sensor designed to silently detect horizontal port scans. By monitoring raw SYN packets directly on the network interface, it identifies scanning activity aimed at closed or unused ports before it ever reaches a firewall or application log.</p>
            
            <h3>Features</h3>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Zero-Setup SDK Integration:</strong> Natively built on the HoneyWire Go SDK.</li>
                <li><strong>In-Memory Parsing:</strong> Analyzes raw TCP headers directly in memory.</li>
                <li><strong>Configurable Thresholds:</strong> Easily adjust how many unique ports must be hit within a specific time window to trigger an alert.</li>
                <li><strong>Distroless Container:</strong> Compiled as a statically-linked binary running inside a minimal Docker image.</li>
            </ul>

            <h3>Configuration</h3>
            <p class="mb-2">All configuration is handled via Environment Variables.</p>
            
            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Core Ecosystem Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Example</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">HW_HUB_ENDPOINT</td><td class="p-3">The URL of your central HoneyWire Hub.</td><td class="p-3 mono text-xs">http://127.0.0.1:8080</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_HUB_KEY</td><td class="p-3">The shared secret API key to authenticate with the Hub.</td><td class="p-3 mono text-xs">super_secret_key_123</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SENSOR_ID</td><td class="p-3">A unique identifier for this specific trap.</td><td class="p-3 mono text-xs">scan-detector-01</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SEVERITY</td><td class="p-3">Alert severity sent to the Hub.</td><td class="p-3 mono text-xs">critical</td></tr>
                    </tbody>
                </table>
            </div>

            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Sensor-Specific Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Default</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">HW_SCAN_THRESHOLD</td><td class="p-3">Number of unique ports that must be hit to trigger an alert.</td><td class="p-3 mono text-xs">5</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SCAN_WINDOW</td><td class="p-3">The time window (in seconds) to track the threshold.</td><td class="p-3 mono text-xs">5</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_IGNORE_PORTS</td><td class="p-3">Comma-separated ports to ignore (e.g., actual open services).</td><td class="p-3 mono text-xs">80,443</td></tr>
                    </tbody>
                </table>
            </div>
            
            <h3>Security Architecture</h3>
            <p>This sensor is architected for extreme resilience against exploits by utilizing a minimal attack surface and enforcing strict container sandboxing to safely handle raw network traffic.</p>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Raw Socket Isolation:</strong> Bypasses heavy NIDS frameworks by interacting directly with network packets in pure Go, eliminating external C-library vulnerabilities.</li>
                <li><strong>Least Privilege Execution:</strong> Runs as container root strictly to bind the raw socket, but relies on capability dropping to prevent privilege escalation.</li>
                <li><strong>Kernel Capability Stripping:</strong> Drops all default Linux kernel capabilities (<code>cap_drop: ALL</code>) and only adds back <code>NET_RAW</code>, ensuring the sensor can read packets but cannot modify the system.</li>
                <li><strong>Distroless Isolation:</strong> Built on a statically-linked Distroless image. It completely lacks a shell (<code>/bin/sh</code>), package managers, or standard Linux utilities, leaving attackers with zero tools to pivot to the host network.</li>
                <li><strong>In-Memory Operation:</strong> Processes all payload data exclusively in memory, ensuring zero malicious disk I/O operations occur on the host system.</li>
            </ul>
        `
    },
    {
        id: 'tcp-tarpit',
        name: 'TCP Tarpit',
        osi: 'L4 Transport',
        shortDesc: 'Binds to decoy ports and intentionally stalls attackers to waste their time while extracting payloads.',
        icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z', 
        compose: `services:
  tcp-tarpit:
    image: ghcr.io/andreicscs/honeywire-tcptarpit:latest
    container_name: hw-tcp-tarpit
    restart: unless-stopped
    
    # Preserves the real Source IP of the attacker.
    network_mode: "host"
    # Required to bind to low ports
    user: "0:0" 
    
    # --- SECURITY Root Sandbox ---
    cap_drop: ["ALL"]
    cap_add: ["NET_BIND_SERVICE"]
    read_only: true
    security_opt: ["no-new-privileges:true"]
    # -----------------------------
    
    environment:
      - HW_HUB_ENDPOINT=__HUB_ENDPOINT__
      - HW_HUB_KEY=__HUB_KEY__
      - HW_SENSOR_ID=\${HW_SENSOR_ID:-tcp-tarpit-01}
      - HW_SEVERITY=\${HW_SEVERITY:-high}
      - HW_TEST_MODE=false
      
      # Tarpit Configuration
      - HW_DECOY_PORTS=2222,3306
      - HW_TARPIT_MODE=hold
      - HW_TARPIT_BANNER=SSH-2.0-OpenSSH_8.9p1 Ubuntu-3ubuntu0.4\\r\\n`,
        readme: `
            <p>The TCP Tarpit is a high-fidelity, low-interaction honeypot designed to detect network reconnaissance and brute-force attempts. It can act as a "Tarpit," binding to decoy ports and intentionally stalling attackers to waste their time while silently extracting their IP and payload data to report to the HoneyWire Hub, or instantly close the connection and report the IP to the Hub.</p>

            <h3>Features</h3>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Zero-Setup SDK Integration:</strong> Natively built on the HoneyWire Go SDK.</li>
                <li><strong>Massive Concurrency:</strong> Powered by Go routines and channels, capable of trapping thousands of automated bots simultaneously with microscopic memory overhead.</li>
                <li><strong>Tarpit Modes:</strong> Supports <code>hold</code> (silent stall), <code>echo</code> (repeat data back), or <code>close</code> (immediate drop).</li>
                <li><strong>Forensic Capture:</strong> Safely buffers up to 10 lines of payload data without risking memory exhaustion.</li>
                <li><strong>Distroless Container:</strong> Compiled as a statically-linked binary running inside a hardened, unprivileged <code>:nonroot</code> Distroless Docker image to prevent container breakouts.</li>
            </ul>

            <h3>Configuration</h3>
            <p class="mb-2">All configuration is handled via Environment Variables.</p>
            
            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Core Ecosystem Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Example</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">HW_HUB_ENDPOINT</td><td class="p-3">The URL of your central HoneyWire Hub.</td><td class="p-3 mono text-xs">http://127.0.0.1:8080</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_HUB_KEY</td><td class="p-3">The shared secret API key to authenticate with the Hub.</td><td class="p-3 mono text-xs">super_secret_key_123</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SENSOR_ID</td><td class="p-3">A unique identifier for this specific trap.</td><td class="p-3 mono text-xs">ssh-tarpit-01</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SEVERITY</td><td class="p-3">Alert severity sent to the Hub.</td><td class="p-3 mono text-xs">high</td></tr>
                    </tbody>
                </table>
            </div>

            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Sensor-Specific Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Default</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">HW_DECOY_PORTS</td><td class="p-3">Comma-separated list of TCP ports to monitor.</td><td class="p-3 mono text-xs">2222,3306</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_TARPIT_MODE</td><td class="p-3">The behavior of the trap: <code>hold</code>, <code>echo</code>, or <code>close</code>.</td><td class="p-3 mono text-xs">hold</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_TARPIT_BANNER</td><td class="p-3">(Optional) A fake service banner to send on connect.</td><td class="p-3 mono text-xs">SSH-2.0-OpenSSH_8.2p1\\r\\n</td></tr>
                    </tbody>
                </table>
            </div>

            <h3>Tarpit Modes Explained</h3>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong><code>hold</code> (Default):</strong> The sensor accepts the connection but sends nothing. It holds the TCP socket open as long as possible (up to 1 hour), dripping empty bytes to drain the attacker's resources and slow down automated scanners like Nmap or brute-force tools.</li>
                <li><strong><code>echo</code>:</strong> Acts as an echo server, repeating whatever the attacker sends back to them. Useful for confusing automated scripts.</li>
                <li><strong><code>close</code>:</strong> Logs the connection, captures the initial payload, and forcefully closes the socket.</li>
            </ul>

            <h3>Security Architecture</h3>
            <p>This sensor is architected for extreme resilience against exploitation by adhering to the principle of least privilege and enforcing strict resource limits.</p>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Kernel Capability Stripping:</strong> Drops all Linux kernel capabilities (<code>cap_drop: ALL</code>) via the Docker Compose configuration, neutralizing advanced kernel exploitation techniques.</li>
                <li><strong>Distroless Isolation:</strong> Built on a statically-linked Distroless image. It completely lacks a shell (<code>/bin/sh</code>), package managers, or common Linux utilities (like <code>curl</code> or <code>wget</code>), leaving attackers with zero tools to pivot if they achieve Remote Code Execution.</li>
                <li><strong>Concurrency Capping:</strong> Utilizes a native Go buffered channel (semaphore) to strictly cap concurrent connections at <code>1000</code>. This prevents attackers from launching a Denial of Service (DoS) attack designed to exhaust the host machine's File Descriptors or RAM.</li>
                <li><strong>In-Memory Operation:</strong> Processes all payload data exclusively in memory, ensuring zero malicious disk I/O operations occur on the host system.</li>
            </ul>
        `
    },
    {
        id: 'web-decoy',
        name: 'Web Router Decoy',
        osi: 'L7 Application',
        shortDesc: 'Serves a deceptive router login page. Captures IP, user agent, and attempted credentials.',
        icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01', 
        compose: `services:
  web-decoy:
    image: ghcr.io/andreicscs/honeywire-webrouterdecoy:latest
    container_name: hw-web-decoy
    restart: unless-stopped
    
    # Preserves the real Source IP of the attacker.
    network_mode: "host"
    # Required to bind to low ports
    user: "0:0"

    # --- SECURITY SANDBOX ---
    read_only: true
    cap_drop: ["ALL"]
    security_opt: ["no-new-privileges:true"]
    # ------------------------

    environment:
      - HW_HUB_ENDPOINT=__HUB_ENDPOINT__
      - HW_HUB_KEY=__HUB_KEY__
      - HW_SENSOR_ID=\${HW_SENSOR_ID:-web-decoy-01}
      - HW_SEVERITY=\${HW_SEVERITY:-critical}
      - HW_TEST_MODE=false
      
      # Web Honeypot Configuration
      - HW_BIND_PORT=8081
      - HW_ROUTER_BRAND=Cisco`,
        readme: `
            <p>The Web Router Decoy is a web honeypot designed to detect credential stuffing, automated web scanners, and targeted administrative panel attacks. It serves a deceptive, router login page. When an attacker attempts to log in, the sensor captures their IP, user agent, and attempted credentials, silently reports them to the HoneyWire Hub, and safely returns a "401 Unauthorized" response to keep them guessing.</p>
            
            <h3>Features</h3>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Zero-Setup SDK Integration:</strong> Natively built on the HoneyWire Go SDK.</li>
                <li><strong>Dynamic Brand Variable:</strong> Automatically injects the specified router brand (e.g., Cisco, Netgear, ASUS) directly into the HTML template to make the trap more convincing.</li>
                <li><strong>Distroless Container:</strong> Compiled as a statically-linked binary running inside a hardened, unprivileged <code>:nonroot</code> Distroless Docker image to prevent container breakouts.</li>
            </ul>

            <h3>Configuration</h3>
            <p class="mb-2">All configuration is handled via Environment Variables.</p>
            
            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Core Ecosystem Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Example</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">HW_HUB_ENDPOINT</td><td class="p-3">The URL of your central HoneyWire Hub.</td><td class="p-3 mono text-xs">http://127.0.0.1:8080</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_HUB_KEY</td><td class="p-3">The shared secret API key to authenticate with the Hub.</td><td class="p-3 mono text-xs">super_secret_key_123</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SENSOR_ID</td><td class="p-3">A unique identifier for this specific trap.</td><td class="p-3 mono text-xs">web-decoy-01</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_SEVERITY</td><td class="p-3">Alert severity sent to the Hub.</td><td class="p-3 mono text-xs">critical</td></tr>
                    </tbody>
                </table>
            </div>

            <h4 class="font-bold text-slate-700 dark:text-zinc-300 mt-4 mb-2">Sensor-Specific Variables</h4>
            <div class="overflow-x-auto mb-6 border border-slate-200 dark:border-zinc-800 rounded-lg">
                <table class="w-full text-left text-sm">
                    <thead class="bg-slate-50 dark:bg-[#121215] text-slate-500 dark:text-zinc-400">
                        <tr><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Variable</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Description</th><th class="p-3 border-b border-slate-200 dark:border-zinc-800">Default</th></tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                        <tr><td class="p-3 mono text-xs">HW_BIND_PORT</td><td class="p-3">The TCP port the fake web server will listen on.</td><td class="p-3 mono text-xs">8080</td></tr>
                        <tr><td class="p-3 mono text-xs">HW_ROUTER_BRAND</td><td class="p-3">The brand name injected into the fake login page.</td><td class="p-3 mono text-xs">Netgear</td></tr>
                    </tbody>
                </table>
            </div>

            <h3>Security Architecture</h3>
            <p>This sensor is architected for extreme resilience against web-based exploits by utilizing a minimal attack surface and enforcing strict container sandboxing.</p>
            <ul class="list-disc pl-5 mb-6 space-y-1">
                <li><strong>Framework-Free Execution:</strong> Built purely on Go's native <code>net/http</code> library, eliminating the massive attack surface and supply-chain risks associated with heavy third-party web frameworks (like FastAPI, Flask, or Express).</li>
                <li><strong>Unprivileged Execution:</strong> Runs entirely as a non-root user (<code>UID 65532</code>), preventing system-level modifications even in the event of a container breach.</li>
                <li><strong>Kernel Capability Stripping:</strong> Drops all Linux kernel capabilities (<code>cap_drop: ALL</code>) via the Docker Compose configuration, neutralizing advanced kernel exploitation techniques.</li>
                <li><strong>Distroless Isolation:</strong> Built on a statically-linked Distroless image. It completely lacks a shell (<code>/bin/sh</code>), package managers, or standard Linux utilities (like <code>curl</code> or <code>wget</code>), leaving attackers with zero tools to download secondary payloads or pivot to the host network.</li>
                <li><strong>In-Memory Operation:</strong> Processes all payload data exclusively in memory, ensuring zero malicious disk I/O operations occur on the host system.</li>
            </ul>
        `
    }
]
</script>

<template>
    <div class="h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6">
        
        <div class="mb-6 shrink-0 mt-4 sm:mt-6">
            <h1 class="text-2xl font-bold text-slate-900 dark:text-white">Sensor Store</h1>
            <p class="text-sm text-slate-500 dark:text-zinc-400 mt-1 max-w-3xl">Deploy new HoneyWire nodes across your infrastructure. Click on a sensor to view documentation and deployment configurations.</p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pb-10">
            <div v-for="s in sensors" :key="s.id" 
                 @click="openSensor(s)"
                 class="bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800/80 rounded-lg p-5 shadow-sm hover:border-blue-500 dark:hover:border-zinc-300/20 hover:shadow-md cursor-pointer transition-all group flex flex-col">
                
                <div class="flex justify-between items-start mb-4">
                    <div class="w-12 h-12 rounded-md bg-slate-50 dark:bg-[#151518] border border-slate-200 dark:border-zinc-800/80 text-blue-600 dark:text-zinc-300 flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform duration-300">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="s.icon"></path></svg>
                    </div>
                    <span class="px-2 py-1 rounded text-[10px] font-bold uppercase tracking-wider bg-slate-100 dark:bg-zinc-800 text-slate-500 dark:text-zinc-400 border border-slate-200 dark:border-zinc-700">
                        {{ s.osi }}
                    </span>
                </div>
                
                <h3 class="text-base font-bold text-slate-900 dark:text-zinc-100 mb-1">{{ s.name }}</h3>
                <p class="text-xs text-slate-500 dark:text-zinc-400 leading-relaxed line-clamp-2">{{ s.shortDesc }}</p>
            </div>
        </div>

        <Teleport to="body">
            <transition enter-active-class="transition duration-200 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-150 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="selectedSensor" class="fixed inset-0 z-50 flex justify-center items-center p-4 sm:p-6 bg-slate-900/60 dark:bg-black/60 backdrop-blur-sm" @click.self="closeSensor">
                    
                    <div class="bg-white dark:bg-[#0a0a0c] w-full max-w-4xl h-full max-h-[85vh] rounded-lg shadow-2xl flex flex-col overflow-hidden border border-slate-200 dark:border-zinc-800/80 transform transition-all">
                        
                        <div class="px-6 py-5 border-b border-slate-100 dark:border-zinc-800/80 flex justify-between items-start bg-slate-50/50 dark:bg-[#0c0c0e] shrink-0">
                            <div class="flex items-center gap-4">
                                <div class="w-12 h-12 rounded-md bg-white dark:bg-[#151518] border border-slate-200 dark:border-zinc-800/80 text-blue-600 dark:text-zinc-300 flex items-center justify-center shrink-0 shadow-sm">
                                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="selectedSensor.icon"></path></svg>
                                </div>
                                <div>
                                    <div class="flex items-center gap-3">
                                        <h2 class="text-xl font-bold text-slate-900 dark:text-zinc-100">{{ selectedSensor.name }}</h2>
                                        <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-slate-200 dark:bg-zinc-800 text-slate-600 dark:text-zinc-400 border border-slate-300 dark:border-zinc-700 hidden sm:block">
                                            {{ selectedSensor.osi }}
                                        </span>
                                    </div>
                                    <p class="text-sm text-slate-500 dark:text-zinc-400 mt-0.5">{{ selectedSensor.shortDesc }}</p>
                                </div>
                            </div>
                            <button @click="closeSensor" class="p-2 -mr-2 text-slate-400 hover:text-slate-600 dark:text-zinc-500 dark:hover:text-zinc-300 transition-colors rounded-full hover:bg-slate-100 dark:hover:bg-zinc-800/50">
                                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path></svg>
                            </button>
                        </div>

                        <div class="flex border-b border-slate-200 dark:border-zinc-800/80 px-6 shrink-0 bg-white dark:bg-[#0a0a0c]">
                            <button @click="activeTab = 'readme'" 
                                    class="py-3 px-2 mr-6 text-xs font-bold uppercase tracking-wider border-b-2 transition-colors focus:outline-none"
                                    :class="activeTab === 'readme' ? 'border-blue-500 text-blue-600 dark:border-zinc-300 dark:text-zinc-300' : 'border-transparent text-slate-500 dark:text-zinc-500 hover:text-slate-700 dark:hover:text-zinc-300'">
                                Overview
                            </button>
                            <button @click="activeTab = 'compose'" 
                                    class="py-3 px-2 text-xs font-bold uppercase tracking-wider border-b-2 transition-colors focus:outline-none"
                                    :class="activeTab === 'compose' ? 'border-blue-500 text-blue-600 dark:border-zinc-300 dark:text-zinc-300' : 'border-transparent text-slate-500 dark:text-zinc-500 hover:text-slate-700 dark:hover:text-zinc-300'">
                                Deployment Script
                            </button>
                        </div>

                        <div class="flex-1 overflow-y-auto custom-scroll bg-white dark:bg-[#0a0a0c]">
                            
                            <div v-show="activeTab === 'readme'" class="p-6 md:p-8 readme-container text-slate-700 dark:text-zinc-300 text-sm">
                                <div v-html="selectedSensor.readme"></div>
                            </div>

                            <div v-show="activeTab === 'compose'" class="p-6 md:p-8 relative h-full flex flex-col">
                                <div class="mb-4">
                                    <p class="text-sm text-slate-600 dark:text-zinc-400">Review and modify the configuration below. Once ready, save it as <code>docker-compose.yml</code> on your target server and deploy using <code class="bg-slate-100 dark:bg-zinc-800 px-1 py-0.5 rounded-md text-blue-600 dark:text-slate-300">docker compose up -d</code>.</p>
                                </div>
                                <div class="relative flex-1 min-h-[350px]">
                                    <textarea v-model="editableCompose"
                                              spellcheck="false"
                                              class="absolute inset-0 w-full h-full bg-slate-50 dark:bg-[#121215] text-slate-800 dark:text-zinc-300 p-5 rounded-md text-[13px] mono custom-scroll border border-slate-200 dark:border-zinc-800/80 leading-relaxed shadow-inner resize-none focus:outline-none focus:ring-1 focus:ring-blue-500/50 dark:focus:ring-zinc-500/50"
                                    ></textarea>
                                    <button id="copy-btn" @click="copyToClipboard"
                                            class="absolute top-4 right-6 px-3 py-1.5 rounded-md bg-white dark:bg-[#1f1f22] hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:border-blue-500/50 text-slate-600 dark:text-zinc-300 text-[11px] font-bold uppercase tracking-wider transition-colors border border-slate-200 dark:border-zinc-700 shadow-sm active:scale-95 z-10">
                                        Copy
                                    </button>
                                </div>
                            </div>

                        </div>
                    </div>
                </div>
            </transition>
        </Teleport>

    </div>
</template>

<style scoped>
.readme-container :deep(h3) {
    font-size: 1.1rem;
    font-weight: 700;
    color: #0f172a;
    margin-top: 1.5rem;
    margin-bottom: 0.75rem;
}
.dark .readme-container :deep(h3) {
    color: #f4f4f5;
}
.readme-container :deep(h4) {
    font-size: 0.95rem;
    font-weight: 700;
    margin-top: 1.5rem;
    margin-bottom: 0.5rem;
}
.readme-container :deep(p) {
    line-height: 1.6;
    margin-bottom: 1rem;
}
.readme-container :deep(code) {
    font-family: 'JetBrains Mono', monospace;
    background-color: #f1f5f9;
    color: #0f172a;
    padding: 0.1rem 0.3rem;
    border-radius: 0.25rem;
    font-size: 0.9em;
}
.dark .readme-container :deep(code) {
    background-color: #27272a;
    color: #e4e4e7;
}
</style>