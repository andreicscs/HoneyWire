export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const handleResponse = async (res: Response): Promise<Response> => {
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    let message = text
    try {
      const parsed = JSON.parse(text)
      if (parsed && typeof parsed.error === 'string') {
        message = parsed.error
      } else if (parsed && typeof parsed.message === 'string') {
        message = parsed.message
      }
    } catch {
      // Not JSON, fallback to raw text
    }
    throw new ApiError(message || `HTTP ${res.status}`, res.status)
  }
  return res
}

export const api = {
  async get(url: string, options: RequestInit = {}): Promise<Response> {
    return handleResponse(await fetch(url, { ...options, method: 'GET', cache: 'no-store' }))
  },

  async post(url: string, body?: any, options: RequestInit = {}): Promise<Response> {
    return handleResponse(await fetch(url, {
      ...options,
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...options.headers },
      body: body ? JSON.stringify(body) : undefined,
    }))
  },

  async put(url: string, body?: any, options: RequestInit = {}): Promise<Response> {
    return handleResponse(await fetch(url, {
      ...options,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...options.headers },
      body: body ? JSON.stringify(body) : undefined,
    }))
  },

  async patch(url: string, body?: any, options: RequestInit = {}): Promise<Response> {
    return handleResponse(await fetch(url, {
      ...options,
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', ...options.headers },
      body: body ? JSON.stringify(body) : undefined,
    }))
  },

  async delete(url: string, options: RequestInit = {}): Promise<Response> {
    return handleResponse(await fetch(url, { ...options, method: 'DELETE' }))
  },

  async request(url: string, options: RequestInit = {}): Promise<Response> {
    return handleResponse(await fetch(url, options))
  },
}