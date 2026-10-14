import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import TrafficFilters from '../TrafficFilters.vue'
import { useFleetStore } from '../../../stores/Fleet/fleet'
import { api } from '../../../api/client'

enableAutoUnmount(afterEach)

vi.mock('../../../api/client', () => {
  const now = new Date().toISOString()
  const nodes = [
    {
      nodeId: 'node-1', alias: 'Node One', status: 'up', hasPendingConfig: false,
      lastHeartbeat: now, publicIp: null, privateIp: null, tags: [], apiKey: 'k1',
      activeRevision: 'r1', desiredRevision: 'r1', hasUpdateAvailable: false, lastEvent: 'Never',
      installedSensors: [{ sensorId: 's1', customName: 'S One', status: 'up', isSilenced: false, envVars: {} }],
    },
    {
      nodeId: 'node-2', alias: 'Node Two', status: 'down', hasPendingConfig: false,
      lastHeartbeat: null, publicIp: null, privateIp: null, tags: [], apiKey: 'k2',
      activeRevision: 'r1', desiredRevision: 'r1', hasUpdateAvailable: false, lastEvent: 'Never',
      installedSensors: [{ sensorId: 's2', customName: 'S Two', status: 'down', isSilenced: false, envVars: {} }],
    },
  ]
  const respond = async (data: any = {}) => ({ ok: true, json: async () => data, text: async () => '' })
  return {
    api: {
      get: vi.fn(async (url: string) => {
        if (url === '/api/v2/nodes') return respond(nodes)
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

const originalScrollTo = Element.prototype.scrollTo

async function mountFilters() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const fleetStore = useFleetStore()
  await fleetStore.fetchFleet()
  const wrapper = mount(TrafficFilters, {
    attachTo: document.body,
    global: { plugins: [pinia] },
  })
  await flushPromises()
  return { wrapper, fleetStore }
}

function closeMenus() {
  document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

describe('TrafficFilters', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    Element.prototype.scrollTo = vi.fn() as any
    vi.clearAllMocks()
  })

  afterEach(() => {
    closeMenus()
    document.body.innerHTML = ''
    Element.prototype.scrollTo = originalScrollTo
  })

  it('renders all-traffic and node pills', async () => {
    const { wrapper } = await mountFilters()
    expect(wrapper.text()).toContain('All Traffic')
    expect(wrapper.text()).toContain('Node One')
    expect(wrapper.text()).toContain('Node Two')
  })

  it('selects a node on pill click', async () => {
    const { wrapper, fleetStore } = await mountFilters()
    await wrapper.find('#pill-node-1 [data-testid="node-pill-select"]').trigger('click')
    expect(fleetStore.selectedNodeId).toBe('node-1')
  })

  it('clears selection on All Traffic click', async () => {
    const { wrapper, fleetStore } = await mountFilters()
    fleetStore.selectTarget('node-1', null, false)
    await wrapper.find('#pill-all').trigger('click')
    expect(fleetStore.selectedNodeId).toBeNull()
  })

  it('silences a node from the menu', async () => {
    const { wrapper } = await mountFilters()
    await wrapper.findAll('[data-testid="meatball-trigger"]')[0].trigger('click')
    document
      .body.querySelector('[data-testid="filter-menu-silence"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(api.patch).toHaveBeenCalledWith(
      expect.stringContaining('/silence'),
      expect.objectContaining({ isSilenced: true })
    )
  })

  it('cancels node forget when confirm is dismissed', async () => {
    vi.stubGlobal('confirm', vi.fn(() => false))
    const { wrapper } = await mountFilters()
    await wrapper.findAll('[data-testid="meatball-trigger"]')[0].trigger('click')
    document
      .body.querySelector('[data-testid="filter-menu-forget"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(api.delete).not.toHaveBeenCalled()
    vi.unstubAllGlobals()
  })
})
