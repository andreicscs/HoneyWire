import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SeverityChart from '../SeverityChart.vue'
import { useEventsStore } from '../../../stores/Events/events'
import { api } from '../../../api/client'

enableAutoUnmount(afterEach)

vi.mock('chart.js/auto', () => {
  const MockChart = vi.fn(function (this: any) {
    return {
      data: { labels: [], datasets: [{ data: [0, 0, 0, 0, 0], backgroundColor: [] }] },
      options: { plugins: { tooltip: {} }, scales: { x: { ticks: {} }, y: {} } },
      update: vi.fn(),
      destroy: vi.fn(),
    }
  }) as any
  MockChart.register = vi.fn()
  return { default: MockChart }
})

vi.mock('../../../api/client', () => {
  const respond = async (data: any = {}) => ({ ok: true, json: async () => data, text: async () => '' })
  return {
    api: {
      get: vi.fn(async (url: string) => {
        if (url.startsWith('/api/v2/events/severity')) {
          return respond({ timeframe: 'alltime', total: 3, critical: 1, high: 1, medium: 1, low: 0, info: 0 })
        }
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

function mountChart() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(SeverityChart, {
    attachTo: document.body,
    global: { plugins: [pinia] },
  })
  return { wrapper, eventsStore: useEventsStore() }
}

describe('SeverityChart', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders title and canvas', async () => {
    const { wrapper } = await mountChart()
    await flushPromises()
    expect(wrapper.text()).toContain('Severity Distribution')
    expect(wrapper.text()).toContain('Active Threat Breakdown')
    expect(wrapper.find('canvas').exists()).toBe(true)
  })

  it('fetches the severity projection and shows the total', async () => {
    const { wrapper, eventsStore } = await mountChart()
    await flushPromises()
    await flushPromises()
    expect(api.get).toHaveBeenCalledWith(
      expect.stringContaining('/api/v2/events/severity'),
      expect.anything()
    )
    expect(eventsStore.severityProjection?.total).toBe(3)
    expect(wrapper.text()).toContain('3')
    expect(wrapper.text()).toContain('Events')
  })

  it('renders the legend', async () => {
    const { wrapper } = await mountChart()
    await flushPromises()
    for (const label of ['Crit', 'High', 'Med', 'Low', 'Info']) {
      expect(wrapper.text()).toContain(label)
    }
  })
})
