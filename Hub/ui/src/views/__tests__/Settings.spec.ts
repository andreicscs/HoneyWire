import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Settings from '../Settings.vue'

function mountSettings() {
  return mount(Settings, {
    attachTo: document.body,
    global: {
      plugins: [createPinia()],
    },
  })
}

describe('Settings view (regression)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    document.body.innerHTML = ''
  })

  it('renders the page header and layout container', () => {
    const wrapper = mountSettings()
    expect(wrapper.text()).toContain('System Settings')
    expect(wrapper.find('[data-testid="settings-page"]').exists()).toBe(true)
  })

  it('renders the content scroll region', () => {
    const wrapper = mountSettings()
    expect(wrapper.find('[data-testid="settings-scroll-region"]').exists()).toBe(true)
  })

  it('renders all settings tabs in the vertical nav', () => {
    const wrapper = mountSettings()
    const text = wrapper.text()
    for (const label of [
      'Hub Configuration',
      'Data Retention',
      'Push Notifications',
      'SIEM Forwarding',
      'Security & Access',
    ]) {
      expect(text).toContain(label)
    }
  })

  it('disables Save Changes when there are no unsaved changes', () => {
    const wrapper = mountSettings()
    const save = wrapper.find('[data-testid="settings-save-button"]')
    expect(save.exists()).toBe(true)
    expect(save.text()).toContain('Save Changes')
    expect(save.attributes('disabled')).toBeDefined()
  })

  it('shows general tab content by default', () => {
    const wrapper = mountSettings()
    expect(wrapper.text()).toContain('Network Configuration')
    expect(wrapper.text()).toContain('System Updates')
    expect(wrapper.text()).toContain('Event Filtering')
  })

  it('switches visible panel when a nav tab is clicked', async () => {
    const wrapper = mountSettings()
    const navButtons = wrapper.findAll('nav button')
    const dataTab = navButtons.find((b) => b.text().includes('Data Retention'))
    expect(dataTab).toBeTruthy()
    await dataTab!.trigger('click')
    expect(wrapper.text()).toContain('Database Retention Policies')
    expect(wrapper.text()).toContain('Uptime Data')
  })

  it('renders five severity trigger buttons in alerts tab', async () => {
    const wrapper = mountSettings()
    const navButtons = wrapper.findAll('nav button')
    const alertsTab = navButtons.find((b) => b.text().includes('Push Notifications'))
    await alertsTab!.trigger('click')
    const sevButtons = wrapper.findAll('[data-testid="severity-trigger"]')
    expect(sevButtons).toHaveLength(5)
    for (const sev of ['critical', 'high', 'medium', 'low', 'info']) {
      expect(wrapper.find(`[data-severity="${sev}"]`).exists()).toBe(true)
    }
    // buttons use inline :style (not classes) — must be preserved by refactor
    expect(sevButtons[0].attributes('style')).toBeTruthy()
  })

  it('toggles a severity when its button is clicked', async () => {
    const wrapper = mountSettings()
    const navButtons = wrapper.findAll('nav button')
    await navButtons.find((b) => b.text().includes('Push Notifications'))!.trigger('click')
    const before = wrapper.find('[data-severity="critical"]')
    const styleBefore = before.attributes('style') ?? ''
    await before.trigger('click')
    const after = wrapper.find('[data-severity="critical"]')
    expect(after.attributes('style') ?? '').not.toBe(styleBefore)
  })

  it('shows unsafe-protocol warning for default http origin', () => {
    const wrapper = mountSettings()
    // jsdom origin is http://localhost — default hubEndpoint triggers warning
    expect(wrapper.text()).toContain('Unsafe Protocol Detected')
    expect(wrapper.find('[data-testid="unsafe-protocol-warning"]').exists()).toBe(true)
  })

  it('renders manual update check affordance', () => {
    const wrapper = mountSettings()
    expect(wrapper.text()).toContain('Manual Check')
    const checkBtn = wrapper.find('[data-testid="update-check-button"]')
    expect(checkBtn.exists()).toBe(true)
    expect(checkBtn.text()).toContain('Check now')
  })

  it('opens the password modal from the security tab', async () => {
    const wrapper = mountSettings()
    const navButtons = wrapper.findAll('nav button')
    await navButtons.find((b) => b.text().includes('Security'))!.trigger('click')
    const changeBtn = wrapper.find('[data-testid="change-password-button"]')
    expect(changeBtn.exists()).toBe(true)
    await changeBtn.trigger('click')
    // BaseModal teleports to body
    expect(document.body.textContent).toContain('Update Password')
    expect(document.body.textContent).toContain('Current Password')
  })

  it('opens the factory-reset modal from the security tab', async () => {
    const wrapper = mountSettings()
    const navButtons = wrapper.findAll('nav button')
    await navButtons.find((b) => b.text().includes('Security'))!.trigger('click')
    const resetBtn = wrapper.find('[data-testid="reset-system-button"]')
    expect(resetBtn.exists()).toBe(true)
    await resetBtn.trigger('click')
    expect(document.body.textContent).toContain('Confirm Factory Reset')
    expect(document.body.querySelector('[data-testid="reset-password-input"]')).not.toBeNull()
  })

  it('renders danger-zone card with danger styling hooks', async () => {
    const wrapper = mountSettings()
    const navButtons = wrapper.findAll('nav button')
    await navButtons.find((b) => b.text().includes('Security'))!.trigger('click')
    expect(wrapper.text()).toContain('Danger Zone')
    expect(wrapper.text()).toContain('Factory Reset')
    expect(wrapper.find('[data-testid="danger-zone"]').exists()).toBe(true)
  })
})
