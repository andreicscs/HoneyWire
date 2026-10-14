import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SettingsAlertsTab from '../SettingsAlertsTab.vue'

const baseSettings = () => ({
  webhookType: 'ntfy',
  webhookUrl: '',
  webhookEvents: [] as string[],
})

function mountTab(settings = baseSettings()) {
  setActivePinia(createPinia())
  return mount(SettingsAlertsTab, {
    props: { modelValue: settings },
    global: { plugins: [createPinia()] },
  })
}

describe('SettingsAlertsTab', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders provider selection and target URL input', () => {
    const wrapper = mountTab()
    expect(wrapper.text()).toContain('Push Notifications')
    expect(wrapper.text()).toContain('Service Provider')
    expect(wrapper.text()).toContain('Target URL')
    expect(wrapper.text()).toContain('ntfy')
  })

  it('renders five severity buttons with inline styles', () => {
    const wrapper = mountTab()
    const sevButtons = wrapper.findAll('[data-testid="severity-trigger"]')
    expect(sevButtons).toHaveLength(5)
    for (const sev of ['critical', 'high', 'medium', 'low', 'info']) {
      expect(wrapper.find(`[data-severity="${sev}"]`).exists()).toBe(true)
    }
    expect(sevButtons[0].attributes('style')).toBeTruthy()
  })

  it('activates a severity on click', async () => {
    const wrapper = mountTab()
    const styleBefore = wrapper.find('[data-severity="critical"]').attributes('style') ?? ''
    await wrapper.find('[data-severity="critical"]').trigger('click')
    expect(wrapper.find('[data-severity="critical"]').attributes('style') ?? '').not.toBe(styleBefore)
  })

  it('deactivates an active severity on click', async () => {
    const wrapper = mountTab({ ...baseSettings(), webhookEvents: ['critical'] })
    const styleBefore = wrapper.find('[data-severity="critical"]').attributes('style') ?? ''
    await wrapper.find('[data-severity="critical"]').trigger('click')
    expect(wrapper.find('[data-severity="critical"]').attributes('style') ?? '').not.toBe(styleBefore)
  })

  it('updates the target URL description for gotify', async () => {
    const wrapper = mountTab()
    await wrapper.setProps({ modelValue: { ...baseSettings(), webhookType: 'gotify' } })
    expect(wrapper.text()).toContain('Gotify')
  })
})
