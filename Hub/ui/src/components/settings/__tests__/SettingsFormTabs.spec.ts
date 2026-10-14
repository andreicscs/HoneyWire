import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import SettingsDataTab from '../SettingsDataTab.vue'
import SettingsSiemTab from '../SettingsSiemTab.vue'
import SettingsSecurityTab from '../SettingsSecurityTab.vue'

function mountWithPinia(component: any, props: Record<string, unknown> = {}) {
  setActivePinia(createPinia())
  return mount(component, {
    props,
    attachTo: document.body,
    global: { plugins: [createPinia()] },
  })
}

describe('SettingsDataTab', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders retention policies and steppers', () => {
    const wrapper = mountWithPinia(SettingsDataTab, {
      modelValue: { autoArchiveDays: 0, autoPurgeDays: 0, autoPurgeHeartbeatsDays: 0 },
    })
    expect(wrapper.text()).toContain('Database Retention Policies')
    expect(wrapper.text()).toContain('Events Data')
    expect(wrapper.text()).toContain('Uptime Data')
    expect(wrapper.text()).toContain('Auto-Archive Events')
    expect(wrapper.text()).toContain('Auto-Purge Archive')
    expect(wrapper.text()).toContain('Auto-Purge Uptime Data')
  })
})

describe('SettingsSiemTab', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders SIEM forwarding controls', () => {
    const wrapper = mountWithPinia(SettingsSiemTab, {
      modelValue: { siemAddress: '', siemProtocol: 'tcp' },
    })
    expect(wrapper.text()).toContain('SIEM Forwarding')
    expect(wrapper.text()).toContain('Server Address')
    expect(wrapper.text()).toContain('Protocol')
  })
})

describe('SettingsSecurityTab', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders authentication and danger zone', () => {
    const wrapper = mountWithPinia(SettingsSecurityTab)
    expect(wrapper.text()).toContain('Authentication')
    expect(wrapper.text()).toContain('Change Master Password')
    expect(wrapper.text()).toContain('Danger Zone')
    expect(wrapper.text()).toContain('Factory Reset')
    expect(wrapper.find('[data-testid="danger-zone"]').exists()).toBe(true)
  })

  it('emits change-password when the password button is clicked', async () => {
    const wrapper = mountWithPinia(SettingsSecurityTab)
    await wrapper.find('[data-testid="change-password-button"]').trigger('click')
    expect(wrapper.emitted('change-password')).toHaveLength(1)
  })

  it('emits reset-system when the reset button is clicked', async () => {
    const wrapper = mountWithPinia(SettingsSecurityTab)
    await wrapper.find('[data-testid="reset-system-button"]').trigger('click')
    expect(wrapper.emitted('reset-system')).toHaveLength(1)
  })
})
