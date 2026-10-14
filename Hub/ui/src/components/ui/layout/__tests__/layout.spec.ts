import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseCard from '../BaseCard.vue'
import BaseDivider from '../BaseDivider.vue'
import BaseWidget from '../BaseWidget.vue'
import PageHeader from '../PageHeader.vue'

describe('BaseCard', () => {
  it('renders title and slot content', () => {
    const wrapper = mount(BaseCard, {
      props: { title: 'Card Title' },
      slots: { default: 'card body' },
    })
    expect(wrapper.text()).toContain('Card Title')
    expect(wrapper.text()).toContain('card body')
  })

  it('renders icon slot in danger mode', () => {
    const wrapper = mount(BaseCard, {
      props: { danger: true },
      slots: { icon: '<i>!</i>', default: 'danger body' },
    })
    expect(wrapper.text()).toContain('danger body')
    expect(wrapper.find('i').exists()).toBe(true)
  })
})

describe('BaseDivider', () => {
  it('renders', () => {
    expect(mount(BaseDivider).isVisible()).toBe(true)
  })
})

describe('BaseWidget', () => {
  it('renders header, body and footer slots', () => {
    const wrapper = mount(BaseWidget, {
      slots: { header: 'head', default: 'body', footer: 'foot' },
    })
    expect(wrapper.text()).toContain('head')
    expect(wrapper.text()).toContain('body')
    expect(wrapper.text()).toContain('foot')
  })
})

describe('PageHeader', () => {
  it('renders md title and description', () => {
    const wrapper = mount(PageHeader, {
      props: { title: 'Big Title', description: 'Sub line' },
    })
    expect(wrapper.find('h1').exists()).toBe(true)
    expect(wrapper.text()).toContain('Big Title')
    expect(wrapper.text()).toContain('Sub line')
  })

  it('renders sm title as h3', () => {
    const wrapper = mount(PageHeader, { props: { title: 'Small', size: 'sm' } })
    expect(wrapper.find('h1').exists()).toBe(false)
    expect(wrapper.find('h3').exists()).toBe(true)
    expect(wrapper.text()).toContain('Small')
  })

  it('renders actions slot', () => {
    const wrapper = mount(PageHeader, {
      props: { title: 'T' },
      slots: { actions: '<button>Go</button>' },
    })
    expect(wrapper.text()).toContain('Go')
  })
})
