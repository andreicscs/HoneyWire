import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseLogo from '../BaseLogo.vue'
import ThemeToggle from '../ThemeToggle.vue'

describe('BaseLogo', () => {
  it('renders the mark', () => {
    const wrapper = mount(BaseLogo)
    expect(wrapper.find('svg').exists()).toBe(true)
  })

  it('shows text when requested', () => {
    const wrapper = mount(BaseLogo, { props: { showText: true } })
    expect(wrapper.text()).toContain('HoneyWire Sentinel')
  })
})

describe('ThemeToggle', () => {
  it('emits toggle on click', async () => {
    const wrapper = mount(ThemeToggle)
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('toggle')).toHaveLength(1)
  })
})
