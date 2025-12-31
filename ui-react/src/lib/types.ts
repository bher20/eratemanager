// API Types for eRateManager

export interface Provider {
  key: string
  name: string
  landing_url?: string
  default_pdf_path?: string
  notes?: string
  type?: 'electric' | 'water'
}

export interface ProvidersResponse {
  providers: Provider[]
}

export interface ResidentialRates {
  energy_rate_usd_per_kwh?: number
  tva_fuel_rate_usd_per_kwh?: number
  customer_charge_monthly_usd?: number
  effective_date?: string
}

export interface RatesResponse {
  provider: string
  fetched_at: string
  rates: {
    residential_standard?: ResidentialRates
  }
  source_url?: string
}

export interface WaterRateDetails {
  meter_sizes?: Record<string, number>
  default_meter_size?: string
  base_charge: number
  use_rate: number
  use_rate_unit: string
  effective_date?: string
}

export interface SewerRateDetails {
  base_charge: number
  use_rate: number
  use_rate_unit: string
  effective_date?: string
}

export interface WaterRatesResponse {
  provider_key: string
  provider_name: string
  fetched_at: string
  water: WaterRateDetails
  sewer?: SewerRateDetails
}

export interface WaterProvidersResponse {
  providers: Array<{
    key: string
    name: string
  }>
}

export interface RefreshResponse {
  status: string
  pdf_url?: string
  error?: string
}

export interface ApiError {
  message: string
  status?: number
}
