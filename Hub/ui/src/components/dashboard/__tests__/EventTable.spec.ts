import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import EventTable from '../EventTable.vue'
import { useEventsStore } from '../../../stores/Events/events'
import { api } from '../../../api/client'

enableAutoUnmount(afterEach)

vi.mock('../../../api/client', () => {
  const oldTs = new Date(Date.now() - 3600000).toISOString()
  const newTs = new Date().toISOString()
  const events = [
    {
      id: 'e-old', timestamp: oldTs, nodeId: 'node-1', sensorId: 's1',
      severity: 'critical', eventTrigger: 'port_scan', source: '9.9.9.9',
      target: '10.0.0.1', isRead: true, isArchived: false, details: { port: 22 },
    },
    {
      id: 'e-new', timestamp: newTs, nodeId: 'node-1', sensorId: 's1',
      severity: 'high', eventTrigger: 'login_attempt', source: '8.8.8.8',
      target: '10.0.0.2', isRead: false, isArchived: false, details: {},
    },
  ]
  const nodes = [
    {
      nodeId: 'node-1', alias: 'Node One', status: 'up', hasPendingConfig: false,
      lastHeartbeat: newTs, publicIp: null, privateIp: null, tags: [], apiKey: 'k1',
      activeRevision: 'r1', desiredRevision: 'r1', hasUpdateAvailable: false, lastEvent: 'Never',
      installedSensors: [],
    },
  ]
  const respond = async (data: any = {}) => ({ ok: true, json: async () => data, text: async () => '' })
  return {
    api: {
      get: vi.fn(async (url: string) => {
        if (url.startsWith('/api/v2/events/unread')) return respond({ count: 0 })
        if (url === '/api/v2/nodes') return respond(nodes)
        if (url.startsWith('/api/v2/events')) return respond(events)
        return respond()
      }),
      post: vi.fn(respond),
      put: vi.fn(respond),
      patch: vi.fn(respond),
      delete: vi.fn(respond),
      request: vi.fn(respond),
    },
  }
})

const originalScrollIntoView = Element.prototype.scrollIntoView

async function mountTable() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const eventsStore = useEventsStore()
  const { useFleetStore } = await import('../../../stores/Fleet/fleet')
  await useFleetStore().fetchFleet()
  await eventsStore.fetchEvents()
  const wrapper = mount(EventTable, {
    attachTo: document.body,
    global: { plugins: [pinia] },
  })
  await flushPromises()
  return { wrapper, eventsStore }
}

describe('EventTable', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    Element.prototype.scrollIntoView = vi.fn() as any
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
    Element.prototype.scrollIntoView = originalScrollIntoView
  })

  it('renders the threat queue with events', async () => {
    const { wrapper } = await mountTable()
    expect(wrapper.text()).toContain('Active Threat Queue')
    expect(wrapper.text()).toContain('port scan')
    expect(wrapper.text()).toContain('login attempt')
    expect(wrapper.findAll('[data-testid="event-row"]')).toHaveLength(2)
  })

  it('orders newest first by default', async () => {
    const { wrapper } = await mountTable()
    const rows = wrapper.findAll('[data-testid="event-row"]')
    expect(rows[0].attributes('data-event-id')).toBe('e-new')
    expect(rows[1].attributes('data-event-id')).toBe('e-old')
  })

  it('sorts by severity on header click', async () => {
    const { wrapper } = await mountTable()
    await wrapper.find('[aria-label="Sort by threat severity"]').trigger('click')
    const rows = wrapper.findAll('[data-testid="event-row"]')
    expect(rows[0].attributes('data-event-id')).toBe('e-old')
    expect(rows[0].text()).toContain('critical')
  })

  it('expands a row and marks it read', async () => {
    const { wrapper } = await mountTable()
    await wrapper.find('[data-event-id="e-new"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="event-details"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="event-details"]').text()).toContain('Trace ID: e-new')
    expect(api.patch).toHaveBeenCalledWith('/api/v2/events/e-new/read')
  })

  it('archives a single event after confirmation', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    const { wrapper } = await mountTable()
    const archiveBtns = wrapper.findAll('[aria-label="Archive this event"]')
    expect(archiveBtns.length).toBeGreaterThan(0)
    await archiveBtns[0].trigger('click')
    await flushPromises()
    expect(api.patch).toHaveBeenCalledWith(expect.stringContaining('/api/v2/events/e-new/archive'))
  })

  it('archives all events after confirmation', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    const { wrapper } = await mountTable()
    await wrapper.find('[aria-label="Archive all active events"]').trigger('click')
    await flushPromises()
    expect(api.patch).toHaveBeenCalledWith('/api/v2/events/archive-all')
  })

  it('hides pagination for few events', async () => {
    const { wrapper } = await mountTable()
    expect(wrapper.text()).not.toContain('Showing')
  })
})
