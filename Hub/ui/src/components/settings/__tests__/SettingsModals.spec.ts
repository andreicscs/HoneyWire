import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import SettingsPasswordModal from '../SettingsPasswordModal.vue'
import SettingsResetModal from '../SettingsResetModal.vue'

function mountModal(component: any, show = true) {
  setActivePinia(createPinia())
  document.body.innerHTML = ''
  return mount(component, {
    props: { show },
    attachTo: document.body,
    global: { plugins: [createPinia()] },
  })
}

function submitForm(testid: string) {
  const form = document.body.querySelector(`[data-testid="${testid}"]`)!
  form.dispatchEvent(new Event('submit', { cancelable: true }))
  return nextTick()
}

function fillInput(testid: string, value: string) {
  const input = document.body.querySelector(`[data-testid="${testid}"] input`) as HTMLInputElement
  input.value = value
  input.dispatchEvent(new Event('input'))
  return nextTick()
}

function errorText(testid: string) {
  return document.body.querySelector(`[data-testid="${testid}"]`)?.textContent ?? ''
}

describe('SettingsPasswordModal', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders nothing when closed', () => {
    mountModal(SettingsPasswordModal, false)
    expect(document.body.textContent).not.toContain('Update Password')
    expect(document.body.querySelector('[data-testid="password-form"]')).toBeNull()
  })

  it('renders password fields when open', () => {
    mountModal(SettingsPasswordModal, true)
    expect(document.body.textContent).toContain('Update Password')
    for (const testid of ['pwd-current', 'pwd-new', 'pwd-confirm']) {
      expect(document.body.querySelector(`[data-testid="${testid}"] input`)).not.toBeNull()
    }
  })

  it('requires all fields', async () => {
    mountModal(SettingsPasswordModal, true)
    await submitForm('password-form')
    expect(errorText('password-error')).toContain('All fields are required.')
  })

  it('rejects mismatched new passwords', async () => {
    mountModal(SettingsPasswordModal, true)
    await fillInput('pwd-current', 'current')
    await fillInput('pwd-new', 'new-one')
    await fillInput('pwd-confirm', 'new-two')
    await submitForm('password-form')
    expect(errorText('password-error')).toContain('New passwords do not match.')
  })

  it('emits close from the cancel button', async () => {
    const wrapper = mountModal(SettingsPasswordModal, true)
    const buttons = [...document.body.querySelectorAll('[data-testid="password-form"] button')]
    const cancelBtn = buttons.find((b) => b.textContent?.includes('Cancel'))!
    cancelBtn.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})

describe('SettingsResetModal', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders nothing when closed', () => {
    mountModal(SettingsResetModal, false)
    expect(document.body.textContent).not.toContain('Confirm Factory Reset')
    expect(document.body.querySelector('[data-testid="reset-form"]')).toBeNull()
  })

  it('renders confirmation copy when open', () => {
    mountModal(SettingsResetModal, true)
    expect(document.body.textContent).toContain('Confirm Factory Reset')
    const input = document.body.querySelector('[data-testid="reset-password-input"] input') as HTMLInputElement
    expect(input?.getAttribute('placeholder')).toBe('Master Password')
  })

  it('requires the master password', async () => {
    mountModal(SettingsResetModal, true)
    await submitForm('reset-form')
    expect(errorText('reset-error')).toContain('Master password is required.')
  })
})
