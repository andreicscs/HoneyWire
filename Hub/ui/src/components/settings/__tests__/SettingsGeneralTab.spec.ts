import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SettingsGeneralTab from '../SettingsGeneralTab.vue'

const baseSettings = () => ({
  hubEndpoint: 'http://localhost:8080',
  registryUrl: '',
  updateCheckIntervalHours: 24,
  whitelistedSources: '',
})

function mountTab(settings = baseSettings()) {
  setActivePinia(createPinia())
  return mount(SettingsGeneralTab, {
    props: { modelValue: settings },
    global: { plugins: [createPinia()] },
  })
}

describe('SettingsGeneralTab', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders network, updates and filtering sections', () => {
    const wrapper = mountTab()
    expect(wrapper.text()).toContain('Network Configuration')
    expect(wrapper.text()).toContain('System Updates')
    expect(wrapper.text()).toContain('Event Filtering')
    expect(wrapper.text()).toContain('Hub Endpoint URL')
    expect(wrapper.text()).toContain('Update Check Interval (Hours)')
    expect(wrapper.text()).toContain('Whitelisted Event Sources')
  })

  it('shows the unsafe-protocol warning for http endpoints', () => {
    const wrapper = mountTab()
    expect(wrapper.text()).toContain('Unsafe Protocol Detected')
    expect(wrapper.find('[data-testid="unsafe-protocol-warning"]').exists()).toBe(true)
  })

  it('hides the unsafe-protocol warning for https endpoints', () => {
    const wrapper = mountTab({ ...baseSettings(), hubEndpoint: 'https://hub.example.com' })
    expect(wrapper.text()).not.toContain('Unsafe Protocol Detected')
    expect(wrapper.find('[data-testid="unsafe-protocol-warning"]').exists()).toBe(false)
  })

  it('renders the manual update-check affordance', () => {
    const wrapper = mountTab()
    expect(wrapper.text()).toContain('Manual Check')
    const checkBtn = wrapper.find('[data-testid="update-check-button"]')
    expect(checkBtn.exists()).toBe(true)
    expect(checkBtn.text()).toContain('Check now')
  })
})
