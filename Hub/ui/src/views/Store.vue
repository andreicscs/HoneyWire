<script setup>
import { ref, watch, computed, onMounted, nextTick } from 'vue'
import { useConfig } from '../api/useConfig'
import PageHeader from '../components/ui/PageHeader.vue' // Adjust path if needed

const { config } = useConfig()

const selectedSensor = ref(null)
const activeTab = ref('readme')
const envVarValues = ref({}) 
const activeEnvVar = ref(null) 

const sensors = ref([])
const isLoading = ref(true)
const fetchError = ref(false)

const rawCompose = ref('')
const highlightedCompose = ref('')
const baseYaml = ref('') 
const composePre = ref(null) 

onMounted(async () => {
    try {
        isLoading.value = true;
        const response = await fetch("/api/v1/manifests", { cache: "no-store" });
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        sensors.value = await response.json();
    } catch (error) {
        console.error("HoneyWire: Failed to fetch sensor registry.", error);
        fetchError.value = true;
    } finally {
        isLoading.value = false;
    }
});

const getUIDefault = (def) => {
    if (!def) return '';
    if (!def.includes('{{')) return def; 
    const elseMatch = def.match(/\{\{\s*else\s*\}\}(.*?)\{\{\s*end\s*\}\}/);
    if (elseMatch) return elseMatch[1].trim();
    const funcMatch = def.match(/\{\{\s*[a-zA-Z]+\s+([0-9]+)\s*\}\}/);
    if (funcMatch) return funcMatch[1].trim();
    return '';
}

const coreVars = ['HW_HUB_ENDPOINT', 'HW_HUB_KEY', 'HW_NODE_ID', 'HW_NODE_ALIAS', 'HW_SENSOR_ID', 'HW_SEVERITY', 'HW_TEST_MODE', 'HW_LOG_LEVEL'];

const sortedEnvVars = computed(() => {
    if (!selectedSensor.value?.deployment?.env_vars) return [];
    
    return [...selectedSensor.value.deployment.env_vars]
        .filter(env => !env.hidden)
        .sort((a, b) => {
            const aIsCore = coreVars.includes(a.name);
            const bIsCore = coreVars.includes(b.name);
            if (aIsCore && !bIsCore) return -1;
            if (!aIsCore && bIsCore) return 1;
            if (aIsCore && bIsCore) return coreVars.indexOf(a.name) - coreVars.indexOf(b.name);
            return a.name.localeCompare(b.name);
        });
});

watch(envVarValues, () => {
    applyLocalUpdates();
}, { deep: true });

watch(activeEnvVar, () => {
    applyHighlighting();
});

const openSensor = (sensor) => {
    selectedSensor.value = sensor
    activeTab.value = 'readme'
    envVarValues.value = {}
    
    sensor.deployment.env_vars?.forEach(env => {
        if (env.name === 'HW_HUB_ENDPOINT') {
            envVarValues.value[env.name] = config.hubEndpoint || window.location.origin;
        } else if (env.name === 'HW_HUB_KEY') {
            envVarValues.value[env.name] = config.hubKey || '<YOUR_HW_HUB_KEY>';
        } else {
            envVarValues.value[env.name] = getUIDefault(env.default);
        }
    })
    
    document.body.style.overflow = 'hidden'
    fetchYamlFromHub(); 
}

const closeSensor = () => {
    selectedSensor.value = null
    envVarValues.value = {}
    activeEnvVar.value = null
    document.body.style.overflow = ''
}

const fetchYamlFromHub = async () => {
    if (!selectedSensor.value) return;

    const safeEnvValues = Object.fromEntries(
        Object.entries(envVarValues.value).map(([k, v]) => [k, v !== undefined && v !== null ? String(v) : ''])
    );

    const payload = {
        node_id: "${HW_NODE_ID}",
        hub_endpoint: config.hubEndpoint || window.location.origin,
        hub_key: config.hubKey || '<YOUR_HW_HUB_KEY>',
        sensors: [{
            sensor_id: selectedSensor.value.id,
            env_values: safeEnvValues,
            manifest: selectedSensor.value
        }]
    };

    try {
        const response = await fetch('/api/v1/compose/generate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        
        if (!response.ok) throw new Error("Failed to compile YAML");
        
        const rawBackendYaml = await response.text();
        baseYaml.value = formatComposeYaml(rawBackendYaml); 
        applyLocalUpdates(); 

    } catch (e) {
        console.error("YAML Generation Error:", e);
        rawCompose.value = "services:\n  error:\n    image: error_generating_yaml";
        highlightedCompose.value = rawCompose.value;
    }
}

const formatComposeYaml = (yamlStr) => {
    let lines = yamlStr.split('\n');
    let parsedLines = [];
    let inEnvBlock = false;
    let envVars = [];
    
    for (let i = 0; i < lines.length; i++) {
        let line = lines[i];
        if (line.match(/^ {4}environment:/) || line.match(/^ {2}environment:/)) {
            inEnvBlock = true;
            parsedLines.push(line);
            continue;
        }

        if (inEnvBlock) {
            if (line.trim().startsWith('- ')) {
                envVars.push(line);
                if (i === lines.length - 1) {
                    envVars.sort((a, b) => sortEnvVarsLogic(a, b));
                    parsedLines.push(...envVars);
                }
            } else {
                envVars.sort((a, b) => sortEnvVarsLogic(a, b));
                parsedLines.push(...envVars);
                envVars = [];
                inEnvBlock = false;
                parsedLines.push(line);
            }
        } else {
            parsedLines.push(line);
        }
    }

    const finalYaml = [];
    let servicesIndex = parsedLines.findIndex(l => l.startsWith('services:'));
    
    if (servicesIndex === -1) return parsedLines.join('\n'); 

    finalYaml.push(...parsedLines.slice(0, servicesIndex + 1));
    
    let servicesMap = [];
    let currentService = null;
    let currentBlock = [];

    for (let i = servicesIndex + 1; i < parsedLines.length; i++) {
        const line = parsedLines[i];
        
        if (line.match(/^ {2}[a-zA-Z0-9_-]+:/)) {
            if (currentService) {
                servicesMap.push({ name: currentService, lines: currentBlock });
            }
            currentService = line.trim().replace(':', '');
            currentBlock = [line];
        } 
        else if (line.startsWith('    ') || line.startsWith('   ') || line === '') {
            if (currentService) currentBlock.push(line);
            else finalYaml.push(line);
        } 
        else if (!line.startsWith(' ')) {
            if (currentService) {
                servicesMap.push({ name: currentService, lines: currentBlock });
                currentService = null;
            }
            finalYaml.push(...parsedLines.slice(i));
            break;
        }
    }
    
    if (currentService) {
        servicesMap.push({ name: currentService, lines: currentBlock });
    }

    servicesMap.sort((a, b) => {
        const aIsSensor = a.name.startsWith('hw-sensor');
        const bIsSensor = b.name.startsWith('hw-sensor');
        if (aIsSensor && !bIsSensor) return 1;
        if (!aIsSensor && bIsSensor) return -1;
        return 0;
    });

    servicesMap.forEach(svc => {
        finalYaml.push(...svc.lines);
    });

    return finalYaml.join('\n');
}

const sortEnvVarsLogic = (a, b) => {
    const aName = a.trim().replace('- ', '').split(/[:=]/)[0];
    const bName = b.trim().replace('- ', '').split(/[:=]/)[0];
    const aIsCore = coreVars.includes(aName);
    const bIsCore = coreVars.includes(bName);
    if (aIsCore && !bIsCore) return -1;
    if (!aIsCore && bIsCore) return 1;
    if (aIsCore && bIsCore) return coreVars.indexOf(aName) - coreVars.indexOf(bName);
    return aName.localeCompare(bName);
}

const applyLocalUpdates = () => {
    if (!baseYaml.value) return;
    let updatedYaml = baseYaml.value;

    for (const [key, val] of Object.entries(envVarValues.value)) {
        const safeVal = val !== undefined && val !== null ? String(val) : '';
        const regex = new RegExp(`^(\\s*(?:-\\s+)?${key}\\s*[:=]\\s*["']?)([^"\'\\r\\n]*)(["']?.*)$`, 'gm');
        updatedYaml = updatedYaml.replace(regex, `$1${safeVal}$3`);
    }

    rawCompose.value = updatedYaml;
    applyHighlighting();
}

const applyHighlighting = () => {
    let htmlYaml = rawCompose.value;
    
    if (activeEnvVar.value) {
        const escapedName = activeEnvVar.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const regex = new RegExp(`^.*\\b${escapedName}\\b.*$`, 'gm');
        htmlYaml = htmlYaml.replace(regex, `<span class="bg-highlight-bg text-highlight-text ring-1 ring-highlight-ring px-1 rounded transition-colors duration-[var(--duration-fast)] active-highlight">$&</span>`);
    }
    
    highlightedCompose.value = htmlYaml;

    nextTick(() => {
        if (composePre.value) {
            const highlightEl = composePre.value.querySelector('.active-highlight');
            if (highlightEl) {
                const scrollPos = highlightEl.offsetTop - (composePre.value.clientHeight / 2) + (highlightEl.clientHeight / 2);
                composePre.value.scrollTo({ top: Math.max(0, scrollPos), behavior: 'smooth' });
            }
        }
    });
}

const copyToClipboard = () => {
    if (!selectedSensor.value) return
    navigator.clipboard.writeText(rawCompose.value)

    const btn = document.getElementById('copy-btn')
    const originalText = btn.innerHTML
    
    btn.innerHTML = 'Copied!'
    btn.classList.add('bg-success-bg', 'text-success-text', 'border-success-border')
    btn.classList.remove('bg-secondary-main', 'text-secondary-text', 'border-secondary-border', 'hover:bg-secondary-hover', 'hover:text-text-h')
    
    setTimeout(() => { 
        btn.innerHTML = originalText 
        btn.classList.remove('bg-success-bg', 'text-success-text', 'border-success-border')
        btn.classList.add('bg-secondary-main', 'text-secondary-text', 'border-secondary-border', 'hover:bg-secondary-hover', 'hover:text-text-h')
    }, 2000)
}
</script>

<template>
    <div class="h-full flex flex-col max-w-[1600px] w-full mx-auto px-2 sm:px-4 lg:px-6">
        
        <PageHeader 
            class="mb-6 mt-4 sm:mt-6 shrink-0"
            title="Sensor Store" 
            description="Deploy new HoneyWire nodes across your infrastructure. Click on a sensor to view documentation and deployment configurations."
        />

        <div v-if="isLoading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pb-10">
            <div v-for="i in 4" :key="i" class="bg-bg-surface border border-border-default rounded-lg p-5 h-36 animate-pulse flex flex-col justify-between">
                <div class="flex justify-between items-start">
                    <div class="w-12 h-12 rounded-md bg-bg-inset"></div>
                    <div class="w-20 h-5 rounded bg-bg-inset"></div>
                </div>
                <div class="space-y-2 mt-4">
                    <div class="h-4 bg-bg-inset rounded w-3/4"></div>
                    <div class="h-3 bg-bg-inset rounded w-full"></div>
                </div>
            </div>
        </div>
        
        <div v-else-if="fetchError" class="flex flex-col items-center justify-center py-20 text-center">
            <svg class="w-12 h-12 text-danger-text mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
            <h3 class="text-base font-medium text-text-h">Unable to reach Sensor Registry</h3>
            <p class="text-base text-text-m mt-2 max-w-md">Please ensure this Hub has internet access to pull the latest sensor manifests from GitHub.</p>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pb-10">
            <div v-for="s in sensors" :key="s.id" 
                 @click="openSensor(s)"
                 class="bg-bg-surface border border-border-default rounded-lg p-5 shadow-sm hover:border-primary-main hover:shadow-md cursor-pointer transition-all duration-[var(--duration-normal)] group flex flex-col">
                
                <div class="flex justify-between items-start mb-4">
                    <div class="w-12 h-12 rounded-md bg-bg-base border border-border-default/50 text-text-h flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform duration-[var(--duration-normal)]">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="s.icon_svg"></path></svg>
                    </div>
                    <span class="px-2 py-0.5 rounded text-sm bg-bg-inset text-text-m border border-border-default/50">
                        {{ s.osi_layer }}
                    </span>
                </div>
                
                <h3 class="text-base font-medium text-text-h mb-1">{{ s.name }}</h3>
                <p class="text-sm text-text-m leading-relaxed line-clamp-2">{{ s.description }}</p>
            </div>
        </div>

        <Teleport to="body">
            <transition enter-active-class="transition duration-[var(--duration-normal)] ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-[var(--duration-fast)] ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
                <div v-if="selectedSensor" class="fixed inset-0 z-50 flex justify-center items-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm" @click.self="closeSensor">
                    
                    <div class="bg-bg-base w-full max-w-4xl h-full max-h-[85vh] rounded-lg shadow-2xl flex flex-col overflow-hidden border border-border-default transform transition-all">
                        
                        <div class="px-6 py-5 border-b border-border-default flex justify-between items-start bg-bg-surface shrink-0">
                            <div class="flex items-center gap-4">
                                <div class="w-12 h-12 rounded-md bg-bg-inset border border-border-default/50 text-text-h flex items-center justify-center shrink-0 shadow-sm">
                                    <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="selectedSensor.icon_svg"></path></svg>
                                </div>
                                <div>
                                    <div class="flex items-center gap-3">
                                        <h2 class="text-[length:var(--text-h1)] font-medium text-text-h">{{ selectedSensor.name }}</h2>
                                        <span class="px-2 py-0.5 rounded text-sm bg-bg-inset text-text-m border border-border-default/50 hidden sm:block">
                                            {{ selectedSensor.osi_layer }}
                                        </span>
                                    </div>
                                    <p class="text-sm text-text-m mt-0.5">{{ selectedSensor.description }}</p>
                                </div>
                            </div>
                            <button @click="closeSensor" class="p-2 -mr-2 text-text-m hover:text-text-h transition-colors duration-[var(--duration-fast)] rounded-full hover:bg-secondary-hover focus:outline-none">
                                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path></svg>
                            </button>
                        </div>

                        <div class="flex border-b border-border-default px-6 shrink-0 bg-bg-base">
                            <button @click="activeTab = 'readme'" 
                                    class="py-3 px-2 mr-6 text-base border-b-2 transition-colors duration-[var(--duration-fast)] focus:outline-none"
                                    :class="activeTab === 'readme' ? 'border-primary-main text-text-h font-medium' : 'border-transparent text-text-m hover:text-text-h'">
                                Overview
                            </button>
                            <button @click="activeTab = 'compose'" 
                                    class="py-3 px-2 text-base border-b-2 transition-colors duration-[var(--duration-fast)] focus:outline-none"
                                    :class="activeTab === 'compose' ? 'border-primary-main text-text-h font-medium' : 'border-transparent text-text-m hover:text-text-h'">
                                Deployment Script
                            </button>
                        </div>

                        <div class="flex-1 overflow-y-auto custom-scroll bg-bg-base">
                            
                            <div v-show="activeTab === 'readme'" class="p-6 md:p-8 readme-container text-text-m text-base">
                                <p class="mb-6 text-base font-medium text-text-h">{{ selectedSensor.documentation.summary }}</p>
                                
                                <div v-for="section in selectedSensor.documentation.sections" :key="section.title" class="mb-6">
                                    <h3 class="text-base font-medium text-text-h mb-3">{{ section.title }}</h3>
                                    <ul v-if="section.type === 'list'" class="list-disc pl-5 space-y-1">
                                        <li v-for="item in section.content" :key="item">{{ item }}</li>
                                    </ul>
                                </div>
                            </div>

                            <div v-show="activeTab === 'compose'" class="p-6 md:p-8 relative h-full flex flex-col">
                                <div class="mb-4">
                                    <p class="text-base text-text-m">Configure the sensor deployment below. Once ready, save it as <code>docker-compose.yml</code> on your target server and deploy using <code class="bg-input-bg px-1.5 py-0.5 rounded-md text-text-h border border-input-border font-mono">docker compose up -d</code>.</p>
                                </div>
                                
                                <div v-if="sortedEnvVars.length > 0" class="mb-6">
                                    <h4 class="text-base font-medium text-text-h mb-3">Configuration</h4>
                                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                        <div v-for="env in sortedEnvVars" :key="env.name" class="space-y-1">
                                            <label class="block text-sm text-text-h mb-0.5">{{ env.name }}</label>
                                            <input 
                                                v-model="envVarValues[env.name]"
                                                :type="env.type === 'int' ? 'number' : 'text'"
                                                :placeholder="getUIDefault(env.default)"
                                                @focus="activeEnvVar = env.name"
                                                @blur="activeEnvVar = null"
                                                class="w-full px-3 py-2 text-base text-text-h bg-input-bg border border-input-border rounded-md focus:outline-none focus:border-primary-main focus:ring-1 focus:ring-focus-ring transition-all duration-[var(--duration-fast)] shadow-sm placeholder:text-text-m/50"
                                            />
                                            <p class="text-sm text-text-m">{{ env.description }}</p>
                                        </div>
                                    </div>
                                </div>
                                
                                <div class="relative flex-1 min-h-[350px]">
                                    <pre 
                                        ref="composePre"
                                        v-html="highlightedCompose"
                                        class="absolute inset-0 w-full h-full bg-bg-surface text-text-m p-5 rounded-md text-sm font-mono custom-scroll border border-border-default leading-relaxed overflow-auto focus:outline-none scroll-smooth shadow-inner"
                                    ></pre>
                                    
                                    <button id="copy-btn" @click="copyToClipboard"
                                            class="absolute top-4 right-6 px-3 py-1.5 rounded-md bg-secondary-main border border-secondary-border text-secondary-text hover:bg-secondary-hover hover:text-text-h text-sm font-medium transition-colors duration-[var(--duration-fast)] shadow-sm active:scale-95 z-10 focus:outline-none">
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
    font-size: var(--text-base); 
    font-weight: var(--font-weight-medium);
    color: var(--text-h);
    margin-top: 1.5rem;
    margin-bottom: 0.75rem;
}
.readme-container :deep(h4) {
    font-size: var(--text-base); 
    font-weight: var(--font-weight-medium);
    color: var(--text-h);
    margin-top: 1.5rem;
    margin-bottom: 0.5rem;
}
.readme-container :deep(p) {
    line-height: 1.6;
    margin-bottom: 1rem;
}
.readme-container :deep(code) {
    font-family: var(--font-mono);
    background-color: var(--input-bg);
    color: var(--text-h);
    padding: 0.1rem 0.3rem;
    border-radius: var(--radius-sm);
    font-size: var(--text-sm);
    border: 1px solid var(--input-border);
}
</style>