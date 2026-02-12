const API_BASE = import.meta.env.VITE_API_URL || ''

function getToken() {
  return localStorage.getItem('token')
}

export async function api(endpoint, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
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
