import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import Header from '../Header.vue'
import { useAppStore } from '../../../stores/System/app'
import { useEventsStore } from '../../../stores/Events/events'
import { api } from '../../../api/client'

vi.mock('../../../api/client', () => {
  const respond = async (data: any = {}) => ({ ok: true, json: async () => data, text: async () => '' })
  return {
    api: {
      get: vi.fn(async (url: string) => {
        if (url.startsWith('/api/v2/system/state')) return respond({ isArmed: false })
        if (url.startsWith('/api/v2/system/updates')) {
          return respond({
            update_available: false,
            latest_version: null,
            release_notes_url: null,
            wizard_update_available: true,
            latest_wizard_version: '9.9.9',
            wizard_release_notes_url: 'https://example.com/notes',
            acknowledged_wizard_release: null,
          })
        }
        if (url.startsWith('/api/v2/events/unread')) return respond({ count: 3 })
        if (url.startsWith('/api/v2/events')) {
          return respond([
            {
              id: 'e1', timestamp: new Date().toISOString(), nodeId: 'node-1',
              sensorId: 's1', severity: 'high', eventTrigger: 'x', source: 'y',
              target: 'z', isRead: false, isArchived: false,
            },
          ])
        }
        return respond()
      }),
      post: vi.fn(respond),
      put: vi.fn(respond),
      patch: vi.fn(respond),
      delete: vi.fn(respond),
      request: vi.fn(respond),
    },
  }
})

const routes = [
  { path: '/dashboard', name: 'dashboard', component: { template: '<div />' } },
  { path: '/fleet', name: 'fleet', component: { template: '<div />' } },
]

function mountHeader() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({ history: createMemoryHistory(), routes })
  router.push('/dashboard')
  return router.isReady().then(() => {
    const wrapper = mount(Header, {
      attachTo: document.body,
      global: { plugins: [pinia, router] },
    })
    return { wrapper, appStore: useAppStore(), eventsStore: useEventsStore() }
  })
}

describe('Header', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    document.documentElement.classList.remove('dark')
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    document.documentElement.classList.remove('dark')
    localStorage.clear()
    document.body.innerHTML = ''
  })

  it('renders branding and route name', async () => {
    const { wrapper } = await mountHeader()
    expect(wrapper.find('[data-testid="app-header"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('HoneyWire')
    expect(wrapper.text()).toContain('dashboard')
  })

  it('toggles armed state', async () => {
    const { wrapper, appStore } = await mountHeader()
    expect(wrapper.find('[data-testid="armed-button"]').text()).toContain('Armed')
    await wrapper.find('[data-testid="armed-button"]').trigger('click')
    await flushPromises()
    expect(appStore.isArmed).toBe(false)
    expect(wrapper.find('[data-testid="armed-button"]').text()).toContain('Passive')
  })

  it('marks all events read from the unread button', async () => {
    const { wrapper, eventsStore } = await mountHeader()
    await eventsStore.fetchEvents()
    await flushPromises()
    const unread = wrapper.find('[data-testid="unread-button"]')
    expect(unread.exists()).toBe(true)
    expect(unread.text()).toContain('Unread')
    await unread.trigger('click')
    await flushPromises()
    expect(api.patch).toHaveBeenCalledWith('/api/v2/events/read')
    expect(eventsStore.unreadCount).toBe(0)
    expect(wrapper.find('[data-testid="unread-button"]').isVisible()).toBe(false)
  })

  it('opens and closes the install modal', async () => {
    const { wrapper } = await mountHeader()
    await wrapper.find('[data-testid="install-button"]').trigger('click')
    expect(document.body.textContent).toContain('Install HoneyWire Wizard')
    expect(document.body.textContent).toContain('curl -fsSL https://get.honeywire.dev | bash')
    const closeBtn = [...document.body.querySelectorAll('button')].find((b) =>
      b.getAttribute('aria-label') === 'Close modal'
    )!
    closeBtn.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(document.body.textContent).not.toContain('Install HoneyWire Wizard')
  })

  it('shows and dismisses the wizard update notice', async () => {
    const { wrapper, appStore } = await mountHeader()
    await appStore.fetchSystemUpdates()
    await flushPromises()
    expect(wrapper.find('[data-testid="install-button"]').attributes('title')).toContain('9.9.9')
    await wrapper.find('[data-testid="install-button"]').trigger('click')
    expect(document.body.textContent).toContain('9.9.9')
    const dismiss = [...document.body.querySelectorAll('button')].find((b) =>
      b.getAttribute('aria-label') === 'Dismiss update notice'
    )!
    dismiss.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(api.post).toHaveBeenCalledWith(
      '/api/v2/system/wizard/acknowledge',
      expect.objectContaining({ version: '9.9.9' })
    )
    expect(document.body.textContent).not.toContain('9.9.9')
  })

  it('toggles the theme', async () => {
    const { wrapper } = await mountHeader()
    await wrapper.find('[data-testid="theme-button"]').trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
  })
})
