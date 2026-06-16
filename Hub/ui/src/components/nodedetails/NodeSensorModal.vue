<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import escapeHtml from 'escape-html'
import BaseButton from '../ui/forms/BaseButton.vue'
import BaseInput from '../ui/forms/BaseInput.vue'
import { useClipboard } from '../../utils/useClipboard'
import { useFleetStore } from '../../stores/Fleet/fleet'
import { useConfigStore } from '../../stores/Config/config'

const props = defineProps<{ show: boolean, sensor: any, isEditing: boolean, initialEnvVars: Record<string, any>, apiKey: string | null }>()
const emit = defineEmits<{ (e: 'close'): void, (e: 'apply', payload: Record<string, string>): void }>()

const fleetStore = useFleetStore()
const configStore = useConfigStore()
const { copiedStates, handleCopy } = useClipboard() as any

const activeTab = ref('readme')
const envVarValues = ref<Record<string, any>>({})
const activeEnvVar = ref<string | null>(null)
const isSeverityOpen = ref(false)
const rawCompose = ref('')
const highlightedCompose = ref('')
const composePre = ref<HTMLElement | null>(null)

watch(() => props.show, (shown) => {
    if (shown && props.sensor) {
        activeTab.value = props.isEditing ? 'config' : 'readme'
        envVarValues.value = { ...props.initialEnvVars }
    } else {
        envVarValues.value = {}
        rawCompose.value = ''
        highlightedCompose.value = ''
        activeEnvVar.value = null
    }
}, { immediate: true })

watch([activeTab, () => props.sensor], async ([tab, sensor]) => {
    if (tab === 'config' && sensor) {
        await nextTick()
        const inputs = document.querySelectorAll<HTMLInputElement>('input.config-input')
        if (inputs.length > 0) inputs[0].focus()
    }
})

const severityOptions = [
  { value: 'info', label: 'Info', textClass: 'text-info', hoverClass: 'hover:bg-info/10 hover:text-info' },
  { value: 'low', label: 'Low', textClass: 'text-low', hoverClass: 'hover:bg-low/10 hover:text-low' },
  { value: 'medium', label: 'Medium', textClass: 'text-medium', hoverClass: 'hover:bg-medium/10 hover:text-medium' },
  { value: 'high', label: 'High', textClass: 'text-high', hoverClass: 'hover:bg-high/10 hover:text-high' },
  { value: 'critical', label: 'Critical', textClass: 'text-critical', hoverClass: 'hover:bg-critical/10 hover:text-critical' }
]

const currentSeverity = computed(() => severityOptions.find(s => s.value === envVarValues.value['HW_SEVERITY']) || severityOptions[3])
const toggleSeverity = () => { isSeverityOpen.value = !isSeverityOpen.value; activeEnvVar.value = isSeverityOpen.value ? 'HW_SEVERITY' : null }
const closeSeverity = () => { isSeverityOpen.value = false; activeEnvVar.value = null }
const selectSeverity = (val: string) => { envVarValues.value['HW_SEVERITY'] = val; closeSeverity() }

const getUIDefault = (def: any) => {
  if (def === undefined || def === null) return ''
  const strDef = String(def)
  if (!strDef.includes('{{')) return strDef
  const elseMatch = strDef.match(/\{\{\s*else\s*\}\}(.*?)\{\{\s*end\s*\}\}/)
  if (elseMatch) return elseMatch[1].trim()
  const funcMatch = strDef.match(/\{\{\s*[a-zA-Z]+\s+([0-9]+)\s*\}\}/)
  if (funcMatch) return funcMatch[1].trim()
  return ''
}

const getEnvType = (env: any) => {
  if (env.type === 'boolean' || env.type === 'bool') return 'boolean'
  if (env.type === 'int' || env.type === 'number') return 'number'
  const def = getUIDefault(env.default).trim()
  if (def === 'true' || def === 'false') return 'boolean'
  if (def !== '' && !isNaN(Number(def))) return 'number'
  return 'text'
}

const coreVars = ['HW_HUB_ENDPOINT', 'HW_HUB_KEY', 'HW_SENSOR_ID', 'HW_SEVERITY', 'HW_TEST_MODE', 'HW_LOG_LEVEL']
const sortedEnvVars = computed(() => {
  if (!props.sensor?.deployment?.env_vars) return []
  return [...props.sensor.deployment.env_vars].filter(env => !env.hidden).sort((a, b) => {
      const aIsCore = coreVars.includes(a.name), bIsCore = coreVars.includes(b.name)
      if (aIsCore && !bIsCore) return -1
      if (!aIsCore && bIsCore) return 1
      if (aIsCore && bIsCore) return coreVars.indexOf(a.name) - coreVars.indexOf(b.name)
      return a.name.localeCompare(b.name)
  })
})

const fetchYamlFromHub = async () => {
  if (!props.sensor) return
  const safeEnvValues = Object.fromEntries(Object.entries(envVarValues.value).map(([k, v]) => [k, v !== undefined && v !== null ? String(v) : '']))
  try {
    rawCompose.value = await fleetStore.generateCompose({
      hubEndpoint: configStore.config.hubEndpoint || window.location.origin,
      hubKey: props.apiKey || '<YOUR_HW_NODE_KEY>',
      sensors: [{ sensorId: props.sensor.id, envValues: safeEnvValues, manifest: props.sensor }]
    })
    applyHighlighting()
  } catch (e: any) {
    rawCompose.value = `# ERROR GENERATING PREVIEW:\n# ${(e.response?.data || e.message || String(e)).trim().split('\n').join('\n# ')}`
    highlightedCompose.value = `<span class="text-danger-main font-semibold">${escapeHtml(rawCompose.value)}</span>`
  }
}

const applyHighlighting = () => {
  let htmlYaml = escapeHtml(rawCompose.value)
  if (activeEnvVar.value) htmlYaml = htmlYaml.replace(new RegExp(`^.*\\b${activeEnvVar.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b.*$`, 'gm'), `<span class="bg-highlight-bg text-highlight-text ring-1 ring-highlight-ring px-1 rounded transition-colors duration-[var(--duration-fast)] active-highlight">$&</span>`)
  highlightedCompose.value = htmlYaml
  nextTick(() => {
    if (composePre.value) {
      const highlightEl = composePre.value.querySelector('.active-highlight') as HTMLElement | null
      if (highlightEl) composePre.value.scrollTo({ top: Math.max(0, Number(highlightEl.offsetTop) - Number(composePre.value.clientHeight / 2) + Number(highlightEl.clientHeight / 2)), behavior: 'smooth' })
    }
  })
}

watch(envVarValues, () => fetchYamlFromHub(), { deep: true })
watch(activeEnvVar, () => applyHighlighting())
</script>

<template>
    <Teleport to="body">
        <transition enter-active-class="transition duration-normal ease-out" enter-from-class="opacity-0" enter-to-class="opacity-100" leave-active-class="transition duration-[var(--duration-fast)] ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
            <div v-if="sensor && show" class="fixed inset-0 z-[var(--z-modal)] flex justify-center items-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm" @mousedown.self="$emit('close')">
                <div class="bg-bg-base w-full max-w-3xl h-full max-h-[85vh] rounded-lg shadow-2xl flex flex-col overflow-hidden border border-border-default transform transition-all">
                    <div class="px-6 py-5 border-b border-border-default flex justify-between items-start bg-bg-surface shrink-0">
                        <div class="flex items-center gap-4"><div class="w-12 h-12 rounded-md bg-bg-inset border border-border-default/50 text-text-h flex items-center justify-center shrink-0 shadow-sm"><svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" :d="sensor.icon_svg"></path></svg></div><div><h2 class="text-base font-semibold text-text-h">{{ sensor.name }}</h2><p class="text-sm text-text-m mt-0.5">{{ sensor.description }}</p></div></div>
                        <button @click="$emit('close')" class="p-2 -mr-2 text-text-m hover:text-text-h transition-colors duration-[var(--duration-fast)] rounded-full hover:bg-secondary-hover focus:outline-none"><svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path></svg></button>
                    </div>
                    <div class="flex border-b border-border-default px-6 shrink-0 bg-bg-base"><button @click="activeTab = 'readme'" class="py-3 px-2 mr-6 text-sm border-b-2 transition-colors duration-[var(--duration-fast)] focus:outline-none" :class="activeTab === 'readme' ? 'border-primary-main text-text-h font-semibold' : 'border-transparent text-text-m hover:text-text-h'">Overview</button><button @click="activeTab = 'config'" class="py-3 px-2 text-sm border-b-2 transition-colors duration-[var(--duration-fast)] focus:outline-none" :class="activeTab === 'config' ? 'border-primary-main text-text-h font-semibold' : 'border-transparent text-text-m hover:text-text-h'">Configuration</button></div>
                    <div class="flex-1 overflow-y-auto custom-scroll bg-bg-base">
                        <div v-show="activeTab === 'readme'" class="p-6 md:p-8 readme-container text-sm text-text-m"><p class="mb-6 text-sm font-medium text-text-h">{{ sensor.documentation?.summary }}</p><div v-for="section in sensor.documentation?.sections" :key="section.title" class="mb-6"><h3 class="text-sm font-semibold text-text-h mb-3">{{ section.title }}</h3><ul v-if="section.type === 'list'" class="list-disc pl-5 space-y-1"><li v-for="item in section.content" :key="item">{{ item }}</li></ul></div></div>
                        <form v-show="activeTab === 'config'" @submit.prevent="$emit('apply', Object.fromEntries(Object.entries(envVarValues).map(([k, v]) => [k, v !== undefined && v !== null ? String(v) : ''])))" class="p-6 md:p-8 relative h-full flex flex-col">
                            <div class="mb-6"><h4 class="text-sm font-semibold text-text-h mb-3">Sensor Settings</h4>
                                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                                    <div class="space-y-1 relative"><label class="block text-sm text-text-h mb-0.5">Alert Severity</label><div @click="toggleSeverity" class="w-full px-3 py-2 text-sm bg-input-bg border rounded-md cursor-pointer flex justify-between items-center transition-all duration-[var(--duration-fast)] shadow-sm select-none" :class="isSeverityOpen ? 'border-primary-main ring-1 ring-focus-ring' : 'border-input-border hover:border-border-default'"><span :class="currentSeverity.textClass" class="font-medium flex items-center gap-2"><span class="w-2 h-2 rounded-full shrink-0" :class="currentSeverity.textClass.replace('text-', 'bg-')"></span>{{ currentSeverity.label }}</span><svg class="w-4 h-4 text-text-m shrink-0 transition-transform duration-200" :class="isSeverityOpen ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg></div><div v-if="isSeverityOpen" @click="closeSeverity" class="fixed inset-0 z-[var(--z-elevated)]"></div><transition enter-active-class="transition duration-100 ease-out" enter-from-class="transform scale-95 opacity-0" enter-to-class="transform scale-100 opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="transform scale-100 opacity-100" leave-to-class="transform scale-95 opacity-0"><ul v-if="isSeverityOpen" class="absolute top-full left-0 z-[var(--z-dropdown)] w-full mt-1 bg-bg-surface border border-border-default rounded-md shadow-lg py-1 overflow-hidden"><li v-for="option in severityOptions" :key="option.value" @click="selectSeverity(option.value)" class="px-3 py-2 cursor-pointer transition-colors text-sm font-medium duration-[var(--duration-fast)] flex items-center gap-2" :class="[option.textClass, option.hoverClass]"><span class="w-2 h-2 rounded-full shrink-0" :class="option.textClass.replace('text-', 'bg-')"></span>{{ option.label }}</li></ul></transition></div>
                                    <template v-for="env in sortedEnvVars" :key="env.name"><BaseInput v-if="env.name !== 'HW_SEVERITY'" v-model="envVarValues[env.name]" :type="getEnvType(env)" :label="env.name" :description="env.description" :placeholder="getUIDefault(env.default)" :defaultFallback="getUIDefault(env.default)" @focus="activeEnvVar = env.name" @blur="activeEnvVar = null" /></template>
                                </div>
                            </div>
                            <div class="relative flex-1 min-h-[250px] mb-6">
                                <button type="button" @click="handleCopy('compose-yaml', rawCompose)" class="absolute top-3 right-3 px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-[var(--duration-fast)] shadow-sm active:scale-95 z-10 focus:outline-none border" :class="copiedStates['compose-yaml'] ? 'bg-success-bg text-success-text border-success-border' : 'bg-secondary-main text-text-h border-secondary-border hover:bg-secondary-hover'">{{ copiedStates['compose-yaml'] ? 'Copied!' : 'Copy' }}</button>
                                <!-- codeql[js/xss] Data is explicitly sanitized via escape-html before injection -->
                                <!-- nosemgrep: javascript.vue.security.audit.xss.templates.avoid-v-html.avoid-v-html - Data is explicitly sanitized via escape-html before injection -->
                                <pre ref="composePre" v-html="highlightedCompose" class="absolute inset-0 w-full h-full bg-bg-surface text-text-m p-4 rounded-md text-sm font-mono custom-scroll border border-border-default leading-relaxed overflow-auto focus:outline-none scroll-smooth shadow-inner"></pre>
                            </div>
                            <div class="mt-auto border-t border-border-default pt-4 flex justify-end"><BaseButton variant="primary" type="submit" class="px-6">{{ isEditing ? 'Apply Settings' : 'Add to Node' }}</BaseButton></div>
                        </form>
                    </div>
                </div>
            </div>
        </transition>
    </Teleport>
</template>
<style scoped>
.readme-container :deep(h3) { font-size: var(--text-sm); font-weight: var(--font-weight-medium); color: var(--text-h); margin-top: 1.5rem; margin-bottom: 0.75rem; }
.readme-container :deep(p) { line-height: var(--text-leading-normal); margin-bottom: 1rem; }
.readme-container :deep(code) { font-family: var(--font-mono); background-color: var(--input-bg); color: var(--text-h); padding: 0.1rem 0.3rem; border-radius: var(--radius-sm); font-size: var(--text-sm); border: 1px solid var(--input-border); }
</style>