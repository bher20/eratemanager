import type {
  ProvidersResponse,
  RatesResponse,
  WaterProvider,
  WaterRatesResponse,
  RefreshResponse,
} from './types'

const API_BASE = ''

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

  return response.json()
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

export async function refreshWaterProvider(providerKey: string): Promise<RefreshResponse> {
  return fetchApi<RefreshResponse>(`/rates/water/${encodeURIComponent(providerKey)}/refresh`, {
    method: 'POST',
  })
}
