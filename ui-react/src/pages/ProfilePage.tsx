import { useState, useEffect } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components'
import { Moon, Sun, Palette, Monitor } from 'lucide-react'
import { useAuth } from '@/context/AuthContext'

export function ProfilePage() {
  const { user } = useAuth()
  const [theme, setTheme] = useState<'dark' | 'light' | 'auto'>(() => {
    if (localStorage.getItem('theme') === 'auto') return 'auto'
    return document.documentElement.classList.contains('light') ? 'light' : 'dark'
  })

  useEffect(() => {
    const root = document.documentElement
    
    if (theme === 'auto') {
      localStorage.setItem('theme', 'auto')
      const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
      root.classList.toggle('light', systemTheme === 'light')
      root.classList.toggle('dark', systemTheme === 'dark')

      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      const handleChange = (e: MediaQueryListEvent) => {
        const newSystemTheme = e.matches ? 'dark' : 'light'
        root.classList.toggle('light', newSystemTheme === 'light')
        root.classList.toggle('dark', newSystemTheme === 'dark')
      }

      mediaQuery.addEventListener('change', handleChange)
      return () => mediaQuery.removeEventListener('change', handleChange)
    } else {
      localStorage.removeItem('theme')
      root.classList.toggle('light', theme === 'light')
      root.classList.toggle('dark', theme === 'dark')
    }
  }, [theme])

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Profile</h1>
        <p className="mt-2 text-muted-foreground">
          Manage your account settings and preferences
        </p>
      </div>

      {/* User Info */}
      <Card>
        <CardHeader>
          <CardTitle>User Information</CardTitle>
          <CardDescription>Your account details</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="text-sm font-medium text-muted-foreground">Username</label>
              <p className="text-lg font-medium">{user?.username}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-muted-foreground">Role</label>
              <p className="text-lg font-medium capitalize">{user?.role}</p>
            </div>
          </div>
        </CardContent>
      </Card>

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
                <button
                  onClick={() => setTheme('auto')}
                  className={`flex items-center gap-2 rounded-lg border px-4 py-3 transition-all ${
                    theme === 'auto'
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border hover:border-muted-foreground'
                  }`}
                >
                  <Monitor className="h-5 w-5" />
                  <span className="font-medium">Auto</span>
                </button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
