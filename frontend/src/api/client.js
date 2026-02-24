const API_BASE = import.meta.env.VITE_API_URL || ''

function getToken() {
  return localStorage.getItem('token')
}

export async function api(endpoint, options = {}) {
  const headers = { ...options.headers }
  const isFormData = options.body instanceof FormData
  if (!isFormData && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }
  const token = getToken()
  if (token) {
    headers['Authorization'] = token.startsWith('Bearer ') ? token : `Bearer ${token}`
  }
  const res = await fetch(`${API_BASE}${endpoint}`, { ...options, headers })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`)
  }
  return data
}

export const auth = {
  register: (body) => api('/api/v1/auth/register', { method: 'POST', body: JSON.stringify(body) }),
  login: (body) => api('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(body) }),
  me: () => api('/api/v1/auth/me'),
}

export const profile = {
  get: () => api('/api/v1/profile/me'),
  update: (body) => api('/api/v1/profile/me', { method: 'PUT', body: JSON.stringify(body) }),
}

export const users = {
  search: (params) => {
    const q = new URLSearchParams(params).toString()
    return api(`/api/v1/users${q ? '?' + q : ''}`)
  },
  getById: (id) => api(`/api/v1/users/${id}`),
  getPhotos: (id) => api(`/api/v1/users/${id}/photos`),
  like: (id) => api(`/api/v1/users/${id}/like`, { method: 'POST' }),
  unlike: (id) => api(`/api/v1/users/${id}/like`, { method: 'DELETE' }),
}

export const matches = {
  list: (params = {}) => {
    const q = new URLSearchParams(params).toString()
    return api(`/api/v1/matches${q ? '?' + q : ''}`)
  },
}

export const chat = {
  listMessages: (userId, params = {}) => {
    const q = new URLSearchParams(params).toString()
    return api(`/api/v1/users/${userId}/messages${q ? '?' + q : ''}`)
  },
  sendMessage: (userId, content) =>
    api(`/api/v1/users/${userId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
  markRead: (userId) =>
    api(`/api/v1/users/${userId}/messages/read`, {
      method: 'PATCH',
      body: JSON.stringify({}),
    }),
}

export const notifications = {
  list: (params = {}) => {
    const q = new URLSearchParams(params).toString()
    return api(`/api/v1/notifications${q ? '?' + q : ''}`)
  },
  markAllRead: () =>
    api('/api/v1/notifications/read-all', {
      method: 'PATCH',
      body: JSON.stringify({}),
    }),
}

export const presence = {
  get: (userId) => api(`/api/v1/presence/${userId}`),
}

export const photos = {
  listMe: () => api('/api/v1/photos/me'),
  upload: (file) => {
    const body = new FormData()
    body.append('file', file)
    return api('/api/v1/photos', { method: 'POST', body })
  },
  remove: (id) => api(`/api/v1/photos/${id}`, { method: 'DELETE' }),
  setPrimary: (id) => api(`/api/v1/photos/${id}/primary`, { method: 'PATCH', body: JSON.stringify({}) }),
}

export function wsChatUrl() {
  const token = getToken()
  if (!token) return null
  const normalized = token.startsWith('Bearer ') ? token.slice(7) : token
  const base = API_BASE || window.location.origin
  const wsBase = base
    .replace(/^http:\/\//, 'ws://')
    .replace(/^https:\/\//, 'wss://')
    .replace(/\/$/, '')
  return `${wsBase}/api/v1/ws/chat?token=${encodeURIComponent(normalized)}`
}
