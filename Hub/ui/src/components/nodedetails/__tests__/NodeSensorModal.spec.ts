import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import NodeSensorModal from '../NodeSensorModal.vue'

vi.mock('../../../api/client', () => {
  const respond = async () => ({ ok: true, json: async () => ({}), text: async () => 'services:\n  demo: {}' })
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

const sensorFixture = () => ({
  id: 'hw-sensor-demo',
  name: 'Demo Sensor',
  description: 'Watches things.',
  icon_svg: 'M12 2v20',
  documentation: {
    summary: 'Demo summary',
    sections: [{ title: 'What', type: 'list', content: ['watches', 'alerts'] }],
  },
  deployment: {
    env_vars: [
      { name: 'HW_SEVERITY', type: 'text', default: 'high', description: 'Alert level' },
      { name: 'HW_LOG_LEVEL', type: 'text', default: 'info', description: 'Verbosity' },
    ],
  },
})

function mountModal(overrides = {}) {
  setActivePinia(createPinia())
  document.body.innerHTML = ''
  return mount(NodeSensorModal, {
    props: {
      show: true,
      sensor: sensorFixture(),
      isEditing: false,
      initialEnvVars: { HW_SEVERITY: 'high', HW_LOG_LEVEL: 'debug' },
      apiKey: 'key-1',
      ...overrides,
    },
    attachTo: document.body,
    global: { plugins: [createPinia()] },
  })
}

function bodyText() {
  return document.body.textContent ?? ''
}

describe('NodeSensorModal', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('renders nothing when closed', () => {
    mountModal({ show: false })
    expect(bodyText()).not.toContain('Demo Sensor')
  })

  it('renders sensor overview when open', () => {
    mountModal()
    expect(bodyText()).toContain('Demo Sensor')
    expect(bodyText()).toContain('Demo summary')
  })

  it('switches to the configuration tab', async () => {
    mountModal()
    const configTab = [...document.body.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Configuration')
    )!
    configTab.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    await nextTick()
    expect(bodyText()).toContain('Sensor Settings')
    expect(bodyText()).toContain('Alert Severity')
  })

  it('changes severity through the dropdown', async () => {
    mountModal()
    document
      .body.querySelector('[data-testid="severity-select"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    document
      .body.querySelector('[data-testid="severity-option"][data-value="low"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(
      document.body.querySelector('[data-testid="severity-select"]')?.textContent
    ).toContain('Low')
  })

  it('emits apply with stringified env values', async () => {
    const wrapper = mountModal()
    document
      .body.querySelector('[data-testid="severity-select"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    document
      .body.querySelector('[data-testid="severity-option"][data-value="low"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    document
      .body.querySelector('[data-testid="sensor-config-form"]')!
      .dispatchEvent(new Event('submit', { cancelable: true }))
    await nextTick()
    expect(wrapper.emitted('apply')?.[0]).toEqual([
      { HW_SEVERITY: 'low', HW_LOG_LEVEL: 'debug' },
    ])
  })

  it('emits close from the close button', async () => {
    const wrapper = mountModal()
    document
      .body.querySelector('[data-testid="sensor-modal-close"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
