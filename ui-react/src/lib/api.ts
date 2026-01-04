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
  TokenResponse,
  EmailConfig
} from './types'

const API_BASE = ''

export interface SystemInfo {
  storage: string
  version: string
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
    const text = await response.text()
    let error = { message: 'An error occurred' }
    if (text) {
      try {
        error = JSON.parse(text)
      } catch {
        error = { message: text }
      }
    }
    throw new Error(error.message || 'An error occurred')
  }

  // Handle empty responses (e.g., 200 OK with no body)
  const text = await response.text()
  if (!text || text.trim() === '') {
    return undefined as T
  }
  
  try {
    return JSON.parse(text)
  } catch (e) {
    console.error('Failed to parse JSON response:', text)
    throw new Error('Invalid JSON response from server')
  }
}

export async function getProviders(): Promise<ProvidersResponse> {
  return fetchApi<ProvidersResponse>('/providers')
}

export async function getRates(provider: string): Promise<RatesResponse> {
  return fetchApi<RatesResponse>(`/rates/electric/${provider}/residential`)
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

export async function refreshWaterProvider(provider: string): Promise<RefreshResponse> {
  return refreshRates(provider, 'water')
}

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

export async function createToken(name: string, role: string, expiresIn?: string): Promise<TokenResponse> {
  return fetchApi<TokenResponse>('/auth/tokens', {
    method: 'POST',
    body: JSON.stringify({ name, role, expires_in: expiresIn || 'never' }),
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

export async function getRefreshInterval(): Promise<{ interval: string }> {
  return fetchApi<{ interval: string }>('/settings/refresh-interval')
}

export async function setRefreshInterval(interval: string): Promise<void> {
  await fetchApi('/settings/refresh-interval', {
    method: 'POST',
    body: JSON.stringify({ interval }),
  })
}

export async function createUser(username: string, password: string, role: string): Promise<User> {
  return fetchApi<User>('/auth/users', {
    method: 'POST',
    body: JSON.stringify({ username, password, role }),
  })
}

export async function updateUser(id: string, role: string): Promise<User> {
  return fetchApi<User>(`/auth/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  })
}

export async function addPolicy(role: string, resource: string, action: string): Promise<void> {
  await fetchApi('/auth/privileges', {
    method: 'POST',
    body: JSON.stringify({ role, resource, action }),
  })
}

export async function removePolicy(role: string, resource: string, action: string): Promise<void> {
  await fetchApi('/auth/privileges', {
    method: 'DELETE',
    body: JSON.stringify({ role, resource, action }),
  })
}


export async function createRole(role: string, policies: { resource: string; action: string }[] = []): Promise<void> {
  await fetchApi('/auth/roles', {
    method: 'POST',
    body: JSON.stringify({ role, policies }),
  })
}

export async function getEmailConfig(): Promise<EmailConfig> {
  return fetchApi<EmailConfig>('/api/v1/settings/email')
}

export async function saveEmailConfig(config: EmailConfig): Promise<void> {
  return fetchApi<void>('/api/v1/settings/email', {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

export async function testEmailConfig(config: EmailConfig, to: string): Promise<void> {
  return fetchApi<void>('/api/v1/settings/email/test', {
    method: 'POST',
    body: JSON.stringify({ config, to }),
  })
}
