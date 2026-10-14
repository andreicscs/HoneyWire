import { describe, it, expect } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import Dashboard from '../Dashboard.vue'
import TrafficFilters from '../../components/dashboard/TrafficFilters.vue'
import SeverityChart from '../../components/dashboard/SeverityChart.vue'
import UptimeHeatmap from '../../components/dashboard/UptimeHeatmap.vue'
import EventTable from '../../components/dashboard/EventTable.vue'
import ThreatVelocity from '../../components/dashboard/ThreatVelocity.vue'

// Shallow: the chart widgets need canvas/store wiring that belongs to their
// own specs. Dashboard owns only the composition, which is what's tested here.
function mountDashboard() {
  return shallowMount(Dashboard, { attachTo: document.body })
}

describe('Dashboard view (regression)', () => {
  it('renders the page container', () => {
    const wrapper = mountDashboard()
    expect(wrapper.find('[data-testid="dashboard-page"]').exists()).toBe(true)
  })

  it('composes all five widgets', () => {
    const wrapper = mountDashboard()
    for (const widget of [TrafficFilters, ThreatVelocity, SeverityChart, UptimeHeatmap, EventTable]) {
      expect(wrapper.findComponent(widget).exists()).toBe(true)
    }
  })

  it('renders the charts row with three sized slots', () => {
    const wrapper = mountDashboard()
    expect(wrapper.find('[data-testid="dashboard-charts"]').exists()).toBe(true)
    for (const testid of ['chart-velocity', 'chart-severity', 'chart-uptime']) {
      expect(wrapper.find(`[data-testid="${testid}"]`).exists()).toBe(true)
    }
  })

  it('nests each widget in its slot', () => {
    const wrapper = mountDashboard()
    expect(wrapper.find('[data-testid="chart-velocity"]').findComponent(ThreatVelocity).exists()).toBe(true)
    expect(wrapper.find('[data-testid="chart-severity"]').findComponent(SeverityChart).exists()).toBe(true)
    expect(wrapper.find('[data-testid="chart-uptime"]').findComponent(UptimeHeatmap).exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-events"]').findComponent(EventTable).exists()).toBe(true)
  })
})
