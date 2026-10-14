import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseVerticalNav from '../BaseVerticalNav.vue'
import BaseMeatballMenu from '../BaseMeatballMenu.vue'

const tabs = [
  { id: 'a', label: 'Alpha' },
  { divider: true },
  { id: 'b', label: 'Beta', danger: true },
]

describe('BaseVerticalNav', () => {
  it('renders tabs and marks the active one', () => {
    const wrapper = mount(BaseVerticalNav, { props: { modelValue: 'a', tabs } })
    expect(wrapper.text()).toContain('Alpha')
    expect(wrapper.text()).toContain('Beta')
    expect(wrapper.findAll('button')).toHaveLength(2)
    expect(wrapper.find('[data-active="true"]').text()).toBe('Alpha')
  })

  it('emits the picked tab', async () => {
    const wrapper = mount(BaseVerticalNav, { props: { modelValue: 'a', tabs } })
    await wrapper.findAll('button')[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['b']])
  })
})

describe('BaseMeatballMenu', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  afterEach(() => {
    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    document.body.innerHTML = ''
  })

  it('opens the dropdown with slot content', async () => {
    const wrapper = mount(BaseMeatballMenu, {
      props: { id: 'menu-1' },
      slots: { default: '<button>Item</button>' },
      attachTo: document.body,
    })
    expect(document.body.textContent).not.toContain('Item')
    await wrapper.find('[data-testid="meatball-trigger"]').trigger('click')
    expect(document.body.textContent).toContain('Item')
  })

  it('closes on outside click', async () => {
    const wrapper = mount(BaseMeatballMenu, {
      props: { id: 'menu-2' },
      slots: { default: '<button>Item</button>' },
      attachTo: document.body,
    })
    await wrapper.find('[data-testid="meatball-trigger"]').trigger('click')
    expect(document.body.textContent).toContain('Item')
    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await Promise.resolve()
    expect(document.body.textContent).not.toContain('Item')
  })
})
