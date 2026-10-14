import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Setup from '../Setup.vue'

const setupMode = vi.hoisted(() => ({ value: 'success' as 'success' | 'failure' }))

vi.mock('../../api/client', () => {
  class FakeApiError extends Error {
    status: number
    constructor(message: string, status: number) {
      super(message)
      this.status = status
    }
  }
  const respond = async () => ({ ok: true, json: async () => ({}), text: async () => '' })
  return {
    api: {
      get: vi.fn(respond),
      post: vi.fn(async (url: string) => {
        if (url === '/api/v2/setup' && setupMode.value === 'failure') {
          throw new FakeApiError('Setup unavailable', 500)
        }
        return respond()
      }),
      put: vi.fn(respond),
      patch: vi.fn(respond),
      delete: vi.fn(respond),
      request: vi.fn(respond),
    },
  }
})

function mountSetup() {
  const pinia = createPinia()
  setActivePinia(pinia)
  return mount(Setup, {
    attachTo: document.body,
    global: { plugins: [pinia] },
  })
}

async function fillSetup(wrapper: any, password: string, confirm: string, endpoint?: string) {
  await wrapper.find('[data-testid="setup-password"] input').setValue(password)
  await wrapper.find('[data-testid="setup-confirm"] input').setValue(confirm)
  if (endpoint !== undefined) {
    await wrapper.find('[data-testid="setup-endpoint"] input').setValue(endpoint)
  }
  await wrapper.find('form').trigger('submit')
  await flushPromises()
}

describe('Setup view (regression)', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    setupMode.value = 'success'
    vi.clearAllMocks()
  })

  it('renders master auth and connectivity sections', () => {
    const wrapper = mountSetup()
    expect(wrapper.text()).toContain('HoneyWire Sentinel')
    expect(wrapper.text()).toContain('Master Authentication')
    expect(wrapper.text()).toContain('Hub Connectivity')
    expect(wrapper.find('[data-testid="setup-submit"]').text()).toContain('Initialize Hub')
  })

  it('requires all fields', async () => {
    const wrapper = mountSetup()
    await fillSetup(wrapper, '', '', '')
    expect(wrapper.find('[data-testid="setup-error"]').text()).toContain('All fields are required.')
    expect(wrapper.emitted('setup-complete')).toBeUndefined()
  })

  it('rejects mismatched passwords', async () => {
    const wrapper = mountSetup()
    await fillSetup(wrapper, 'one', 'two')
    expect(wrapper.find('[data-testid="setup-error"]').text()).toContain('Passwords do not match.')
    expect(wrapper.emitted('setup-complete')).toBeUndefined()
  })

  it('emits setup-complete on success', async () => {
    const wrapper = mountSetup()
    await fillSetup(wrapper, 'secret', 'secret', 'http://hub:8080')
    expect(wrapper.emitted('setup-complete')).toHaveLength(1)
    expect(wrapper.find('[data-testid="setup-error"]').exists()).toBe(false)
  })

  it('shows server errors', async () => {
    setupMode.value = 'failure'
    const wrapper = mountSetup()
    await fillSetup(wrapper, 'secret', 'secret', 'http://hub:8080')
    expect(wrapper.emitted('setup-complete')).toBeUndefined()
    expect(wrapper.find('[data-testid="setup-error"]').text()).toContain('Setup unavailable')
  })
})
