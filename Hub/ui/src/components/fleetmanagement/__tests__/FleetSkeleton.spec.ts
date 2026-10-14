import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FleetSkeleton from '../FleetSkeleton.vue'

describe('FleetSkeleton', () => {
  it('renders four placeholder cards', () => {
    const wrapper = mount(FleetSkeleton, { attachTo: document.body })
    expect(wrapper.find('[data-testid="fleet-skeleton"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid="skeleton-card"]')).toHaveLength(4)
  })
})
