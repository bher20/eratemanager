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
import { getWaterProviders, getWaterRates } from '@/lib/api'
import { formatCurrency, formatDate } from '@/lib/utils'
import { Droplets, DollarSign, Calendar, Download, AlertCircle, Waves } from 'lucide-react'
import type { WaterRatesResponse } from '@/lib/types'

export function WaterPage() {
  const [searchParams] = useSearchParams()
  const urlProvider = searchParams.get('provider')
  const [selectedProvider, setSelectedProvider] = useState<string>(urlProvider || '')
  const [ratesData, setRatesData] = useState<WaterRatesResponse | null>(null)
  const [usage, setUsage] = useState<number>(5000) // Default 5000 gallons
  const [autoLoaded, setAutoLoaded] = useState<boolean>(false)

  const { data: providers, loading: loadingProviders } = useAsync(
    () => getWaterProviders(),
    []
  )

  const {
    mutate: loadRates,
    loading: loadingRates,
    error: ratesError,
  } = useMutation(getWaterRates)

  const providersList = providers || []

  // Auto-load rates if provider is specified in URL
  useEffect(() => {
    if (urlProvider && !autoLoaded && !loadingProviders && providersList.length > 0) {
      const providerExists = providersList.some(p => p.key === urlProvider)
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
  }, [urlProvider, autoLoaded, loadingProviders, providersList])

  const handleLoadRates = async () => {
    if (!selectedProvider) return
    try {
      const data = await loadRates(selectedProvider)
      setRatesData(data)
    } catch {
      // Error handled by mutation state
    }
  }

  // Calculate estimated monthly bill
  const calculateBill = () => {
    if (!ratesData) return null
    const water = ratesData.water
    const sewer = ratesData.sewer

    const waterCost = water.base_charge + (usage * water.use_rate)
    const sewerCost = sewer ? sewer.base_charge + (usage * sewer.use_rate) : 0
    const total = waterCost + sewerCost

    return { waterCost, sewerCost, total }
  }

  const bill = calculateBill()

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Water Rates</h1>
        <p className="mt-2 text-muted-foreground">
          View water and sewer rates from your utility providers
        </p>
      </div>

      {/* Provider Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Droplets className="h-5 w-5 text-blue-500" />
            Select Provider
          </CardTitle>
          <CardDescription>
            Choose a water utility provider to view their current rates
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loadingProviders ? (
            <LoadingOverlay message="Loading providers..." />
          ) : (
            <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
              <div className="flex-1">
                <Select
                  label="Water Provider"
                  value={selectedProvider}
                  onChange={(e) => setSelectedProvider(e.target.value)}
                  options={[
                    { value: '', label: 'Select a provider...' },
                    ...providersList.map((p) => ({
                      value: p.key,
                      label: p.name || p.key.toUpperCase(),
                    })),
                  ]}
                />
              </div>
              <Button
                onClick={handleLoadRates}
                disabled={!selectedProvider || loadingRates}
                loading={loadingRates}
              >
                <Download className="mr-2 h-4 w-4" />
                Load Rates
              </Button>
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
      {ratesData && (
        <div className="space-y-6 animate-slide-up">
          {/* Provider Info */}
          <div className="flex items-center gap-3">
            <Badge variant="success">Active</Badge>
            <span className="text-lg font-semibold">
              {ratesData.provider_name}
            </span>
            {ratesData.fetched_at && (
              <span className="text-sm text-muted-foreground">
                Last updated: {formatDate(ratesData.fetched_at)}
              </span>
            )}
          </div>

          {/* Water Rates */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Droplets className="h-5 w-5 text-blue-500" />
                Water Rates
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <StatCard
                  title="Base Charge"
                  value={formatCurrency(ratesData.water.base_charge)}
                  subtitle="Monthly minimum"
                  icon={<DollarSign className="h-5 w-5" />}
                />
                <StatCard
                  title="Usage Rate"
                  value={`${formatCurrency(ratesData.water.use_rate, 5)}/${ratesData.water.use_rate_unit}`}
                  subtitle="Per unit charge"
                  icon={<Droplets className="h-5 w-5" />}
                />
                {ratesData.water.effective_date && (
                  <StatCard
                    title="Effective Date"
                    value={ratesData.water.effective_date}
                    subtitle="Rate schedule"
                    icon={<Calendar className="h-5 w-5" />}
                  />
                )}
              </div>
            </CardContent>
          </Card>

          {/* Sewer Rates */}
          {ratesData.sewer && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Waves className="h-5 w-5 text-green-500" />
                  Sewer Rates
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  <StatCard
                    title="Base Charge"
                    value={formatCurrency(ratesData.sewer.base_charge)}
                    subtitle="Monthly minimum"
                    icon={<DollarSign className="h-5 w-5" />}
                  />
                  <StatCard
                    title="Usage Rate"
                    value={`${formatCurrency(ratesData.sewer.use_rate, 5)}/${ratesData.sewer.use_rate_unit}`}
                    subtitle="Per unit charge"
                    icon={<Waves className="h-5 w-5" />}
                  />
                  {ratesData.sewer.effective_date && (
                    <StatCard
                      title="Effective Date"
                      value={ratesData.sewer.effective_date}
                      subtitle="Rate schedule"
                      icon={<Calendar className="h-5 w-5" />}
                    />
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Bill Calculator */}
          <Card variant="gradient">
            <CardHeader>
              <CardTitle>Bill Calculator</CardTitle>
              <CardDescription>
                Estimate your monthly water bill based on usage
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-6">
                <div className="max-w-xs">
                  <label className="text-sm font-medium text-muted-foreground">
                    Monthly Usage (gallons)
                  </label>
                  <input
                    type="number"
                    value={usage}
                    onChange={(e) => setUsage(Number(e.target.value))}
                    className="mt-1.5 flex h-10 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
                  />
                </div>
                {bill && (
                  <div className="grid gap-4 sm:grid-cols-3">
                    <div className="rounded-lg border border-border bg-card p-4">
                      <p className="text-sm text-muted-foreground">Water</p>
                      <p className="mt-1 text-2xl font-bold text-blue-500">
                        {formatCurrency(bill.waterCost)}
                      </p>
                    </div>
                    {ratesData.sewer && (
                      <div className="rounded-lg border border-border bg-card p-4">
                        <p className="text-sm text-muted-foreground">Sewer</p>
                        <p className="mt-1 text-2xl font-bold text-green-500">
                          {formatCurrency(bill.sewerCost)}
                        </p>
                      </div>
                    )}
                    <div className="rounded-lg border border-primary/50 bg-primary/5 p-4">
                      <p className="text-sm text-muted-foreground">Total</p>
                      <p className="mt-1 text-2xl font-bold text-primary">
                        {formatCurrency(bill.total)}
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

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
            <Droplets className="h-12 w-12 text-muted-foreground/50" />
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
