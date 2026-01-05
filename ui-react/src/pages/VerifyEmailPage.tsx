import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { verifyEmail } from '@/lib/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/Card'
import { LoadingSpinner } from '@/components/Loading'
import { CheckCircle, XCircle } from 'lucide-react'

export function VerifyEmailPage() {
  const [searchParams] = useSearchParams()
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading')
  const [message, setMessage] = useState('')
  const navigate = useNavigate()

  useEffect(() => {
    const token = searchParams.get('token')
    if (!token) {
      setStatus('error')
      setMessage('Invalid verification link. Token is missing.')
      return
    }

    verifyEmail(token)
      .then(() => {
        setStatus('success')
        setMessage('Your email has been successfully verified.')
        setTimeout(() => {
          navigate('/login')
        }, 3000)
      })
      .catch((err) => {
        setStatus('error')
        setMessage(err.message || 'Failed to verify email. The link may be expired or invalid.')
      })
  }, [searchParams, navigate])

  return (
    <div className="flex items-center justify-center min-h-[80vh]">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl text-center">Email Verification</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center text-center space-y-4">
          {status === 'loading' && (
            <>
              <LoadingSpinner size="lg" />
              <p>Verifying your email...</p>
            </>
          )}
          {status === 'success' && (
            <>
              <CheckCircle className="h-16 w-16 text-green-500" />
              <p className="text-lg font-medium text-green-600">{message}</p>
              <p className="text-sm text-muted-foreground">Redirecting to login...</p>
            </>
          )}
          {status === 'error' && (
            <>
              <XCircle className="h-16 w-16 text-destructive" />
              <p className="text-lg font-medium text-destructive">{message}</p>
              <Link to="/login" className="text-primary hover:underline">
                Back to Login
              </Link>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
