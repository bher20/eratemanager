import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import { updateProfile, sendVerificationEmail } from '@/lib/api'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/Card'
import { Button } from '@/components/Button'
import { Input } from '@/components/Input'
import { Label } from '@/components/Label'
import { Alert, AlertDescription } from '@/components/Alert'
import { LoadingSpinner } from '@/components/Loading'
import { Mail, CheckCircle, AlertTriangle } from 'lucide-react'

export function EmailVerificationRequiredPage() {
  const { user, refreshUser } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [verificationSent, setVerificationSent] = useState(false)

  useEffect(() => {
    if (user) {
      setEmail(user.email || '')
      // If user is already verified or skipped, redirect to dashboard
      if (user.email_verified || user.skip_email_verification || user.role === 'admin') {
        navigate('/')
      }
    }
  }, [user, navigate])

  const handleUpdateEmail = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setMessage(null)
    try {
      await updateProfile(email)
      await refreshUser()
      setMessage({ type: 'success', text: 'Email updated successfully. Please verify your new email.' })
      setVerificationSent(false) // Reset so they can send verification to new email
    } catch (err: any) {
      setMessage({ type: 'error', text: err.message || 'Failed to update email' })
    } finally {
      setLoading(false)
    }
  }

  const handleSendVerification = async () => {
    setLoading(true)
    setMessage(null)
    try {
      await sendVerificationEmail()
      setVerificationSent(true)
      setMessage({ type: 'success', text: 'Verification email sent! Please check your inbox.' })
    } catch (err: any) {
      setMessage({ type: 'error', text: err.message || 'Failed to send verification email' })
    } finally {
      setLoading(false)
    }
  }

  const handleCheckVerification = async () => {
    setLoading(true)
    try {
      await refreshUser()
    } catch (error) {
      console.error('Failed to refresh user status', error)
    } finally {
      setLoading(false)
    }
  }

  if (!user) return <LoadingSpinner />

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="flex items-center justify-center mb-4">
            <div className="p-3 bg-yellow-100 rounded-full">
              <AlertTriangle className="h-8 w-8 text-yellow-600" />
            </div>
          </div>
          <CardTitle className="text-center text-2xl">Email Verification Required</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <p className="text-center text-muted-foreground">
            To continue using eRateManager, you must verify your email address.
          </p>

          {message && (
            <Alert variant={message.type === 'error' ? 'destructive' : 'default'} className={message.type === 'success' ? 'bg-green-50 text-green-900 border-green-200' : ''}>
              {message.type === 'success' && <CheckCircle className="h-4 w-4 mr-2" />}
              <AlertDescription>{message.text}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-4">
            <form onSubmit={handleUpdateEmail} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">Email Address</Label>
                <div className="flex gap-2">
                  <Input
                    id="email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="Enter your email"
                    required
                  />
                  <Button type="submit" variant="outline" disabled={loading || email === user.email}>
                    Update
                  </Button>
                </div>
              </div>
            </form>

            <div className="pt-4 border-t">
              <Button 
                className="w-full" 
                onClick={handleSendVerification} 
                disabled={loading || !user.email || verificationSent}
              >
                {loading ? <LoadingSpinner size="sm" className="mr-2" /> : <Mail className="mr-2 h-4 w-4" />}
                {verificationSent ? 'Email Sent' : 'Send Verification Email'}
              </Button>
              <p className="text-xs text-center text-muted-foreground mt-2">
                Click the link in the email to verify your account.
              </p>
            </div>
          </div>
        </CardContent>
        <CardFooter className="justify-center border-t pt-4">
          <Button variant="ghost" size="sm" onClick={handleCheckVerification} disabled={loading}>
            {loading ? <LoadingSpinner size="sm" className="mr-2" /> : null}
            I've verified my email, continue
          </Button>
        </CardFooter>
      </Card>
    </div>
  )
}
