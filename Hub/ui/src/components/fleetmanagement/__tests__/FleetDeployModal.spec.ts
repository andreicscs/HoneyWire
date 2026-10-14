import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import FleetDeployModal from '../FleetDeployModal.vue'

const createMode = vi.hoisted(() => ({ value: 'success' as 'success' | 'failure' }))

vi.mock('../../../api/client', () => {
  class FakeApiError extends Error {
    status: number
    constructor(message: string, status: number) {
      super(message)
      this.status = status
    }
  }
  const respond = async () => ({ ok: true, json: async () => ({}), text: async () => '' })
  return {
    api: {
      get: vi.fn(async () => ({ ok: true, json: async () => [], text: async () => '' })),
      post: vi.fn(async (url: string) => {
        if (url === '/api/v2/nodes' && createMode.value === 'failure') {
          throw new FakeApiError('Taken', 409)
        }
        if (url === '/api/v2/nodes') {
          return { ok: true, json: async () => ({ nodeId: 'node-9', apiKey: 'fresh-key' }), text: async () => '' }
        }
        return respond()
      }),
      put: vi.fn(respond),
      patch: vi.fn(respond),
      delete: vi.fn(respond),
      request: vi.fn(respond),
    },
  }
})

function mountModal(show = true) {
  const pinia = createPinia()
  setActivePinia(pinia)
  return mount(FleetDeployModal, {
    props: { show },
    attachTo: document.body,
    global: { plugins: [pinia] },
  })
}

function bodyButton(text: string) {
  return [...document.body.querySelectorAll('button')].find((b) => b.textContent?.includes(text))!
}

async function fillAlias(value: string) {
  const input = document.body.querySelector('[data-testid="deploy-alias-input"] input') as HTMLInputElement
  input.value = value
  input.dispatchEvent(new Event('input'))
  await flushPromises()
}

describe('FleetDeployModal', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    createMode.value = 'success'
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  it('renders nothing when closed', () => {
    mountModal(false)
    expect(document.body.textContent).not.toContain('Deploy New Node')
  })

  it('renders the create form when open', () => {
    mountModal()
    expect(document.body.textContent).toContain('Deploy New Node')
    expect(document.body.textContent).toContain('Node Alias')
    expect(document.body.querySelector('[data-testid="deploy-alias-input"]')).not.toBeNull()
  })

  it('disables create without an alias', () => {
    const wrapper = mountModal()
    const create = [...document.body.querySelectorAll('button')].find((b) => b.textContent === 'Create Node')!
    expect(create.hasAttribute('disabled')).toBe(true)
    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('creates the node and shows install instructions', async () => {
    mountModal()
    await fillAlias('Edge Node')
    bodyButton('Create Node').dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    await flushPromises()
    expect(document.body.textContent).toContain('Node created successfully.')
    expect(document.body.textContent).toContain('fresh-key')
    expect(document.body.textContent).toContain('Wizard Installation Command')
  })

  it('stays on the form when creation fails', async () => {
    createMode.value = 'failure'
    vi.stubGlobal('alert', vi.fn())
    mountModal()
    await fillAlias('Edge Node')
    bodyButton('Create Node').dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    await flushPromises()
    expect(document.body.textContent).not.toContain('Node created successfully.')
    expect(document.body.textContent).toContain('Node Alias')
  })

  it('emits close from Cancel', async () => {
    const wrapper = mountModal()
    bodyButton('Cancel').dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('emits close from Done after creation', async () => {
    const wrapper = mountModal()
    await fillAlias('Edge Node')
    bodyButton('Create Node').dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    await flushPromises()
    bodyButton('Done').dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
