// src/api/client.js

class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const handleResponse = async (res) => {
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new ApiError(text || `HTTP ${res.status}`, res.status)
  }
  return res
}

export const api = {
  async get(url) {
    return handleResponse(await fetch(url))
  },

  async post(url, body) {
    return handleResponse(await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }))
  },

  async patch(url, body) {
    return handleResponse(await fetch(url, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }))
  },

  async delete(url) {
    return handleResponse(await fetch(url, { method: 'DELETE' }))
  },

  // For requests with custom headers (e.g. Authorization) or non-JSON bodies
  async request(url, options = {}) {
    return handleResponse(await fetch(url, options))
  },
}