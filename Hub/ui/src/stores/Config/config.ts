import { defineStore } from 'pinia'
import { ref, readonly } from 'vue'
import { api } from '../../api/client'

// Represents the exact JSON schema returned by GET /api/v1/config
export interface ConfigApiResponse {
  hubEndpoint: string
  registryUrl: string
  autoArchiveDays: number
  autoPurgeDays: number
  webhookType: string
  webhookUrl: string
  webhookEvents: string[]
  siemAddress: string
  siemProtocol: 'tcp' | 'udp' | ''
}

// Represents the internal reactive state used by Vue components
export interface ConfigState {
  isLoaded: boolean
  hubEndpoint: string
  registryUrl: string
  autoArchiveDays: number
  autoPurgeDays: number
  webhookType: string
  webhookUrl: string
  webhookEvents: string[]
  siemAddress: string
  siemProtocol: 'tcp' | 'udp' | ''
}

export const useConfigStore = defineStore('config', () => {
  const state = ref<ConfigState>({
    isLoaded: false,
    hubEndpoint: '',
    registryUrl: '',
    autoArchiveDays: 0,
    autoPurgeDays: 0,
    webhookType: 'ntfy',
    webhookUrl: '',
    webhookEvents: [],
    siemAddress: '',
    siemProtocol: 'tcp'
  })

  const fetchConfig = async (): Promise<void> => {
    try {
      const res = await api.get('/api/v1/config')
      const data = (await res.json()) as ConfigApiResponse
      
      state.value.hubEndpoint = data.hubEndpoint || window.location.origin
      state.value.registryUrl = data.registryUrl || ''
      state.value.autoArchiveDays = data.autoArchiveDays != null ? data.autoArchiveDays : 0
      state.value.autoPurgeDays = data.autoPurgeDays != null ? data.autoPurgeDays : 0
      state.value.webhookType = data.webhookType || 'ntfy'
      state.value.webhookUrl = data.webhookUrl || ''
      
      if (data.webhookEvents && data.webhookEvents.length > 0) {
        state.value.webhookEvents = data.webhookEvents
      } else {
        state.value.webhookEvents = ['critical', 'high', 'medium', 'low', 'info']
      }
      
      state.value.siemAddress = data.siemAddress || ''
      state.value.siemProtocol = data.siemProtocol || 'tcp'
      
      state.value.isLoaded = true
    } catch (error) {
      console.error('Failed to load config', error)
    }
  }

  const patchConfig = async (updates: Partial<Omit<ConfigState, 'isLoaded'>>): Promise<boolean> => {
    try {
      await api.patch('/api/v1/config', updates)

      Object.assign(state.value, updates)
      return true
    } catch (error) {
      console.error('Config patch network error:', error)
      return false
    }
  }

  return {
    config: readonly(state), 
    fetchConfig,
    patchConfig
  }
})