import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AuthAlert from '../AuthAlert.vue'

function mountAlert(props: { show: boolean; message: string; testid?: string }) {
  setActivePinia(createPinia())
  return mount(AuthAlert, {
    props,
    attachTo: document.body,
    global: { plugins: [createPinia()] },
  })
}

describe('AuthAlert', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders nothing when hidden', () => {
    const wrapper = mountAlert({ show: false, message: 'Boom' })
    expect(wrapper.find('[data-testid="auth-alert"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Boom')
  })

  it('renders the message when shown', () => {
    const wrapper = mountAlert({ show: true, message: 'Boom' })
    expect(wrapper.find('[data-testid="auth-alert"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="auth-alert"]').text()).toContain('Boom')
  })

  it('supports a custom test id', () => {
    const wrapper = mountAlert({ show: true, message: 'Boom', testid: 'login-error' })
    expect(wrapper.find('[data-testid="login-error"]').exists()).toBe(true)
  })
})
