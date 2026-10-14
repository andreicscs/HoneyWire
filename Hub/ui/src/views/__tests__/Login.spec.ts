import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Login from '../Login.vue'

const loginMode = vi.hoisted(() => ({ value: 'success' as 'success' | 'invalid' | 'ratelimited' | 'pending' }))

vi.mock('../../api/client', () => {
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
      get: vi.fn(respond),
      post: vi.fn(async (url: string) => {
        if (url === '/login') {
          if (loginMode.value === 'success') return respond()
          if (loginMode.value === 'pending') return new Promise(() => {})
          if (loginMode.value === 'ratelimited') throw new FakeApiError('Too many attempts', 429)
          throw new FakeApiError('Invalid credentials', 401)
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

function mountLogin() {
  const pinia = createPinia()
  setActivePinia(pinia)
  return mount(Login, {
    attachTo: document.body,
    global: { plugins: [pinia] },
  })
}

async function submitPassword(wrapper: any, password: string) {
  await wrapper.find('[data-testid="login-password"] input').setValue(password)
  await wrapper.find('form').trigger('submit')
  await flushPromises()
}

describe('Login view (regression)', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    loginMode.value = 'success'
    vi.clearAllMocks()
  })

  it('renders branding, key input and submit', () => {
    const wrapper = mountLogin()
    expect(wrapper.text()).toContain('HoneyWire Sentinel')
    expect(wrapper.text()).toContain('Authentication Key')
    const submit = wrapper.find('[data-testid="login-submit"]')
    expect(submit.exists()).toBe(true)
    expect(submit.text()).toContain('Sign in')
  })

  it('emits login-success on valid key', async () => {
    const wrapper = mountLogin()
    await submitPassword(wrapper, 'correct-key')
    expect(wrapper.emitted('login-success')).toHaveLength(1)
    expect(wrapper.find('[data-testid="login-error"]').exists()).toBe(false)
  })

  it('shows access-denied and clears the key on invalid key', async () => {
    loginMode.value = 'invalid'
    const wrapper = mountLogin()
    await submitPassword(wrapper, 'wrong-key')
    expect(wrapper.emitted('login-success')).toBeUndefined()
    const error = wrapper.find('[data-testid="login-error"]')
    expect(error.exists()).toBe(true)
    expect(error.text()).toContain('Access Denied')
    expect((wrapper.find('[data-testid="login-password"] input').element as HTMLInputElement).value).toBe('')
  })

  it('shows rate-limit notice on too many attempts', async () => {
    loginMode.value = 'ratelimited'
    const wrapper = mountLogin()
    await submitPassword(wrapper, 'any-key')
    const notice = wrapper.find('[data-testid="login-rate-limited"]')
    expect(notice.exists()).toBe(true)
    expect(notice.text()).toContain('Too many attempts')
  })

  it('disables submit while authenticating', async () => {
    loginMode.value = 'pending'
    const wrapper = mountLogin()
    await wrapper.find('[data-testid="login-password"] input').setValue('key')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    const submit = wrapper.find('[data-testid="login-submit"]')
    expect(submit.text()).toContain('Authenticating...')
    expect(submit.attributes('disabled')).toBeDefined()
  })
})
