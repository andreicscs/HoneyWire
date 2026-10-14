import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import NodeSensorGrid from '../NodeSensorGrid.vue'

enableAutoUnmount(afterEach)

const sensorFixture = () => ({
  id: 'node-1:hw-sensor-demo',
  sensorId: 'hw-sensor-demo',
  nodeId: 'node-1',
  name: 'Demo Sensor',
  display: 'Demo Sensor',
  status: 'up',
  isSilenced: false,
  envVars: {},
  icon: 'M12 2v20',
  osi: 'Network',
})

function mountGrid(sensors: any[] = [sensorFixture()]) {
  return mount(NodeSensorGrid, {
    props: { sensors },
    attachTo: document.body,
  })
}

function closeMenus() {
  document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

async function openMenu(wrapper: any) {
  await wrapper.find('[data-testid="meatball-trigger"]').trigger('click')
}

function menuItem(testid: string) {
  return document.body.querySelector(`[data-testid="${testid}"]`)!
}

describe('NodeSensorGrid', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  afterEach(() => {
    closeMenus()
    document.body.innerHTML = ''
  })

  it('renders deployed sensors', () => {
    const wrapper = mountGrid()
    expect(wrapper.text()).toContain('Deployed Sensors')
    expect(wrapper.text()).toContain('Demo Sensor')
  })

  it('shows the empty state without sensors', () => {
    const wrapper = mountGrid([])
    expect(wrapper.text()).toContain('No sensors deployed')
    expect(wrapper.text()).toContain('Select a sensor from the catalog below.')
  })

  it('emits edit from the menu', async () => {
    const wrapper = mountGrid()
    await openMenu(wrapper)
    menuItem('sensor-menu-edit').dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(wrapper.emitted('edit')?.[0]).toEqual([sensorFixture()])
  })

  it('emits remove from the menu', async () => {
    const wrapper = mountGrid()
    await openMenu(wrapper)
    menuItem('sensor-menu-remove').dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(wrapper.emitted('remove')?.[0]).toEqual([sensorFixture()])
  })
})
