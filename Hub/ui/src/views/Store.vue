<script setup>
import { ref, computed, onMounted } from 'vue'
import { useConfig } from '../api/useConfig'

const { config } = useConfig()

const selectedSensor = ref(null)
const activeTab = ref('readme')
const envVarValues = ref({}) 
const activeEnvVar = ref(null) 

const sensors = ref([])
const isLoading = ref(true)
const fetchError = ref(false) // Replaces offline fallback flag

// Vite will inject VITE_MANIFEST_URL if it exists in a .env file.
// If it doesn't exist (like in production), it safely falls back to GitHub.
const REGISTRY_URL = import.meta.env.VITE_MANIFEST_URL || "https://raw.githubusercontent.com/andreicscs/HoneyWire/main/Sensors/official/manifests.json"

onMounted(async () => {
    try {
        const response = await fetch(REGISTRY_URL, { cache: "no-store" })
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`)
        sensors.value = await response.json()
    } catch (error) {
        console.error("HoneyWire: Failed to fetch sensor registry.", error)
        fetchError.value = true
    } finally {
        isLoading.value = false
    }
})

// Intelligently extracts a clean default value from Go Templates for the UI
const getUIDefault = (def) => {
    if (!def || !def.includes('{{')) return def;
    // 1. Try to extract the final fallback from an {{ else }} block
    const elseMatch = def.match(/\{\{\s*else\s*\}\}(.*?)\{\{\s*end\s*\}\}/);
    if (elseMatch) return elseMatch[1].trim();
    
    // 2. Try to extract simple function defaults (e.g., {{ availablePort 8080 }})
    const funcMatch = def.match(/\{\{\s*[a-zA-Z]+\s+([0-9]+)\s*\}\}/);
    if (funcMatch) return funcMatch[1].trim();
    
    return '';
}

const openSensor = (sensor) => {
    selectedSensor.value = sensor
    activeTab.value = 'readme'
    envVarValues.value = {}
    sensor.deployment.env_vars?.forEach(env => {
        envVarValues.value[env.name] = getUIDefault(env.default)
    })
    document.body.style.overflow = 'hidden'
}

const closeSensor = () => {
    selectedSensor.value = null
    envVarValues.value = {}
    activeEnvVar.value = null
    document.body.style.overflow = ''
}

const rawCompose = computed(() => {
    if (!selectedSensor.value) return ''
    return generateYaml(selectedSensor.value, envVarValues.value, false)
})

const highlightedCompose = computed(() => {
    if (!selectedSensor.value) return ''
    return generateYaml(selectedSensor.value, envVarValues.value, true)
})

const resolveTemplateValue = (text, envValues) => {
    if (!text) return ''
    return text.replace(/{{\s*\.([A-Za-z0-9_]+)\s*}}/g, (_, key) => {
        const toEnv = (name) => {
            const snake = name.replace(/([A-Z])/g, '_$1').toUpperCase().replace(/^_/, '')
            if (envValues[snake] !== undefined) return snake
            if (envValues[`HW_${snake}`] !== undefined) return `HW_${snake}`
            return null
        }
        const envKey = toEnv(key)
        return envKey && envValues[envKey] !== undefined ? envValues[envKey] : getUIDefault(text)
    })
}


const generateYaml = (sensor, envValues, isHtml) => {
    // 1. Pull the reactive config state locally
    const endpoint = config.hubEndpoint || window.location.origin
    const key = config.hubKey || '<YOUR_HW_HUB_KEY>'

    let yaml = `services:\n`

    if (sensor.deployment.init_containers && sensor.deployment.init_containers.length > 0) {
        sensor.deployment.init_containers.forEach(init => {
            yaml += `  ${init.name}:\n`
            yaml += `    image: ${init.image}\n`
            if (init.command) yaml += `    command: ${init.command}\n`
            if (init.volume_mounts && init.volume_mounts.length > 0) {
                yaml += `    volumes:\n`
                init.volume_mounts.forEach(v => {
                    const sourcePath = envValues['TRAP_PATH'] || getUIDefault(v.source)
                    yaml += `      - type: ${v.type}\n        source: ${sourcePath}\n        target: ${v.target}\n`
                    if (v.read_only) yaml += `        read_only: true\n`
                })
            }
        })
    }

    yaml += `  ${sensor.id}:\n`
    yaml += `    image: ${sensor.deployment.image}\n`
    yaml += `    container_name: hw-${sensor.id}\n`
    yaml += `    restart: unless-stopped\n`
    yaml += `    network_mode: "${sensor.deployment.network_mode}"\n`
    if (sensor.deployment.user) yaml += `    user: "${sensor.deployment.user}"\n`

    yaml += `    cap_drop: ["ALL"]\n`
    if (sensor.deployment.cap_add && sensor.deployment.cap_add.length > 0) {
        yaml += `    cap_add: [${sensor.deployment.cap_add.map(c => `"${c}"`).join(', ')}]\n`
    }
    yaml += `    security_opt: ["no-new-privileges:true"]\n`

    if (sensor.deployment.init_containers && sensor.deployment.init_containers.length > 0) {
        yaml += `    depends_on:\n`
        sensor.deployment.init_containers.forEach(init => {
            yaml += `      ${init.name}:\n        condition: service_completed_successfully\n`
        })
    }

    if (sensor.deployment.volume_mounts && sensor.deployment.volume_mounts.length > 0) {
        yaml += `    volumes:\n`
        sensor.deployment.volume_mounts.forEach(v => {
            const sourcePath = envValues['TRAP_PATH'] || getUIDefault(v.source)
            yaml += `      - type: ${v.type}\n        source: ${sourcePath}\n        target: ${v.target}\n`
            if (v.read_only) yaml += `        read_only: true\n`
        })
    }

    if (sensor.deployment.env_vars && sensor.deployment.env_vars.length > 0) {
        yaml += `    environment:\n`
        sensor.deployment.env_vars.forEach(env => {
            let value = envValues[env.name] !== undefined ? envValues[env.name] : getUIDefault(env.default)
            
            // --- THE FIX: Auto-inject config for hidden core variables ---
            if (env.name === 'HW_HUB_ENDPOINT') value = endpoint
            if (env.name === 'HW_HUB_KEY') value = key
            // -------------------------------------------------------------

            let line = `      - ${env.name}=${value}\n`
            
            if (isHtml && activeEnvVar.value === env.name && value) {
                const escapedValue = value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
                const regex = new RegExp(`(${escapedValue})`, 'g')
                line = line.replace(regex, `<span class="bg-blue-100 text-blue-800 dark:bg-zinc-700/80 dark:text-white font-bold px-1 rounded transition-colors duration-200">$1</span>`)
            }
            yaml += line
        })
        
        if (isHtml && activeEnvVar.value === 'TRAP_PATH' && envValues['TRAP_PATH']) {
             const value = envValues['TRAP_PATH'];
             const escapedValue = value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
             const regex = new RegExp(`(?<!>)${escapedValue}(?!<)`, 'g')
             yaml = yaml.replace(regex, `<span class="bg-blue-100 text-blue-800 dark:bg-zinc-700/80 dark:text-white font-bold px-1 rounded transition-colors duration-200">${value}</span>`)
        }
    }

    return yaml
}

const copyToClipboard = () => {
    if (!selectedSensor.value) return
    navigator.clipboard.writeText(rawCompose.value)

    const btn = document.getElementById('copy-btn')
    const originalText = btn.innerHTML
    
    // Improved "Copied!" styling
    btn.innerHTML = 'Copied!'
    btn.classList.add('bg-green-100', 'text-green-700', 'border-green-300', 'dark:bg-green-900/30', 'dark:text-green-400', 'dark:border-green-800/50')
    btn.classList.remove('text-slate-600', 'dark:text-zinc-300', 'bg-white', 'dark:bg-[#1f1f22]')
    
    setTimeout(() => { 
        btn.innerHTML = originalText 
        btn.classList.remove('bg-green-100', 'text-green-700', 'border-green-300', 'dark:bg-green-900/30', 'dark:text-green-400', 'dark:border-green-800/50')
        btn.classList.add('text-slate-600', 'dark:text-zinc-300', 'bg-white', 'dark:bg-[#1f1f22]')
    }, 2000)
}
</script>

<template>
    <div class="h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6">
        
        <div class="mb-6 shrink-0 mt-4 sm:mt-6 flex justify-between items-end">
            <div>
                <h1 class="text-2xl font-bold text-slate-900 dark:text-white">Sensor Store</h1>
                <p class="text-sm text-slate-500 dark:text-zinc-400 mt-1 max-w-3xl">Deploy new HoneyWire nodes across your infrastructure. Click on a sensor to view documentation and deployment configurations.</p>
            </div>
            <div v-if="isOfflineFallback" class="hidden sm:flex items-center gap-2 text-xs font-medium text-amber-600 bg-amber-50 dark:bg-amber-900/20 px-3 py-1 rounded-full border border-amber-200 dark:border-amber-800/30">
                <span class="w-2 h-2 rounded-full bg-amber-500 animate-pulse"></span> Offline Mode
            </div>
        </div>

        <div v-if="isLoading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pb-10">
            <div v-for="i in 4" :key="i" class="bg-white dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800/50 rounded-lg p-5 h-36 animate-pulse flex flex-col justify-between">
                <div class="flex justify-between items-start">
                    <div class="w-12 h-12 rounded-md bg-slate-200 dark:bg-zinc-800"></div>
                    <div class="w-20 h-5 rounded bg-slate-200 dark:bg-zinc-800"></div>
                </div>
                <div class="space-y-2 mt-4">
                    <div class="h-4 bg-slate-200 dark:bg-zinc-800 rounded w-3/4"></div>
                    <div class="h-3 bg-slate-200 dark:bg-zinc-800 rounded w-full"></div>
                </div>
            </div>
        </div>
        
        <!-- Error State -->
        <div v-else-if="fetchError" class="flex flex-col items-center justify-center py-20 text-center">
            <svg class="w-12 h-12 text-slate-400 dark:text-zinc-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
            <h3 class="text-lg font-bold text-slate-900 dark:text-white">Unable to reach Sensor Registry</h3>
            <p class="text-sm text-slate-500 dark:text-zinc-400 mt-2 max-w-md">Please ensure this Hub has internet access to pull the latest sensor manifests from GitHub.</p>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pb-10">
            <div v-for="s in sensors" :key="s.id" 
                 @click="openSensor(s)"
                 class="bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800/80 rounded-lg p-5 shadow-sm hover:border-blue-500 dark:hover:border-zinc-300/20 hover:shadow-md cursor-pointer transition-all group flex flex-col">
                
                <div class="flex justify-between items-start mb-4">
                    <div class="w-12 h-12 rounded-md bg-slate-50 dark:bg-[#151518] border border-slate-200 dark:border-zinc-800/80 text-blue-600 dark:text-zinc-300 flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform duration-300">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="s.icon_svg"></path></svg>
                    </div>
                    <span class="px-2 py-1 rounded text-[10px] font-bold uppercase tracking-wider bg-slate-100 dark:bg-zinc-800 text-slate-500 dark:text-zinc-400 border border-slate-200 dark:border-zinc-700">
                        {{ s.osi_layer }}
                    </span>
                </div>
                
                <h3 class="text-base font-bold text-slate-900 dark:text-zinc-100 mb-1">{{ s.name }}</h3>
                <p class="text-xs text-slate-500 dark:text-zinc-400 leading-relaxed line-clamp-2">{{ s.description }}</p>
            </div>
        </div>

        <Teleport to="body">
            <transition enter-active-class="transition duration-200 ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-150 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="selectedSensor" class="fixed inset-0 z-50 flex justify-center items-center p-4 sm:p-6 bg-slate-900/60 dark:bg-black/60 backdrop-blur-sm" @click.self="closeSensor">
                    
                    <div class="bg-white dark:bg-[#0a0a0c] w-full max-w-4xl h-full max-h-[85vh] rounded-lg shadow-2xl flex flex-col overflow-hidden border border-slate-200 dark:border-zinc-800/80 transform transition-all">
                        
                        <div class="px-6 py-5 border-b border-slate-100 dark:border-zinc-800/80 flex justify-between items-start bg-slate-50/50 dark:bg-[#0c0c0e] shrink-0">
                            <div class="flex items-center gap-4">
                                <div class="w-12 h-12 rounded-md bg-white dark:bg-[#151518] border border-slate-200 dark:border-zinc-800/80 text-blue-600 dark:text-zinc-300 flex items-center justify-center shrink-0 shadow-sm">
                                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="selectedSensor.icon_svg"></path></svg>
                                </div>
                                <div>
                                    <div class="flex items-center gap-3">
                                        <h2 class="text-xl font-bold text-slate-900 dark:text-zinc-100">{{ selectedSensor.name }}</h2>
                                        <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-slate-200 dark:bg-zinc-800 text-slate-600 dark:text-zinc-400 border border-slate-300 dark:border-zinc-700 hidden sm:block">
                                            {{ selectedSensor.osi_layer }}
                                        </span>
                                    </div>
                                    <p class="text-sm text-slate-500 dark:text-zinc-400 mt-0.5">{{ selectedSensor.description }}</p>
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
                                <p class="mb-6">{{ selectedSensor.documentation.summary }}</p>
                                
                                <div v-for="section in selectedSensor.documentation.sections" :key="section.title" class="mb-6">
                                    <h3 class="text-lg font-bold text-slate-900 dark:text-zinc-100 mb-3">{{ section.title }}</h3>
                                    <ul v-if="section.type === 'list'" class="list-disc pl-5 space-y-1">
                                        <li v-for="item in section.content" :key="item">{{ item }}</li>
                                    </ul>
                                </div>
                            </div>

                            <div v-show="activeTab === 'compose'" class="p-6 md:p-8 relative h-full flex flex-col">
                                <div class="mb-4">
                                    <p class="text-sm text-slate-600 dark:text-zinc-400">Configure the sensor deployment below. Once ready, save it as <code>docker-compose.yml</code> on your target server and deploy using <code class="bg-slate-100 dark:bg-zinc-800 px-1 py-0.5 rounded-md text-blue-600 dark:text-slate-300">docker compose up -d</code>.</p>
                                </div>
                                
                                <!-- Config Forms with FIXED Text Colors -->
                                <div v-if="selectedSensor.deployment.env_vars && selectedSensor.deployment.env_vars.filter(env => !env.hidden).length > 0" class="mb-6">
                                    <h4 class="text-sm font-bold text-slate-900 dark:text-zinc-100 mb-3">Configuration</h4>
                                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                        <div v-for="env in selectedSensor.deployment.env_vars.filter(env => !env.hidden)" :key="env.name" class="space-y-1">
                                            <label class="block text-xs font-medium text-slate-700 dark:text-zinc-300">{{ env.name }}</label>
                                            <input 
                                                v-model="envVarValues[env.name]"
                                                :type="env.type === 'int' ? 'number' : 'text'"
                                                :placeholder="getUIDefault(env.default)"
                                                @focus="activeEnvVar = env.name"
                                                @blur="activeEnvVar = null"
                                                class="w-full px-3 py-2 text-sm text-slate-900 dark:text-zinc-100 bg-white dark:bg-zinc-900 border border-slate-300 dark:border-zinc-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-zinc-500 transition-colors"
                                            />
                                            <p class="text-xs text-slate-500 dark:text-zinc-400">{{ env.description }}</p>
                                        </div>
                                    </div>
                                </div>
                                
                                <!-- YAML Output using PRE for Highlights instead of Textarea -->
                                <div class="relative flex-1 min-h-[350px]">
                                    <pre 
                                        v-html="highlightedCompose"
                                        class="absolute inset-0 w-full h-full bg-slate-50 dark:bg-[#121215] text-slate-800 dark:text-zinc-300 p-5 rounded-md text-[13px] mono custom-scroll border border-slate-200 dark:border-zinc-800/80 leading-relaxed overflow-auto focus:outline-none"
                                    ></pre>
                                    <button id="copy-btn" @click="copyToClipboard"
                                            class="absolute top-4 right-6 px-3 py-1.5 rounded-md bg-white dark:bg-[#1f1f22] hover:bg-blue-50 hover:text-blue-600 dark:hover:text-slate-300 dark:hover:bg-slate-500/30 dark:hover:border-slate-300/50 text-slate-600 dark:text-zinc-300 text-[11px] font-bold uppercase tracking-wider transition-colors border border-slate-200 dark:border-zinc-700 shadow-sm active:scale-95 z-10">
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