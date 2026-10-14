import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseButton from '../BaseButton.vue'
import BaseInput from '../BaseInput.vue'
import BaseNumberStepper from '../BaseNumberStepper.vue'
import BaseRadioGroup from '../BaseRadioGroup.vue'
import BaseTimeFilter from '../BaseTimeFilter.vue'

describe('BaseButton', () => {
  it('renders slot content with defaults', () => {
    const wrapper = mount(BaseButton, { slots: { default: 'Save' } })
    expect(wrapper.text()).toBe('Save')
    expect(wrapper.attributes('type')).toBe('button')
    expect(wrapper.attributes('data-variant')).toBe('secondary')
    expect(wrapper.attributes('disabled')).toBeUndefined()
  })

  it('reflects variant, type and disabled', () => {
    const wrapper = mount(BaseButton, {
      props: { variant: 'danger', type: 'submit', disabled: true },
      slots: { default: 'Delete' },
    })
    expect(wrapper.attributes('data-variant')).toBe('danger')
    expect(wrapper.attributes('type')).toBe('submit')
    expect(wrapper.attributes('disabled')).toBeDefined()
  })
})

describe('BaseInput', () => {
  it('renders label with required marker and description', () => {
    const wrapper = mount(BaseInput, {
      props: { modelValue: '', label: 'Name', required: true, description: 'Pick one' },
    })
    expect(wrapper.text()).toContain('Name')
    expect(wrapper.text()).toContain('*')
    expect(wrapper.text()).toContain('Pick one')
  })

  it('emits text input', async () => {
    const wrapper = mount(BaseInput, { props: { modelValue: '' } })
    await wrapper.find('input').setValue('hello')
    expect(wrapper.emitted('update:modelValue')).toEqual([['hello']])
  })

  it('increments and decrements numbers as strings', async () => {
    const wrapper = mount(BaseInput, { props: { modelValue: 5, type: 'number', min: 0, max: 10 } })
    await wrapper.find('[data-testid="num-increment"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['6']])
    await wrapper.find('[data-testid="num-decrement"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[1]).toEqual(['4'])
  })

  it('selects boolean values from the dropdown', async () => {
    const wrapper = mount(BaseInput, { props: { modelValue: '', type: 'boolean' } })
    expect(wrapper.find('[data-testid="boolean-options"]').exists()).toBe(false)
    await wrapper.find('[data-testid="boolean-select"]').trigger('click')
    expect(wrapper.find('[data-testid="boolean-options"]').exists()).toBe(true)
    await wrapper.find('[data-testid="boolean-option"][data-value="true"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['true']])
  })
})

describe('BaseNumberStepper', () => {
  it('renders label, description and suffix', () => {
    const wrapper = mount(BaseNumberStepper, {
      props: { modelValue: 7, label: 'Days', description: 'How many', suffix: 'Days' },
    })
    expect(wrapper.text()).toContain('Days')
    expect(wrapper.text()).toContain('How many')
    expect(wrapper.find('input').element.value).toBe('7')
  })

  it('adjusts within min/max bounds', async () => {
    const wrapper = mount(BaseNumberStepper, { props: { modelValue: 10, label: 'Days', min: 0, max: 10 } })
    const buttons = wrapper.findAll('button')
    await buttons[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[10]])
    await buttons[0].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[1]).toEqual([9])
  })

  it('emits typed input as number', async () => {
    const wrapper = mount(BaseNumberStepper, { props: { modelValue: 1, label: 'Days' } })
    await wrapper.find('input').setValue('4')
    expect(wrapper.emitted('update:modelValue')).toEqual([[4]])
  })
})

describe('BaseRadioGroup', () => {
  it('renders options and marks the selected one', async () => {
    const wrapper = mount(BaseRadioGroup, {
      props: { modelValue: 'ntfy', label: 'Provider', options: ['ntfy', 'slack'] },
    })
    expect(wrapper.text()).toContain('Provider')
    expect(wrapper.findAll('button')).toHaveLength(2)
    expect(wrapper.find('[data-selected="true"]').text()).toBe('ntfy')
    await wrapper.findAll('button')[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['slack']])
  })
})

describe('BaseTimeFilter', () => {
  it('renders default options with the active one marked', () => {
    const wrapper = mount(BaseTimeFilter, { props: { modelValue: '24H' } })
    for (const opt of ['1H', '24H', '7D', '30D']) {
      expect(wrapper.text()).toContain(opt)
    }
    expect(wrapper.find('[data-active="true"]').text()).toBe('24H')
  })

  it('emits the picked timeframe', async () => {
    const wrapper = mount(BaseTimeFilter, { props: { modelValue: '24H' } })
    await wrapper.findAll('button').find((b) => b.text() === '7D')!.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['7D']])
  })
})
