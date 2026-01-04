import { useState, useEffect } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components'
import { Info, Clock, Save } from 'lucide-react'
import { getRefreshInterval, setRefreshInterval } from '@/lib/api'

export function GeneralSettingsPage() {
  const [interval, setInterval] = useState('3600')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    getRefreshInterval().then(res => setInterval(res.interval)).catch(console.error)
  }, [])

  const handleSaveInterval = async () => {
    setSaving(true)
    try {
      await setRefreshInterval(interval)
    } catch (err) {
      console.error(err)
      alert('Failed to save interval')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="mt-2 text-muted-foreground">
          Configure your eRateManager preferences
        </p>
      </div>

      {/* Data Refresh */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            Data Refresh
          </CardTitle>
          <CardDescription>
            Configure how often the application polls providers for new data
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium">Refresh Interval</label>
              <p className="text-sm text-muted-foreground mb-3">
                Select the frequency of data updates
              </p>
              <div className="flex flex-col gap-3">
                <div className="flex gap-3 items-center">
                  <select
                    value={['300', '900', '3600', '21600', '43200', '86400'].includes(interval) ? interval : 'custom'}
                    onChange={(e) => {
                      if (e.target.value !== 'custom') {
                        setInterval(e.target.value)
                      } else {
                        // If switching to custom, keep current value if it's not in presets, or default to 60s
                        if (['300', '900', '3600', '21600', '43200', '86400'].includes(interval)) {
                          setInterval('') 
                        }
                      }
                    }}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 max-w-[200px]"
                  >
                    <option value="300">Every 5 minutes</option>
                    <option value="900">Every 15 minutes</option>
                    <option value="3600">Every hour</option>
                    <option value="21600">Every 6 hours</option>
                    <option value="43200">Every 12 hours</option>
                    <option value="86400">Every 24 hours</option>
                    <option value="custom">Custom</option>
                  </select>
                  <button
                    onClick={handleSaveInterval}
                    disabled={saving || !interval}
                    className="inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"
                  >
                    {saving ? (
                      'Saving...'
                    ) : (
                      <>
                        <Save className="mr-2 h-4 w-4" />
                        Save
                      </>
                    )}
                  </button>
                </div>
                
                {!['300', '900', '3600', '21600', '43200', '86400'].includes(interval) && (
                  <div className="flex items-center gap-2 animate-fade-in">
                    <div className="grid w-full max-w-sm items-center gap-1.5">
                      <label htmlFor="custom-interval" className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                        Cron Expression or Seconds
                      </label>
                      <input
                        type="text"
                        id="custom-interval"
                        value={interval}
                        onChange={(e) => setInterval(e.target.value)}
                        placeholder="e.g. 0 0 * * * or 3600"
                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                      />
                    </div>
                    <p className="text-sm text-muted-foreground mt-6">
                      {interval && /^\d+$/.test(interval)
                        ? `~${Math.round(parseInt(interval) / 60)} minutes` 
                        : 'Standard cron syntax'}
                    </p>
                  </div>
                )}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* About */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Info className="h-5 w-5" />
            About eRateManager
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              eRateManager is a utility rate tracking and management system that helps you
              monitor electricity and water rates from various utility providers. It automatically
              fetches and parses rate information from provider websites and PDFs.
            </p>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="rounded-lg border border-border p-4">
                <p className="text-sm font-medium">Features</p>
                <ul className="mt-2 space-y-1 text-sm text-muted-foreground">
                  <li>• Automatic rate discovery from PDFs</li>
                  <li>• Water and sewer rate tracking</li>
                  <li>• Home Assistant integration</li>
                  <li>• Prometheus metrics</li>
                  <li>• Kubernetes deployment</li>
                </ul>
              </div>
              <div className="rounded-lg border border-border p-4">
                <p className="text-sm font-medium">Links</p>
                <ul className="mt-2 space-y-1 text-sm">
                  <li>
                    <a
                      href="https://github.com/bher20/eratemanager"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline"
                    >
                      GitHub Repository
                    </a>
                  </li>
                  <li>
                    <a
                      href="/metrics"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline"
                    >
                      Prometheus Metrics
                    </a>
                  </li>
                  <li>
                    <a
                      href="/healthz"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline"
                    >
                      Health Check
                    </a>
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
