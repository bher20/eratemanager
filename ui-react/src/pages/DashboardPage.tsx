import { Link } from 'react-router-dom'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  StatCard,
  LoadingOverlay,
  Badge,
} from '@/components'
import { useAsync } from '@/hooks'
import { getProviders, getWaterProviders } from '@/lib/api'
import { Zap, Droplets, ArrowRight, Activity, Server, Database } from 'lucide-react'

export function DashboardPage() {
  const { data: electricProviders, loading: loadingElectric } = useAsync(
    () => getProviders(),
    []
  )

  const { data: waterProviders, loading: loadingWater } = useAsync(
    () => getWaterProviders(),
    []
  )

  const loading = loadingElectric || loadingWater

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="mt-2 text-muted-foreground">
          Welcome to eRateManager - your utility rate tracking dashboard
        </p>
      </div>

      {loading ? (
        <LoadingOverlay message="Loading dashboard..." />
      ) : (
        <>
          {/* Stats Overview */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              title="Electric Providers"
              value={String(electricProviders?.providers?.length || 0)}
              subtitle="Available utilities"
              icon={<Zap className="h-5 w-5" />}
            />
            <StatCard
              title="Water Providers"
              value={String(waterProviders?.providers?.length || 0)}
              subtitle="Available utilities"
              icon={<Droplets className="h-5 w-5" />}
            />
            <StatCard
              title="API Status"
              value="Online"
              subtitle="All systems operational"
              icon={<Activity className="h-5 w-5" />}
            />
            <StatCard
              title="Data Source"
              value="Live"
              subtitle="Real-time rate data"
              icon={<Database className="h-5 w-5" />}
            />
          </div>

          {/* Quick Access Cards */}
          <div className="grid gap-6 md:grid-cols-2">
            {/* Electric Rates Card */}
            <Card className="group relative overflow-hidden transition-all hover:border-yellow-500/50 hover:shadow-lg hover:shadow-yellow-500/5">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div className="rounded-lg bg-yellow-500/10 p-2.5">
                    <Zap className="h-6 w-6 text-yellow-500" />
                  </div>
                  <Badge>
                    {electricProviders?.providers?.length || 0} providers
                  </Badge>
                </div>
                <CardTitle className="mt-4">Electric Rates</CardTitle>
                <CardDescription>
                  View electricity rates from TVA distributors including CEMC, NES, and KUB
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {electricProviders?.providers?.slice(0, 3).map((provider) => (
                    <div
                      key={provider.key}
                      className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2 text-sm"
                    >
                      <span>{provider.name || provider.key.toUpperCase()}</span>
                      <span className="text-muted-foreground">→</span>
                    </div>
                  ))}
                </div>
                <Link
                  to="/electric"
                  className="mt-4 inline-flex items-center gap-2 text-sm font-medium text-primary transition-colors hover:text-primary/80"
                >
                  View All Electric Rates
                  <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
                </Link>
              </CardContent>
              {/* Decorative gradient */}
              <div className="absolute inset-x-0 bottom-0 h-1 bg-gradient-to-r from-yellow-500/0 via-yellow-500/50 to-yellow-500/0 opacity-0 transition-opacity group-hover:opacity-100" />
            </Card>

            {/* Water Rates Card */}
            <Card className="group relative overflow-hidden transition-all hover:border-blue-500/50 hover:shadow-lg hover:shadow-blue-500/5">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div className="rounded-lg bg-blue-500/10 p-2.5">
                    <Droplets className="h-6 w-6 text-blue-500" />
                  </div>
                  <Badge>
                    {waterProviders?.providers?.length || 0} providers
                  </Badge>
                </div>
                <CardTitle className="mt-4">Water Rates</CardTitle>
                <CardDescription>
                  View water and sewer rates from your local utility district
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {waterProviders?.providers?.slice(0, 3).map((provider) => (
                    <div
                      key={provider.key}
                      className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2 text-sm"
                    >
                      <span>{provider.name || provider.key.toUpperCase()}</span>
                      <span className="text-muted-foreground">→</span>
                    </div>
                  ))}
                  {(!waterProviders?.providers || waterProviders.providers.length === 0) && (
                    <div className="rounded-lg bg-muted/50 px-3 py-2 text-sm text-muted-foreground">
                      No water providers configured
                    </div>
                  )}
                </div>
                <Link
                  to="/water"
                  className="mt-4 inline-flex items-center gap-2 text-sm font-medium text-primary transition-colors hover:text-primary/80"
                >
                  View All Water Rates
                  <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
                </Link>
              </CardContent>
              {/* Decorative gradient */}
              <div className="absolute inset-x-0 bottom-0 h-1 bg-gradient-to-r from-blue-500/0 via-blue-500/50 to-blue-500/0 opacity-0 transition-opacity group-hover:opacity-100" />
            </Card>
          </div>

          {/* System Info */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                System Information
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">Version</p>
                  <p className="font-mono text-sm">v0.5.x</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">API Endpoint</p>
                  <p className="font-mono text-sm">/rates, /water</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">Data Storage</p>
                  <p className="font-mono text-sm">PostgreSQL</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm text-muted-foreground">Refresh Schedule</p>
                  <p className="font-mono text-sm">Hourly</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}
