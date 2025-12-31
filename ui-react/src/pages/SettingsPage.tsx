import { useState, useEffect } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components'
import { Moon, Sun, Palette, Info } from 'lucide-react'

export function SettingsPage() {
  const [theme, setTheme] = useState<'dark' | 'light'>(() => {
    return document.documentElement.classList.contains('light') ? 'light' : 'dark'
  })

  useEffect(() => {
    document.documentElement.classList.toggle('light', theme === 'light')
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }, [theme])

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="mt-2 text-muted-foreground">
          Configure your eRateManager preferences
        </p>
      </div>

      {/* Appearance */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Palette className="h-5 w-5" />
            Appearance
          </CardTitle>
          <CardDescription>
            Customize the look and feel of the dashboard
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium">Theme</label>
              <p className="text-sm text-muted-foreground mb-3">
                Select your preferred color scheme
              </p>
              <div className="flex gap-3">
                <button
                  onClick={() => setTheme('dark')}
                  className={`flex items-center gap-2 rounded-lg border px-4 py-3 transition-all ${
                    theme === 'dark'
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border hover:border-muted-foreground'
                  }`}
                >
                  <Moon className="h-5 w-5" />
                  <span className="font-medium">Dark</span>
                </button>
                <button
                  onClick={() => setTheme('light')}
                  className={`flex items-center gap-2 rounded-lg border px-4 py-3 transition-all ${
                    theme === 'light'
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border hover:border-muted-foreground'
                  }`}
                >
                  <Sun className="h-5 w-5" />
                  <span className="font-medium">Light</span>
                </button>
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
