import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ThreatVelocity from '../ThreatVelocity.vue'
import BaseTimeFilter from '../../ui/forms/BaseTimeFilter.vue'
import { useAppStore } from '../../../stores/System/app'
import { api } from '../../../api/client'

enableAutoUnmount(afterEach)

const velocityMode = vi.hoisted(() => ({ value: 'some' as 'some' | 'zero' }))

vi.mock('chart.js/auto', () => {
  const MockChart = vi.fn(function (this: any) {
    return {
      data: { labels: [], datasets: [] },
      options: { plugins: { tooltip: {} }, scales: { x: { ticks: {} }, y: {} } },
      update: vi.fn(),
      destroy: vi.fn(),
    }
  }) as any
  MockChart.register = vi.fn()
  return { default: MockChart }
})

vi.mock('../../../api/client', () => {
  const some = {
    timeframe: '24H', bucketSizeMs: 3600000, generatedAt: Date.now(),
    bucketTimestamps: [], labels: ['10:00'], exactTimes: ['today 10:00'],
    series: { critical: [1], high: [0], medium: [0], low: [0], info: [0] },
    recentEventCount: 5,
  }
  const zero = { ...some, recentEventCount: 0, series: { critical: [0], high: [0], medium: [0], low: [0], info: [0] } }
  const respond = async (data: any = {}) => ({ ok: true, json: async () => data, text: async () => '' })
  return {
    api: {
      get: vi.fn(async (url: string) => {
        if (url.startsWith('/api/v2/events/velocity')) {
          return respond(velocityMode.value === 'zero' ? zero : some)
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

function mountVelocity() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(ThreatVelocity, {
    attachTo: document.body,
    global: { plugins: [pinia] },
  })
  return { wrapper, appStore: useAppStore() }
}

describe('ThreatVelocity', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    velocityMode.value = 'some'
    vi.clearAllMocks()
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders title and event count', async () => {
    const { wrapper } = await mountVelocity()
    await flushPromises()
    await flushPromises()
    expect(wrapper.text()).toContain('Events Velocity')
    expect(wrapper.text()).toContain('Events Recorded')
    expect(wrapper.text()).toContain('5')
  })

  it('shows awaiting state without telemetry', async () => {
    velocityMode.value = 'zero'
    const { wrapper } = await mountVelocity()
    await flushPromises()
    await flushPromises()
    expect(wrapper.text()).toContain('Awaiting telemetry...')
  })

  it('changes timeframe through the filter', async () => {
    const { wrapper, appStore } = await mountVelocity()
    await flushPromises()
    await wrapper.findComponent(BaseTimeFilter).vm.$emit('update:modelValue', '7D')
    await flushPromises()
    await flushPromises()
    expect(appStore.velocityTimeframe).toBe('7D')
    expect(api.get).toHaveBeenCalledWith(expect.stringContaining('timeframe=7D'), expect.anything())
  })
})
