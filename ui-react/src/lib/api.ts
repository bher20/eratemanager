import type {
  ProvidersResponse,
  RatesResponse,
  WaterProvider,
  WaterRatesResponse,
  RefreshResponse,
  User,
  Token,
} from './types'

const API_BASE = ''

export interface SystemInfo {
  storage: string
}

export async function getSystemInfo(): Promise<SystemInfo> {
  return fetchApi<SystemInfo>('/system/info')
}

async function fetchApi<T>(url: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const headers = new Headers(options?.headers)
  
  if (!headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(`${API_BASE}${url}`, {
    ...options,
    headers,
  })

  if (!response.ok) {
    if (response.status === 401) {
      // Optional: Redirect to login or clear storage
      // window.location.href = '/login'
    }
    const errorText = await response.text()
    throw new Error(errorText || `HTTP ${response.status}`)
  }

  const text = await response.text()
  return text ? JSON.parse(text) : ({} as T)
}

// Electric Providers
export async function getProviders(): Promise<ProvidersResponse> {
  return fetchApi<ProvidersResponse>('/providers')
}

export async function getResidentialRates(providerKey: string): Promise<RatesResponse> {
  return fetchApi<RatesResponse>(`/rates/electric/${encodeURIComponent(providerKey)}/residential`)
}

export async function refreshProvider(providerKey: string): Promise<RefreshResponse> {
  return fetchApi<RefreshResponse>(`/rates/electric/${encodeURIComponent(providerKey)}/refresh`, {
    method: 'POST',
  })
}

// Water Providers
export async function getWaterProviders(): Promise<WaterProvider[]> {
  return fetchApi<WaterProvider[]>('/rates/water/providers')
}

export async function getWaterRates(providerKey: string): Promise<WaterRatesResponse> {
  return fetchApi<WaterRatesResponse>(`/rates/water/${encodeURIComponent(providerKey)}`)
}

// Settings
export async function getRefreshInterval(): Promise<{ interval: string }> {
  return fetchApi<{ interval: string }>('/settings/refresh-interval')
}

export async function setRefreshInterval(interval: string): Promise<void> {
  return fetchApi<void>('/settings/refresh-interval', {
    method: 'POST',
    body: JSON.stringify({ interval }),
  })
}

export async function refreshWaterProvider(providerKey: string): Promise<RefreshResponse> {
  return fetchApi<RefreshResponse>(`/rates/water/${encodeURIComponent(providerKey)}/refresh`, {
    method: 'POST',
  })
}

// Auth
export async function login(username: string, password: string): Promise<{ token: string; user: User }> {
  return fetchApi<{ token: string; user: User }>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}

export async function getTokens(): Promise<Token[]> {
  return fetchApi<Token[]>('/api/auth/tokens')
}

export async function createToken(name: string, role: string): Promise<{ token: Token; token_value: string }> {
  return fetchApi<{ token: Token; token_value: string }>('/api/auth/tokens', {
    method: 'POST',
    body: JSON.stringify({ name, role }),
  })
}

export async function deleteToken(id: string): Promise<void> {
  return fetchApi<void>(`/api/auth/tokens/${id}`, {
    method: 'DELETE',
  })
}

export async function getAuthStatus(): Promise<{ initialized: boolean }> {
  return fetchApi<{ initialized: boolean }>('/api/auth/status')
}

export async function setupAdmin(username: string, password: string): Promise<User> {
  return fetchApi<User>('/api/auth/setup', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}
