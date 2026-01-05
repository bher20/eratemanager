import { useState, useEffect } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components'
import { Moon, Sun, Palette, Monitor, CheckCircle, AlertCircle } from 'lucide-react'
import { useAuth } from '@/context/AuthContext'
import { updateProfile } from '@/lib/api'
import { Button } from '@/components/Button'

export function ProfilePage() {
  const { user, login } = useAuth()
  const [email, setEmail] = useState(user?.email || '')
  const [isEditingEmail, setIsEditingEmail] = useState(false)
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [theme, setTheme] = useState<'dark' | 'light' | 'auto'>(() => {
    if (localStorage.getItem('theme') === 'auto') return 'auto'
    return document.documentElement.classList.contains('light') ? 'light' : 'dark'
  })

  useEffect(() => {
    if (user?.email) {
      setEmail(user.email)
    }
  }, [user])

  const handleUpdateEmail = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setMessage('')

    try {
      const updatedUser = await updateProfile(email)
      // Update local user state
      if (user) {
        login(localStorage.getItem('token') || '', updatedUser)
      }
      setMessage('Email updated. Please check your inbox for a verification link.')
      setIsEditingEmail(false)
    } catch (err: any) {
      setError(err.message || 'Failed to update email')
    } finally {
      setLoading(false)
    }
  }

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
            <div className="sm:col-span-2">
              <label className="text-sm font-medium text-muted-foreground">Email</label>
              {isEditingEmail ? (
                <form onSubmit={handleUpdateEmail} className="mt-2 flex gap-2 items-start max-w-md">
                  <div className="flex-1">
                    <input
                      type="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                      required
                    />
                    {error && <p className="text-xs text-destructive mt-1">{error}</p>}
                  </div>
                  <Button type="submit" disabled={loading}>Save</Button>
                  <Button type="button" variant="outline" onClick={() => {
                    setIsEditingEmail(false)
                    setEmail(user?.email || '')
                    setError('')
                  }}>Cancel</Button>
                </form>
              ) : (
                <div className="flex items-center gap-2 mt-1">
                  <p className="text-lg font-medium">{user?.email || 'No email set'}</p>
                  {user?.email_verified ? (
                    <span className="flex items-center text-xs text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400 px-2 py-0.5 rounded-full">
                      <CheckCircle className="w-3 h-3 mr-1" /> Verified
                    </span>
                  ) : user?.email ? (
                    <span className="flex items-center text-xs text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400 px-2 py-0.5 rounded-full">
                      <AlertCircle className="w-3 h-3 mr-1" /> Unverified
                    </span>
                  ) : null}
                  <Button variant="ghost" size="sm" onClick={() => setIsEditingEmail(true)} className="h-8 text-xs">
                    Edit
                  </Button>
                </div>
              )}
              {message && (
                <div className="mt-2 p-2 text-sm text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400 rounded-md">
                  {message}
                </div>
              )}
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
