import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import NodeSyncModal from '../NodeSyncModal.vue'

const command = 'honeywire apply'
const yaml = 'services:\n  demo: {}'

function mountModal(show = true) {
  return mount(NodeSyncModal, {
    props: { show, syncCommand: command, syncComposeYaml: yaml },
    attachTo: document.body,
  })
}

function clickBodyButton(text: string) {
  const btn = [...document.body.querySelectorAll('button')].find((b) => b.textContent?.includes(text))!
  btn.dispatchEvent(new MouseEvent('click', { bubbles: true }))
  return Promise.resolve()
}

function manualSection(): HTMLElement | null {
  return document.body.querySelector('[data-testid="sync-manual-section"]')
}

describe('NodeSyncModal', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders nothing when closed', () => {
    mountModal(false)
    expect(document.body.textContent).not.toContain('Synchronize Node')
  })

  it('renders the sync command when open', () => {
    mountModal()
    expect(document.body.textContent).toContain('Synchronize Node')
    expect(document.body.textContent).toContain(command)
  })

  it('reveals manual deployment on toggle', async () => {
    mountModal()
    expect(manualSection()?.style.display).toBe('none')
    await clickBodyButton('Advanced / Manual Deployment')
    expect(manualSection()?.style.display).not.toBe('none')
    expect(manualSection()?.textContent).toContain('services:')
    expect(document.body.textContent).toContain('docker compose')
  })

  it('emits close from the Done button', async () => {
    const wrapper = mountModal()
    await clickBodyButton('Done')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
