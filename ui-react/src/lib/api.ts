import type {
  ProvidersResponse,
  RatesResponse,
  WaterProvider,
  WaterRatesResponse,
  RefreshResponse,
  User,
  Privilege,
  AuthStatus,
  LoginResponse,
  Token,
  TokenResponse
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
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options?.headers,
  }

  const response = await fetch(`${API_BASE}${url}`, {
    ...options,
    headers,
  })

  if (response.status === 401) {
    localStorage.removeItem('token')
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'An error occurred' }))
    throw new Error(error.message || 'An error occurred')
  }

  return response.json()
}

export async function getProviders(): Promise<ProvidersResponse> {
  return fetchApi<ProvidersResponse>('/providers')
}

export async function getRates(provider: string): Promise<RatesResponse> {
  return fetchApi<RatesResponse>(`/rates/electric/${provider}`)
}
export const getResidentialRates = getRates

export async function getWaterProviders(): Promise<WaterProvider[]> {
  const response = await fetchApi<{ providers: WaterProvider[] }>('/rates/water/providers')
  return response.providers
}

export async function getWaterRates(provider: string): Promise<WaterRatesResponse> {
  return fetchApi<WaterRatesResponse>(`/rates/water/${provider}`)
}

export async function refreshRates(provider: string, type: 'electric' | 'water' = 'electric'): Promise<RefreshResponse> {
  return fetchApi<RefreshResponse>(`/rates/${type}/${provider}/refresh`, {
    method: 'POST',
  })
}
export const refreshProvider = refreshRates

export async function login(username: string, password: string): Promise<LoginResponse> {
  const response = await fetchApi<LoginResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  localStorage.setItem('token', response.token)
  return response
}

export async function getAuthStatus(): Promise<AuthStatus> {
  return fetchApi<AuthStatus>('/auth/status')
}

export async function createAdmin(username: string, password: string): Promise<void> {
  await fetchApi('/auth/setup', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
}
export const setupAdmin = createAdmin

export async function getTokens(): Promise<Token[]> {
  return fetchApi<Token[]>('/auth/tokens')
}

export async function createToken(name: string, role: string): Promise<TokenResponse> {
  return fetchApi<TokenResponse>('/auth/tokens', {
    method: 'POST',
    body: JSON.stringify({ name, role }),
  })
}

export async function deleteToken(id: string): Promise<void> {
  await fetchApi(`/auth/tokens/${id}`, {
    method: 'DELETE',
  })
}

export async function getUsers(): Promise<User[]> {
  return fetchApi<User[]>('/auth/users')
}

export async function getRoles(): Promise<string[]> {
  return fetchApi<string[]>('/auth/roles')
}

export async function getPrivileges(): Promise<Privilege[]> {
  return fetchApi<Privilege[]>('/auth/privileges')
}

export async function getRefreshInterval(): Promise<{ interval: number }> {
  return fetchApi<{ interval: number }>('/system/refresh-interval')
}

export async function setRefreshInterval(interval: number): Promise<void> {
  await fetchApi('/system/refresh-interval', {
    method: 'POST',
    body: JSON.stringify({ interval }),
  })
}
