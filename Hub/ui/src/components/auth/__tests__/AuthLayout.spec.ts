import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AuthLayout from '../AuthLayout.vue'
import ThemeToggle from '../../ui/branding/ThemeToggle.vue'

function mountLayout(props: Record<string, unknown> = {}) {
  setActivePinia(createPinia())
  return mount(AuthLayout, {
    props,
    slots: { default: '<p data-testid="slot-body">form goes here</p>' },
    attachTo: document.body,
    global: { plugins: [createPinia()] },
  })
}

describe('AuthLayout', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders shared branding and slot content', () => {
    const wrapper = mountLayout()
    expect(wrapper.find('[data-testid="auth-layout"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('HoneyWire Sentinel')
    expect(wrapper.text()).toContain('Authorized personnel only.')
    expect(wrapper.find('[data-testid="slot-body"]').exists()).toBe(true)
  })

  it('uses the narrow container by default', () => {
    const wrapper = mountLayout()
    expect(wrapper.find('[data-testid="auth-container"]').attributes('data-width')).toBe('sm')
  })

  it('uses the wide container when requested', () => {
    const wrapper = mountLayout({ width: 'xl' })
    expect(wrapper.find('[data-testid="auth-container"]').attributes('data-width')).toBe('xl')
  })

  it('re-emits theme toggle', async () => {
    const wrapper = mountLayout()
    await wrapper.findComponent(ThemeToggle).vm.$emit('toggle')
    expect(wrapper.emitted('toggle-theme')).toHaveLength(1)
  })
})
