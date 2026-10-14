import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import UptimeHeatmap from '../UptimeHeatmap.vue'
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
      installedSensors: [{ sensorId: 'hw-sensor-a', customName: 'Sensor A', status: 'up', isSilenced: false, envVars: {} }],
    },
  ]
  const uptime = {
    timeframe: '24H',
    generatedAt: now,
    summary: { overallUptime: 99.5 },
    groups: [
      {
        nodeId: 'node-1', nodeAlias: 'Node One', worstStatus: 'up',
        sensors: [
          {
            sensorId: 'hw-sensor-a', displayName: 'Sensor A', status: 'up', isSilenced: false,
            blocks: [{ status: 'up', label: 'Online', timeLabel: '10:00' }],
          },
        ],
      },
    ],
  }
  const respond = async (data: any = {}) => ({ ok: true, json: async () => data, text: async () => '' })
  return {
    api: {
      get: vi.fn(async (url: string) => {
        if (url === '/api/v2/nodes') return respond(nodes)
        if (url.startsWith('/api/v2/uptime')) return respond(uptime)
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

async function mountHeatmap() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const fleetStore = useFleetStore()
  await fleetStore.fetchFleet()
  await fleetStore.fetchUptime()
  const wrapper = mount(UptimeHeatmap, {
    attachTo: document.body,
    global: { plugins: [pinia] },
  })
  await flushPromises()
  return { wrapper, fleetStore }
}

function closeMenus() {
  document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

describe('UptimeHeatmap', () => {
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

  it('renders fleet uptime percentage', async () => {
    const { wrapper } = await mountHeatmap()
    expect(wrapper.text()).toContain('Fleet Uptime')
    expect(wrapper.text()).toContain('99.50%')
    expect(wrapper.text()).toContain('Node One')
  })

  it('selects a sensor on click', async () => {
    const { wrapper, fleetStore } = await mountHeatmap()
    await wrapper.find('[data-testid="uptime-sensor"]').trigger('click')
    expect(fleetStore.selectedNodeId).toBe('node-1')
    expect(fleetStore.selectedSensorId).toBe('node-1:hw-sensor-a')
  })

  it('selects a node on group click', async () => {
    const { wrapper, fleetStore } = await mountHeatmap()
    await wrapper.find('[data-testid="uptime-group"]').trigger('click')
    expect(fleetStore.selectedNodeId).toBe('node-1')
  })

  it('silences a sensor from the menu', async () => {
    const { wrapper } = await mountHeatmap()
    await wrapper.findAll('[data-testid="meatball-trigger"]')[0].trigger('click')
    document
      .body.querySelector('[data-testid="uptime-menu-silence"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(api.patch).toHaveBeenCalledWith(
      expect.stringContaining('/silence'),
      expect.objectContaining({ isSilenced: true })
    )
  })

  it('cancels sensor forget when confirm is dismissed', async () => {
    vi.stubGlobal('confirm', vi.fn(() => false))
    const { wrapper } = await mountHeatmap()
    await wrapper.findAll('[data-testid="meatball-trigger"]')[0].trigger('click')
    document
      .body.querySelector('[data-testid="uptime-menu-forget"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(api.delete).not.toHaveBeenCalled()
    vi.unstubAllGlobals()
  })
})
