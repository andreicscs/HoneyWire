import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import NodeKeyModal from '../NodeKeyModal.vue'

function mountModal(show: boolean, apiKey: string | null = 'secret-key') {
  return mount(NodeKeyModal, {
    props: { show, apiKey },
    attachTo: document.body,
  })
}

describe('NodeKeyModal', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders nothing when closed', () => {
    mountModal(false)
    expect(document.body.textContent).not.toContain('Manage Node Key')
  })

  it('renders the key when open', () => {
    mountModal(true)
    expect(document.body.textContent).toContain('Manage Node Key')
    expect(document.body.textContent).toContain('secret-key')
    const copyBtn = [...document.body.querySelectorAll('button')].find((b) => b.textContent === 'Copy')
    expect(copyBtn).toBeTruthy()
  })

  it('renders a fallback without a key', () => {
    mountModal(true, null)
    expect(document.body.textContent).toContain('Unavailable')
  })

  it('emits close from the Close button', async () => {
    const wrapper = mountModal(true)
    const closeBtn = [...document.body.querySelectorAll('button')].find((b) => b.textContent === 'Close')!
    closeBtn.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await Promise.resolve()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
