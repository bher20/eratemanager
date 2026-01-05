import React, { useState, useEffect } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/Card'
import { Button } from '@/components/Button'
import { LoadingSpinner } from '@/components/Loading'
import { UserPlus } from 'lucide-react'

export function SetupAccountPage() {
  const [token, setToken] = useState('')
  const [username, setUsername] = useState('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [validating, setValidating] = useState(true)
  const [userInfo, setUserInfo] = useState<{ email: string; role: string; username: string; first_name?: string; last_name?: string } | null>(null)
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()

  useEffect(() => {
    const urlToken = searchParams.get('token')
    if (!urlToken) {
      setError('Invalid setup link. Token is missing.')
      setValidating(false)
      return
    }
    setToken(urlToken)

    // Validate token and get user info
    fetch(`/auth/validate-setup-token?token=${urlToken}`)
      .then(res => {
        if (!res.ok) throw new Error('Invalid or expired setup link')
        return res.json()
      })
      .then(data => {
        setUserInfo({ 
          email: data.email, 
          role: data.role, 
          username: data.username,
          first_name: data.first_name || '',
          last_name: data.last_name || ''
        })
        setUsername(data.username)
        setFirstName(data.first_name || '')
        setLastName(data.last_name || '')
        setValidating(false)
      })
      .catch(err => {
        setError(err.message || 'Invalid or expired setup link')
        setValidating(false)
      })
  }, [searchParams])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setMessage('')

    if (!token) {
      setError('Token is required.')
      return
    }

    if (!username || username.trim().length < 3) {
      setError('Username must be at least 3 characters long.')
      return
    }

    if (newPassword.length < 8) {
      setError('Password must be at least 8 characters long.')
      return
    }

    if (newPassword !== confirmPassword) {
      setError('Passwords do not match.')
      return
    }

    setLoading(true)

    try {
      const res = await fetch('/auth/setup-account', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          token, 
          username: username.trim(), 
          first_name: firstName.trim(),
          last_name: lastName.trim(),
          new_password: newPassword 
        })
      })

      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || 'Failed to set up account')
      }

      setMessage('Account successfully set up! Redirecting to login...')
      setTimeout(() => {
        navigate('/login', { state: { message: 'Account set up successfully. Please login with your new password.' } })
      }, 2000)
    } catch (err: any) {
      setError(err.message || 'Failed to set up account.')
    } finally {
      setLoading(false)
    }
  }

  if (validating) {
    return (
      <div className="flex items-center justify-center min-h-[80vh]">
        <Card className="w-full max-w-md">
          <CardContent className="flex flex-col items-center text-center space-y-4 py-8">
            <LoadingSpinner size="lg" />
            <p>Validating your invitation...</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error && !userInfo) {
    return (
      <div className="flex items-center justify-center min-h-[80vh]">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-2xl text-center">Setup Failed</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col items-center text-center space-y-4">
            <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md w-full">
              {error}
            </div>
            <Link to="/login" className="text-primary hover:underline">
              Return to Login
            </Link>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex items-center justify-center min-h-[80vh]">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="flex items-center justify-center mb-2">
            <div className="bg-blue-100 p-3 rounded-full">
              <UserPlus className="w-8 h-8 text-blue-600" />
            </div>
          </div>
          <CardTitle className="text-2xl text-center">Welcome to eRateManager!</CardTitle>
          {userInfo && (
            <p className="text-sm text-muted-foreground text-center mt-2">
              Set up your account as a <strong>{userInfo.role}</strong>
            </p>
          )}
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {message && (
              <div className="p-3 text-sm text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400 rounded-md">
                {message}
              </div>
            )}
            {error && (
              <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
                {error}
              </div>
            )}
            
            {userInfo && (
              <div className="p-3 bg-muted rounded-md">
                <p className="text-sm">
                  <strong>Email:</strong> {userInfo.email}
                </p>
              </div>
            )}

            <div className="space-y-2">
              <label htmlFor="username" className="text-sm font-medium">
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                minLength={3}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                placeholder="Choose your username"
              />
              <p className="text-xs text-muted-foreground">
                This will be your login username
              </p>
            </div>

            <div className="space-y-2">
              <label htmlFor="firstName" className="text-sm font-medium">
                First Name
              </label>
              <input
                id="firstName"
                type="text"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                placeholder="Your first name"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="lastName" className="text-sm font-medium">
                Last Name
              </label>
              <input
                id="lastName"
                type="text"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                placeholder="Your last name"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="newPassword" className="text-sm font-medium">
                Create Password
              </label>
              <input
                id="newPassword"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
                minLength={8}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                placeholder="Enter your password"
              />
              <p className="text-xs text-muted-foreground">
                Must be at least 8 characters long
              </p>
            </div>

            <div className="space-y-2">
              <label htmlFor="confirmPassword" className="text-sm font-medium">
                Confirm Password
              </label>
              <input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                minLength={8}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                placeholder="Confirm your password"
              />
            </div>

            <Button type="submit" disabled={loading} className="w-full">
              {loading ? 'Setting up...' : 'Complete Setup'}
            </Button>
          </form>

          <div className="mt-6 text-center text-sm">
            <Link to="/login" className="text-primary hover:underline">
              Already have an account? Login
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
