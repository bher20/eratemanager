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

export interface WaterProvider {
  key: string
  name: string
  type?: string
  htmlApiUrl?: string
  landingUrl?: string
  notes?: string
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

export interface User {
  id: string
  username: string
  first_name: string
  last_name: string
  email: string
  email_verified: boolean
  skip_email_verification: boolean
  onboarding_completed: boolean
  role: string
  created_at: string
  updated_at: string
}

export interface Token {
  id: string
  user_id: string
  name: string
  role: string
  created_at: string
  expires_at?: string
  last_used_at?: string
}

export interface Privilege {
  role: string
  resource: string
  action: string
}

export interface AuthStatus {
  initialized: boolean
  authenticated?: boolean
  user?: User
  role?: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface TokenResponse {
  token: Token
  access_token?: string
  token_value?: string
}

export interface EmailConfig {
  id?: string
  provider: 'smtp' | 'gmail' | 'sendgrid' | 'resend'
  host?: string
  port?: number
  username?: string
  password?: string
  from_address: string
  from_name: string
  api_key?: string
  encryption?: 'none' | 'ssl' | 'tls'
  enabled: boolean
  created_at?: string
  updated_at?: string
}
