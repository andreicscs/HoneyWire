import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import FleetNodeWidget from '../FleetNodeWidget.vue'

enableAutoUnmount(afterEach)

const nodeFixture = (overrides = {}) => ({
  id: 'node-1',
  alias: 'Fleet Node One',
  status: 'up',
  hasPendingConfig: false,
  lastHeartbeat: new Date().toISOString(),
  publicIp: '1.2.3.4',
  privateIp: '10.0.0.1',
  tags: ['lab'],
  apiKey: 'key-1',
  hasUpdateAvailable: false,
  isSilenced: false,
  isAwaitingCheckIn: false,
  totalSensors: 2,
  onlineSensors: 1,
  sensorSummary: [{ type: 'Network', count: 2, sensors: [] }],
  ...overrides,
})

function mountWidget(node = nodeFixture(), extraProps = {}) {
  return mount(FleetNodeWidget, {
    props: { node, isManifestLoading: false, isDeleting: false, ...extraProps },
    attachTo: document.body,
  })
}

function closeMenus() {
  document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

async function clickMenuItem(wrapper: any, testid: string) {
  await wrapper.find('[data-testid="meatball-trigger"]').trigger('click')
  document
    .body.querySelector(`[data-testid="${testid}"]`)!
    .dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

describe('FleetNodeWidget', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  afterEach(() => {
    closeMenus()
    document.body.innerHTML = ''
  })

  it('renders alias, IPs, tags and sensor counts', () => {
    const wrapper = mountWidget()
    expect(wrapper.text()).toContain('Fleet Node One')
    expect(wrapper.text()).toContain('1.2.3.4')
    expect(wrapper.text()).toContain('10.0.0.1')
    expect(wrapper.text()).toContain('lab')
    expect(wrapper.text()).toContain('1 / 2 Online')
    expect(wrapper.text()).toContain('Manage Node & Sensors')
  })

  it('renders install CTA without sensors', () => {
    const wrapper = mountWidget(nodeFixture({ totalSensors: 0, onlineSensors: 0, sensorSummary: [] }))
    expect(wrapper.text()).toContain('Install First Sensor')
    expect(wrapper.text()).toContain('No sensors deployed.')
  })

  it('shows awaiting check-in state', () => {
    const wrapper = mountWidget(nodeFixture({ isAwaitingCheckIn: true }))
    expect(wrapper.text()).toContain('Awaiting Initial Check-in')
  })

  it('emits update with the new alias on rename', async () => {
    const wrapper = mountWidget()
    await wrapper.findAll('span').find((s) => s.text() === 'Fleet Node One')!.trigger('click')
    const input = wrapper.find('input')
    await input.setValue('Renamed')
    await input.trigger('keyup.enter')
    expect(wrapper.emitted('update')).toEqual([[{ alias: 'Renamed' }]])
  })

  it('emits silence from the menu', async () => {
    const wrapper = mountWidget()
    await clickMenuItem(wrapper, 'widget-menu-silence')
    expect(wrapper.emitted('silence')).toHaveLength(1)
  })

  it('emits upgradeAll from the menu when updates are available', async () => {
    const wrapper = mountWidget(nodeFixture({ hasUpdateAvailable: true }))
    await clickMenuItem(wrapper, 'widget-menu-upgrade')
    expect(wrapper.emitted('upgradeAll')).toHaveLength(1)
  })

  it('emits delete from the menu', async () => {
    const wrapper = mountWidget()
    await clickMenuItem(wrapper, 'widget-menu-delete')
    expect(wrapper.emitted('delete')).toHaveLength(1)
  })

  it('shows deleting state in the menu while deleting', async () => {
    const wrapper = mountWidget(nodeFixture(), { isDeleting: true })
    await wrapper.find('[data-testid="meatball-trigger"]').trigger('click')
    expect(document.body.textContent).toContain('Deleting...')
  })

  it('emits openDetail from the footer button', async () => {
    const wrapper = mountWidget()
    await wrapper.findAll('button').find((b) => b.text().includes('Manage Node'))!.trigger('click')
    expect(wrapper.emitted('openDetail')).toHaveLength(1)
  })
})
