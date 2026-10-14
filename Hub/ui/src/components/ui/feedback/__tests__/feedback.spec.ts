import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import BaseAlert from '../BaseAlert.vue'
import BaseModal from '../BaseModal.vue'
import BaseStatusDot from '../BaseStatusDot.vue'
import BaseLegend from '../BaseLegend.vue'

enableAutoUnmount(afterEach)

describe('BaseAlert', () => {
  it('renders the message with variant reflection', () => {
    for (const variant of ['info', 'danger', 'success', 'archive'] as const) {
      const wrapper = mount(BaseAlert, { props: { variant, message: `hello ${variant}` } })
      expect(wrapper.text()).toContain(`hello ${variant}`)
      expect(wrapper.attributes('data-variant')).toBe(variant)
    }
  })
})

describe('BaseModal', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })
  it('renders nothing when closed', () => {
    mount(BaseModal, {
      props: { show: false, title: 'Title' },
      slots: { default: 'body' },
      attachTo: document.body,
    })
    expect(document.body.textContent).not.toContain('Title')
  })

  it('renders title and slot when open', () => {
    mount(BaseModal, {
      props: { show: true, title: 'Title' },
      slots: { default: 'modal body' },
      attachTo: document.body,
    })
    expect(document.body.textContent).toContain('Title')
    expect(document.body.textContent).toContain('modal body')
    expect(document.body.querySelector('[data-danger="false"]')).not.toBeNull()
  })

  it('marks danger modals', () => {
    mount(BaseModal, {
      props: { show: true, title: 'Danger', danger: true },
      attachTo: document.body,
    })
    expect(document.body.querySelector('[data-danger="true"]')).not.toBeNull()
  })

  it('emits close on backdrop click', async () => {
    const wrapper = mount(BaseModal, {
      props: { show: true, title: 'Title' },
      attachTo: document.body,
    })
    document
      .body.querySelector('[data-testid="base-modal-overlay"]')!
      .dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    await Promise.resolve()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})

describe('BaseStatusDot', () => {
  it('maps statuses to tones', () => {
    const cases: Array<[string, string]> = [
      ['up', 'success'],
      ['online', 'success'],
      ['down', 'danger'],
      ['offline', 'danger'],
      ['critical', 'danger'],
      ['degraded', 'warning'],
      ['high', 'warning'],
      ['pending', 'muted'],
      ['nodata', 'muted'],
    ]
    for (const [status, tone] of cases) {
      const wrapper = mount(BaseStatusDot, { props: { status } })
      expect(wrapper.attributes('data-tone')).toBe(tone)
    }
  })

  it('renders unknown statuses without a tone', () => {
    const wrapper = mount(BaseStatusDot, { props: { status: 'mystery' } })
    expect(wrapper.attributes('data-tone')).toBe('')
  })
})

describe('BaseLegend', () => {
  it('renders all items with labels', () => {
    const wrapper = mount(BaseLegend, {
      props: { items: [{ label: 'Up', colorClass: 'bg-success-main' }, { label: 'Down', colorClass: 'bg-critical' }] },
    })
    expect(wrapper.text()).toContain('Up')
    expect(wrapper.text()).toContain('Down')
  })
})
