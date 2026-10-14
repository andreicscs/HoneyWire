import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import NodeDetails from '../NodeDetails.vue'
import NodeStatWidgets from '../../components/nodedetails/NodeStatWidgets.vue'
import NodeCatalogGrid from '../../components/nodedetails/NodeCatalogGrid.vue'
import NodeSensorModal from '../../components/nodedetails/NodeSensorModal.vue'
import { useFleetStore } from '../../stores/Fleet/fleet'
import { useEventsStore } from '../../stores/Events/events'

vi.mock('../../api/client', () => {
  const now = new Date().toISOString()
  const nodePayload = {
    nodeId: 'node-1',
    alias: 'Test Node',
    status: 'up',
    hasPendingConfig: false,
    lastHeartbeat: now,
    publicIp: '1.2.3.4',
    privateIp: '10.0.0.1',
    tags: ['lab'],
    apiKey: 'test-api-key',
    activeRevision: 'r1',
    desiredRevision: 'r1',
    hasUpdateAvailable: false,
    lastEvent: 'Never',
    installedSensors: [
      {
        sensorId: 'hw-sensor-demo',
        customName: 'Demo Sensor',
        status: 'up',
        isSilenced: false,
        envVars: {},
        deployedVersion: '1.0.0',
        updateAvailable: false,
        lastHeartbeat: now,
      },
    ],
  }
  const eventPayload = {
    id: 'evt-1',
    timestamp: now,
    nodeId: 'node-1',
    sensorId: 'hw-sensor-demo',
    severity: 'high',
    eventTrigger: 'Port scan detected',
    source: '9.9.9.9',
    target: '10.0.0.1',
    isRead: false,
    isArchived: false,
  }
  const payloadFor = (url: string) => {
    if (url === '/api/v2/nodes') return [nodePayload]
    if (url.startsWith('/api/v2/nodes/')) return nodePayload
    if (url === '/api/v2/manifests') return []
    if (url.startsWith('/api/v2/events/unread')) return { count: 0 }
    if (url.startsWith('/api/v2/events/summary')) return { timeframe: '24H', totalEvents: 1, bySensor: {} }
    if (url.startsWith('/api/v2/events')) return [eventPayload]
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
  { path: '/fleet', name: 'fleet', component: { template: '<div />' } },
  { path: '/dashboard', name: 'dashboard', component: { template: '<div />' } },
  { path: '/fleet/node/:id', name: 'node-details', component: NodeDetails },
]

async function mountAt(path: string, seedEvents = false) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({ history: createMemoryHistory(), routes })
  router.push(path)
  await router.isReady()

  if (seedEvents) {
    await useEventsStore().fetchEvents()
  }

  const wrapper = mount(NodeDetails, {
    attachTo: document.body,
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  await flushPromises()
  return { wrapper, router, fleetStore: useFleetStore(), eventsStore: useEventsStore() }
}

describe('NodeDetails view (regression)', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    document.body.style.overflow = ''
    vi.clearAllMocks()
  })

  it('renders the page and back button', async () => {
    const { wrapper } = await mountAt('/fleet/node/node-1')
    expect(wrapper.find('[data-testid="node-details-page"]').exists()).toBe(true)
    const back = wrapper.find('[data-testid="back-to-fleet"]')
    expect(back.exists()).toBe(true)
    expect(back.text()).toContain('Back to Fleet')
  })

  it('renders the node card with header, sensors and catalog', async () => {
    const { wrapper } = await mountAt('/fleet/node/node-1')
    expect(wrapper.find('[data-testid="node-card"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Test Node')
    expect(wrapper.text()).toContain('Deployed Sensors')
    expect(wrapper.text()).toContain('Demo Sensor')
    expect(wrapper.text()).toContain('Sensor Catalog')
  })

  it('shows an empty catalog when no manifests are registered', async () => {
    const { wrapper } = await mountAt('/fleet/node/node-1')
    expect(wrapper.text()).toContain('No Sensors Available')
  })

  it('shows relative last-event time when events exist', async () => {
    const { wrapper } = await mountAt('/fleet/node/node-1', true)
    expect(wrapper.text()).toContain('Last Event:')
    expect(wrapper.text()).toContain('ago')
  })

  it('navigates back to fleet and clears selection', async () => {
    const { wrapper, router, fleetStore } = await mountAt('/fleet/node/node-1')
    expect(fleetStore.selectedNodeId).toBe('node-1')
    await wrapper.find('[data-testid="back-to-fleet"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/fleet')
    expect(fleetStore.selectedNodeId).toBeNull()
  })

  it('navigates to dashboard on view-all-events', async () => {
    const { wrapper, router } = await mountAt('/fleet/node/node-1')
    await wrapper.findComponent(NodeStatWidgets).vm.$emit('view-all-events')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/dashboard')
  })

  it('opens and closes the sensor modal from the catalog', async () => {
    const { wrapper } = await mountAt('/fleet/node/node-1')
    const manifest = { id: 'hw-sensor-new', name: 'New Sensor', deployment: { env_vars: [] } }
    await wrapper.findComponent(NodeCatalogGrid).vm.$emit('open', manifest)
    expect(wrapper.findComponent(NodeSensorModal).props('show')).toBe(true)
    expect(document.body.style.overflow).toBe('hidden')
    await wrapper.findComponent(NodeSensorModal).vm.$emit('close')
    expect(wrapper.findComponent(NodeSensorModal).props('show')).toBe(false)
    expect(document.body.style.overflow).toBe('')
  })

  it('renders no node card for an unknown node id', async () => {
    const { wrapper } = await mountAt('/fleet/node/unknown')
    expect(wrapper.find('[data-testid="back-to-fleet"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="node-card"]').exists()).toBe(false)
  })
})
