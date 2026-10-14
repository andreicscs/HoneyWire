import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import NodeStatWidgets from '../NodeStatWidgets.vue'

const sensorFixture = (overrides = {}) => ({
  id: 'node-1:hw-sensor-demo',
  sensorId: 'hw-sensor-demo',
  nodeId: 'node-1',
  name: 'Demo Sensor',
  display: 'Demo Sensor',
  status: 'up',
  isSilenced: false,
  envVars: {},
  deployedVersion: '1.0.0',
  lastHeartbeat: null,
  ...overrides,
})

const activityFixture = () => [
  {
    id: 'e1',
    time: '2m ago',
    severity: 'high',
    eventTrigger: 'port_scan',
    source: '9.9.9.9',
    sensorId: 'hw-sensor-demo',
  },
]

function mountWidgets(nodeOverrides = {}, recentActivity: any[] = activityFixture()) {
  setActivePinia(createPinia())
  const node = { id: 'node-1', alias: 'Test Node', installedSensors: [sensorFixture()], ...nodeOverrides }
  return mount(NodeStatWidgets, {
    props: { node, recentActivity },
    attachTo: document.body,
    global: { plugins: [createPinia()] },
  })
}

describe('NodeStatWidgets', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders event volume with sensor names', () => {
    const wrapper = mountWidgets()
    expect(wrapper.text()).toContain('Event Volume (24h)')
    expect(wrapper.text()).toContain('Demo Sensor')
  })

  it('shows empty volume state without sensors', () => {
    const wrapper = mountWidgets({ installedSensors: [] }, [])
    expect(wrapper.text()).toContain('No events recorded.')
  })

  it('renders recent activity rows with readable trigger text', () => {
    const wrapper = mountWidgets()
    expect(wrapper.text()).toContain('Recent Activity')
    expect(wrapper.text()).toContain('high')
    expect(wrapper.text()).toContain('port scan')
    expect(wrapper.text()).toContain('9.9.9.9')
    expect(wrapper.text()).toContain('2m ago')
  })

  it('shows empty activity state without events', () => {
    const wrapper = mountWidgets({}, [])
    expect(wrapper.text()).toContain('No recent activity on this node.')
  })

  it('emits viewAllEvents from the View All button', async () => {
    const wrapper = mountWidgets()
    await wrapper.findAll('button').find((b) => b.text().includes('View All'))!.trigger('click')
    expect(wrapper.emitted('viewAllEvents')).toHaveLength(1)
  })
})
