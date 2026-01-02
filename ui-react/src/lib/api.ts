import type {
  ProvidersResponse,
  RatesResponse,
  WaterProvider,
  WaterRatesResponse,
  RefreshResponse,
} from './types'

const API_BASE = ''

export interface SystemInfo {
  storage: string
}

export async function getSystemInfo(): Promise<SystemInfo> {
  return fetchApi<SystemInfo>('/system/info')
}

async function fetchApi<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${url}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!response.ok) {
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
