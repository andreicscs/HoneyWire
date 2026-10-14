import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import CopyButton from '../CopyButton.vue'

const originalClipboard = (navigator as any).clipboard
const originalSecureContext = Object.getOwnPropertyDescriptor(window, 'isSecureContext')

function mountButton(props: Record<string, unknown> = {}) {
  return mount(CopyButton, {
    props: { copyKey: 'test-key', text: 'copy me', ...props },
    attachTo: document.body,
  })
}

describe('CopyButton', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    Object.defineProperty(window, 'isSecureContext', { value: true, configurable: true })
    Object.assign(navigator, { clipboard: { writeText: vi.fn(async () => {}) } })
  })

  afterEach(() => {
    if (originalSecureContext) Object.defineProperty(window, 'isSecureContext', originalSecureContext)
    Object.assign(navigator, { clipboard: originalClipboard })
    document.body.innerHTML = ''
  })

  it('renders Copy initially', () => {
    const wrapper = mountButton()
    expect(wrapper.find('[data-testid="copy-button"]').text()).toBe('Copy')
  })

  it('copies text and shows confirmation', async () => {
    const wrapper = mountButton()
    await wrapper.find('[data-testid="copy-button"]').trigger('click')
    await flushPromises()
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('copy me')
    expect(wrapper.find('[data-testid="copy-button"]').text()).toBe('Copied!')
  })

  it('ignores empty text', async () => {
    const wrapper = mountButton({ text: '' })
    await wrapper.find('[data-testid="copy-button"]').trigger('click')
    await flushPromises()
    expect(navigator.clipboard.writeText).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="copy-button"]').text()).toBe('Copy')
  })
})
