import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import EventSortHeader from '../EventSortHeader.vue'

function mountHeader(props: Record<string, unknown> = {}) {
  return mount(EventSortHeader, {
    props: { label: 'Threat', sortKey: 'severity', sortCol: 'timestamp', sortDesc: true, ...props },
  })
}

describe('EventSortHeader', () => {
  it('renders the label and default aria text', () => {
    const wrapper = mountHeader()
    expect(wrapper.text()).toContain('Threat')
    expect(wrapper.attributes('aria-label')).toBe('Sort by threat')
  })

  it('supports a custom aria label', () => {
    const wrapper = mountHeader({ ariaLabel: 'Sort by threat severity' })
    expect(wrapper.attributes('aria-label')).toBe('Sort by threat severity')
  })

  it('emits toggle with the sort key on click', async () => {
    const wrapper = mountHeader()
    await wrapper.find('th').trigger('click')
    expect(wrapper.emitted('toggle')).toEqual([['severity']])
  })
})
