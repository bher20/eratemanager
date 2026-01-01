import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Select,
  Button,
  StatCard,
  LoadingOverlay,
  Badge,
} from '@/components'
import { useAsync, useMutation } from '@/hooks'
import { getProviders, getResidentialRates, refreshProvider } from '@/lib/api'
import { formatCurrency, formatRate, formatDate } from '@/lib/utils'
import { Zap, DollarSign, Calendar, RefreshCw, Download, AlertCircle } from 'lucide-react'
import type { RatesResponse } from '@/lib/types'

export function ElectricPage() {
  const [searchParams] = useSearchParams()
  const urlProvider = searchParams.get('provider')
  const [selectedProvider, setSelectedProvider] = useState<string>(urlProvider || '')
  const [ratesData, setRatesData] = useState<RatesResponse | null>(null)
  const [autoLoaded, setAutoLoaded] = useState<boolean>(false)

  const { data: providersData, loading: loadingProviders } = useAsync(
    () => getProviders(),
    []
  )

  const {
    mutate: loadRates,
    loading: loadingRates,
    error: ratesError,
  } = useMutation(getResidentialRates)

  const {
    mutate: doRefresh,
    loading: refreshing,
  } = useMutation(refreshProvider)

  // Filter for electric providers only
  const providers = (providersData?.providers || []).filter(
    (p) => p.type === 'electric' || !p.type
  )

  // Auto-load rates if provider is specified in URL
  useEffect(() => {
    if (urlProvider && !autoLoaded && !loadingProviders && providers.length > 0) {
      const providerExists = providers.some(p => p.key === urlProvider)
      if (providerExists) {
        setSelectedProvider(urlProvider)
        loadRates(urlProvider).then(data => {
          setRatesData(data)
          setAutoLoaded(true)
        }).catch(() => {
          setAutoLoaded(true)
        })
      }
    }
  }, [urlProvider, autoLoaded, loadingProviders, providers])

  const handleLoadRates = async () => {
    if (!selectedProvider) return
    try {
      const data = await loadRates(selectedProvider)
      setRatesData(data)
    } catch {
      // Error handled by mutation state
    }
  }

  const handleRefresh = async () => {
    if (!selectedProvider) return
    try {
      await doRefresh(selectedProvider)
      // Reload rates after refresh
      const data = await loadRates(selectedProvider)
      setRatesData(data)
    } catch {
      // Error handled by mutation state
    }
  }

  const rates = ratesData?.rates?.residential_standard

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Electric Rates</h1>
        <p className="mt-2 text-muted-foreground">
          View and manage electricity rates from your utility providers
        </p>
      </div>

      {/* Provider Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Zap className="h-5 w-5 text-yellow-500" />
            Select Provider
          </CardTitle>
          <CardDescription>
            Choose an electric utility provider to view their current rates
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loadingProviders ? (
            <LoadingOverlay message="Loading providers..." />
          ) : (
            <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
              <div className="flex-1">
                <Select
                  label="Electric Provider"
                  value={selectedProvider}
                  onChange={(e) => setSelectedProvider(e.target.value)}
                  options={[
                    { value: '', label: 'Select a provider...' },
                    ...providers.map((p) => ({
                      value: p.key,
                      label: p.name || p.key.toUpperCase(),
                    })),
                  ]}
                />
              </div>
              <div className="flex gap-2">
                <Button
                  onClick={handleLoadRates}
                  disabled={!selectedProvider || loadingRates}
                  loading={loadingRates}
                >
                  <Download className="mr-2 h-4 w-4" />
                  Load Rates
                </Button>
                <Button
                  variant="outline"
                  onClick={handleRefresh}
                  disabled={!selectedProvider || refreshing}
                  loading={refreshing}
                >
                  <RefreshCw className="mr-2 h-4 w-4" />
                  Refresh
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Error State */}
      {ratesError && (
        <Card className="border-destructive/50 bg-destructive/5">
          <CardContent className="flex items-center gap-3 py-4">
            <AlertCircle className="h-5 w-5 text-destructive" />
            <p className="text-sm text-destructive">
              Failed to load rates: {ratesError.message}
            </p>
          </CardContent>
        </Card>
      )}

      {/* Rates Display */}
      {ratesData && rates && (
        <div className="space-y-6 animate-slide-up">
          {/* Provider Info */}
          <div className="flex items-center gap-3">
            <Badge variant="success">Active</Badge>
            <span className="text-lg font-semibold">
              {ratesData.provider?.toUpperCase()}
            </span>
            {ratesData.fetched_at && (
              <span className="text-sm text-muted-foreground">
                Last updated: {formatDate(ratesData.fetched_at)}
              </span>
            )}
          </div>

          {/* Stats Grid */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {rates.customer_charge_monthly_usd != null && (
              <StatCard
                title="Customer Charge"
                value={formatCurrency(rates.customer_charge_monthly_usd)}
                subtitle="Monthly base fee"
                icon={<DollarSign className="h-5 w-5" />}
              />
            )}
            {rates.energy_rate_usd_per_kwh != null && (
              <StatCard
                title="Energy Rate"
                value={formatRate(rates.energy_rate_usd_per_kwh)}
                subtitle="Per kilowatt-hour"
                icon={<Zap className="h-5 w-5" />}
              />
            )}
            {rates.tva_fuel_rate_usd_per_kwh != null && (
              <StatCard
                title="TVA Fuel Rate"
                value={formatRate(rates.tva_fuel_rate_usd_per_kwh)}
                subtitle="Fuel cost adjustment"
                icon={<Zap className="h-5 w-5" />}
              />
            )}
            {rates.energy_rate_usd_per_kwh != null && rates.tva_fuel_rate_usd_per_kwh != null && (
              <StatCard
                title="Total Rate"
                value={formatRate(
                  rates.energy_rate_usd_per_kwh + rates.tva_fuel_rate_usd_per_kwh
                )}
                subtitle="Energy + Fuel"
                icon={<DollarSign className="h-5 w-5" />}
              />
            )}
            {rates.effective_date && (
              <StatCard
                title="Effective Date"
                value={rates.effective_date}
                subtitle="Rate schedule date"
                icon={<Calendar className="h-5 w-5" />}
              />
            )}
          </div>

          {/* Raw Data */}
          <Card>
            <CardHeader>
              <CardTitle>Raw Rate Data</CardTitle>
              <CardDescription>
                Complete rate information from the API
              </CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="overflow-auto rounded-lg bg-muted/50 p-4 text-xs font-mono">
                {JSON.stringify(ratesData, null, 2)}
              </pre>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Empty State */}
      {!ratesData && !loadingRates && selectedProvider && (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Zap className="h-12 w-12 text-muted-foreground/50" />
            <p className="mt-4 text-lg font-medium">No rates loaded</p>
            <p className="text-sm text-muted-foreground">
              Click "Load Rates" to fetch the current rates
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
