import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import FleetManagement from '../FleetManagement.vue'
import FleetNodeWidget from '../../components/fleetmanagement/FleetNodeWidget.vue'
import FleetDeployModal from '../../components/fleetmanagement/FleetDeployModal.vue'
import { useFleetStore } from '../../stores/Fleet/fleet'
import { api } from '../../api/client'

const mockNodes = vi.hoisted(() => ({ list: [] as any[] }))

vi.mock('../../api/client', () => {
  const payloadFor = (url: string) => {
    if (url === '/api/v2/nodes') return mockNodes.list
    if (url === '/api/v2/manifests') return []
    if (url.startsWith('/api/v2/uptime')) return { timeframe: '24H', summary: {}, groups: [] }
    return {}
  }
  const respond = async (url: string) => ({
    ok: true,
    json: async () => payloadFor(url),
    text: async () => '',
  })
  return {
    api: {
      get: vi.fn(respond),
      post: vi.fn(respond),
      put: vi.fn(respond),
      patch: vi.fn(respond),
      delete: vi.fn(respond),
      request: vi.fn(respond),
    },
  }
})

const routes = [
  { path: '/fleet', name: 'fleet', component: FleetManagement },
  { path: '/fleet/node/:id', name: 'node-details', component: { template: '<div />' } },
]

async function mountFleet() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({ history: createMemoryHistory(), routes })
  router.push('/fleet')
  await router.isReady()
  const wrapper = mount(FleetManagement, {
    attachTo: document.body,
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  await flushPromises()
  return { wrapper, router, fleetStore: useFleetStore() }
}

describe('FleetManagement view (regression)', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    localStorage.clear()
    vi.clearAllMocks()
    mockNodes.list = [
      {
        nodeId: 'node-1',
        alias: 'Fleet Node One',
        status: 'up',
        hasPendingConfig: false,
        lastHeartbeat: new Date().toISOString(),
        publicIp: '1.2.3.4',
        privateIp: '10.0.0.1',
        tags: [],
        apiKey: 'key-1',
        activeRevision: 'r1',
        desiredRevision: 'r1',
        hasUpdateAvailable: false,
        lastEvent: 'Never',
        installedSensors: [],
      },
    ]
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders the page, header and deploy button', async () => {
    const { wrapper } = await mountFleet()
    expect(wrapper.find('[data-testid="fleet-page"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Fleet Overview')
    const deploy = wrapper.find('[data-testid="deploy-node-button"]')
    expect(deploy.exists()).toBe(true)
    expect(deploy.text()).toContain('Deploy New Node')
  })

  it('renders a widget per deployed node', async () => {
    const { wrapper } = await mountFleet()
    expect(wrapper.find('[data-testid="fleet-grid"]').exists()).toBe(true)
    expect(wrapper.findComponent(FleetNodeWidget).exists()).toBe(true)
    expect(wrapper.text()).toContain('Fleet Node One')
  })

  it('shows the empty state when no nodes are deployed', async () => {
    mockNodes.list = []
    const { wrapper } = await mountFleet()
    expect(wrapper.find('[data-testid="fleet-empty"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('No nodes deployed.')
  })

  it('opens the deploy modal from the deploy button', async () => {
    const { wrapper } = await mountFleet()
    expect(wrapper.findComponent(FleetDeployModal).props('show')).toBe(false)
    await wrapper.find('[data-testid="deploy-node-button"]').trigger('click')
    expect(wrapper.findComponent(FleetDeployModal).props('show')).toBe(true)
  })

  it('opens node detail on widget openDetail', async () => {
    const { wrapper, router, fleetStore } = await mountFleet()
    await wrapper.findComponent(FleetNodeWidget).vm.$emit('openDetail')
    await flushPromises()
    expect(fleetStore.selectedNodeId).toBe('node-1')
    expect(router.currentRoute.value.name).toBe('node-details')
    expect(router.currentRoute.value.params.id).toBe('node-1')
  })

  it('cancels node deletion when confirm is dismissed', async () => {
    vi.stubGlobal('confirm', vi.fn(() => false))
    const { wrapper } = await mountFleet()
    await wrapper.findComponent(FleetNodeWidget).vm.$emit('delete')
    await flushPromises()
    expect(api.delete).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Fleet Node One')
  })

  it('deletes the node when confirm is accepted', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    const { wrapper } = await mountFleet()
    await wrapper.findComponent(FleetNodeWidget).vm.$emit('delete')
    await flushPromises()
    expect(api.delete).toHaveBeenCalledWith('/api/v2/nodes/node-1')
  })
})
