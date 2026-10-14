import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import NodeDetailHeader from '../NodeDetailHeader.vue'

enableAutoUnmount(afterEach)

const nodeFixture = () => ({
  id: 'node-1',
  alias: 'Test Node',
  status: 'up',
  hasPendingConfig: false,
  lastHeartbeat: new Date().toISOString(),
  publicIp: '1.2.3.4',
  privateIp: '10.0.0.1',
  tags: ['lab', 'prod'],
  apiKey: 'key-1',
  activeRevision: 'r1',
  desiredRevision: 'r1',
  lastEvent: 'Never',
  hasUpdateAvailable: false,
  isSilenced: false,
  installedSensors: [],
})

function mountHeader(node = nodeFixture(), lastEventTime = 'None') {
  return mount(NodeDetailHeader, {
    props: { node, lastEventTime },
    attachTo: document.body,
  })
}

function closeMenus() {
  document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

describe('NodeDetailHeader', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  afterEach(() => {
    closeMenus()
    document.body.innerHTML = ''
  })

  it('renders alias, IPs, tags and last event time', () => {
    const wrapper = mountHeader()
    expect(wrapper.text()).toContain('Test Node')
    expect(wrapper.text()).toContain('1.2.3.4')
    expect(wrapper.text()).toContain('10.0.0.1')
    expect(wrapper.text()).toContain('lab')
    expect(wrapper.text()).toContain('Last Event:')
    expect(wrapper.text()).toContain('None')
  })

  it('emits update with the new alias on rename', async () => {
    const wrapper = mountHeader()
    await wrapper.find('h1').trigger('click')
    const input = wrapper.find('h1 + input, div input')
    expect(input.exists()).toBe(true)
    await input.setValue('Renamed Node')
    await input.trigger('keyup.enter')
    expect(wrapper.emitted('update')).toEqual([[{ alias: 'Renamed Node' }]])
  })

  it('emits manageKey from the Manage Key button', async () => {
    const wrapper = mountHeader()
    await wrapper.findAll('button').find((b) => b.text().includes('Manage Key'))!.trigger('click')
    expect(wrapper.emitted('manageKey')).toHaveLength(1)
  })

  it('shows pending-sync banner and emits sync', async () => {
    const wrapper = mountHeader({ ...nodeFixture(), hasPendingConfig: true })
    expect(wrapper.text()).toContain('Pending Sync')
    await wrapper.findAll('button').find((b) => b.text() === 'Sync Node')!.trigger('click')
    expect(wrapper.emitted('sync')).toHaveLength(1)
  })

  it('hides the pending-sync banner when in sync', () => {
    const wrapper = mountHeader()
    expect(wrapper.text()).not.toContain('Pending Sync')
  })

  it('emits delete from the meatball menu', async () => {
    const wrapper = mountHeader()
    await wrapper.find('[data-testid="meatball-trigger"]').trigger('click')
    document
      .body.querySelector('[data-testid="node-menu-delete"]')!
      .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(wrapper.emitted('delete')).toHaveLength(1)
  })
})
