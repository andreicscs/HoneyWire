import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import Sidebar from '../Sidebar.vue'
import { useAppStore } from '../../../stores/System/app'
import { useFleetStore } from '../../../stores/Fleet/fleet'

const routes = [
  { path: '/dashboard', name: 'dashboard', component: { template: '<div />' } },
  { path: '/fleet', name: 'fleet', component: { template: '<div />' } },
  { path: '/settings', name: 'settings', component: { template: '<div />' } },
]

function mountSidebar(path = '/dashboard') {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({ history: createMemoryHistory(), routes })
  router.push(path)
  return router.isReady().then(() => {
    const wrapper = mount(Sidebar, {
      attachTo: document.body,
      global: { plugins: [pinia, router] },
    })
    return { wrapper, router, appStore: useAppStore(), fleetStore: useFleetStore() }
  })
}

describe('Sidebar', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  it('renders navigation items', async () => {
    const { wrapper } = await mountSidebar()
    expect(wrapper.find('[data-testid="sidebar"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Dashboard')
    expect(wrapper.text()).toContain('Fleet Management')
    expect(wrapper.text()).toContain('Settings')
  })

  it('navigates to dashboard and clears selection', async () => {
    const { wrapper, router, fleetStore } = await mountSidebar('/fleet')
    fleetStore.selectTarget('node-1', null, false)
    await wrapper.find('[data-testid="nav-dashboard"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/dashboard')
    expect(fleetStore.selectedNodeId).toBeNull()
  })

  it('navigates to fleet and settings', async () => {
    const { wrapper, router } = await mountSidebar()
    await wrapper.find('[data-testid="nav-fleet"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/fleet')
    await wrapper.find('[data-testid="nav-settings"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/settings')
  })

  it('collapses and expands via the toggle', async () => {
    const { wrapper, appStore } = await mountSidebar()
    expect(appStore.sidebarOpen).toBe(true)
    await wrapper.find('[data-testid="sidebar-collapse"]').trigger('click')
    expect(appStore.sidebarOpen).toBe(false)
    await wrapper.find('[data-testid="sidebar-collapse"]').trigger('click')
    expect(appStore.sidebarOpen).toBe(true)
  })

  it('toggles the archive view', async () => {
    const { wrapper, appStore } = await mountSidebar()
    expect(wrapper.find('[data-testid="archive-toggle"]').text()).toContain('Event Archive')
    await wrapper.find('[data-testid="archive-toggle"]').trigger('click')
    expect(appStore.viewingArchive).toBe(true)
    expect(wrapper.find('[data-testid="archive-toggle"]').text()).toContain('Active Events')
  })

  it('alerts when the database is already empty', async () => {
    const fetchMock = vi.fn(async () => ({ ok: true, json: async () => ({ would_delete: 0 }) }))
    vi.stubGlobal('fetch', fetchMock)
    const alertMock = vi.fn()
    vi.stubGlobal('alert', alertMock)
    const { wrapper } = await mountSidebar()
    await wrapper.find('[data-testid="purge-button"]').trigger('click')
    await flushPromises()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(alertMock).toHaveBeenCalledWith('The database is already empty.')
  })

  it('purges after confirmation', async () => {
    const fetchMock = vi.fn(async (...args: any[]) => ({
      ok: true,
      json: async () => (String(args[0]).includes('dryrun=true') ? { would_delete: 5 } : {}),
    }))
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.stubGlobal('alert', vi.fn())
    const { wrapper } = await mountSidebar()
    await wrapper.find('[data-testid="purge-button"]').trigger('click')
    await flushPromises()
    // clearLogs issues two DELETEs; purgeEvents refires projections (GETs) on top
    const deletes = fetchMock.mock.calls.filter((c) => (c[1] as any)?.method === 'DELETE')
    expect(deletes).toHaveLength(2)
    expect(deletes[0][0]).toContain('dryrun=true')
    expect(deletes[1][0]).toContain('dryrun=false')
  })

  it('aborts purge when confirmation is dismissed', async () => {
    const fetchMock = vi.fn(async () => ({ ok: true, json: async () => ({ would_delete: 5 }) }))
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('confirm', vi.fn(() => false))
    vi.stubGlobal('alert', vi.fn())
    const { wrapper } = await mountSidebar()
    await wrapper.find('[data-testid="purge-button"]').trigger('click')
    await flushPromises()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})
