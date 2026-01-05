import { useState, useEffect } from 'react'
import { useAuth } from '@/context/AuthContext'
import { Button } from '@/components/Button'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/Card'
import { Input } from '@/components/Input'
import { Label } from '@/components/Label'
import { Select } from '@/components/Select'
import { CheckCircle, Mail, Users, RefreshCw, ArrowRight, SkipForward, X } from 'lucide-react'
import { EmailConfig } from '@/lib/types'

const STEPS = [
  { id: 'welcome', title: 'Welcome' },
  { id: 'refresh', title: 'Data Refresh' },
  { id: 'email', title: 'Email Setup' },
  { id: 'users', title: 'Invite Users' },
  { id: 'complete', title: 'All Set' },
]

export function OnboardingWizard() {
  const { token, refreshUser } = useAuth()
  const [currentStep, setCurrentStep] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isCompleted, setIsCompleted] = useState(false)

  // Data Refresh State
  const [refreshInterval, setRefreshInterval] = useState('3600')

  // Email State
  const [emailConfig, setEmailConfig] = useState<EmailConfig>({
    provider: 'smtp',
    from_address: 'noreply@example.com',
    from_name: 'eRateManager',
    enabled: true,
    host: '',
    port: 587,
    username: '',
    password: '',
    encryption: 'tls'
  })

  // Invite User State
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteFirstName, setInviteFirstName] = useState('')
  const [inviteLastName, setInviteLastName] = useState('')
  const [inviteRole, setInviteRole] = useState('viewer')
  const [inviteSuccess, setInviteSuccess] = useState(false)
  const [queuedUsers, setQueuedUsers] = useState<Array<{
    email: string
    firstName: string
    lastName: string
    role: string
  }>>([])

  useEffect(() => {
    if (currentStep === 1) {
      // Fetch refresh interval
      fetch('/settings/refresh-interval', {
        headers: { Authorization: `Bearer ${token}` }
      })
        .then(res => res.json())
        .then(data => {
          if (data.interval) setRefreshInterval(data.interval)
        })
        .catch(console.error)
    } else if (currentStep === 2) {
      // Fetch email config
      fetch('/api/v1/settings/email', {
        headers: { Authorization: `Bearer ${token}` }
      })
        .then(res => res.json())
        .then(data => {
          if (data && data.provider) setEmailConfig(data)
        })
        .catch(console.error)
    }
  }, [currentStep, token])

  const handleNext = async () => {
    setIsLoading(true)
    setError(null)

    try {
      if (currentStep === 1) {
        // Save refresh interval
        await fetch('/settings/refresh-interval', {
          method: 'POST',
          headers: { 
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}` 
          },
          body: JSON.stringify({ interval: refreshInterval })
        })
      } else if (currentStep === 2) {
        // Save email config
        const res = await fetch('/api/v1/settings/email', {
          method: 'PUT',
          headers: { 
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}` 
          },
          body: JSON.stringify(emailConfig)
        })
        if (!res.ok) throw new Error('Failed to save email config')
      } else if (currentStep === 3) {
        // Invite user (handled in separate button, this is just skip/next)
      }

      setCurrentStep(prev => prev + 1)
    } catch (err: any) {
      setError(err.message || 'An error occurred')
    } finally {
      setIsLoading(false)
    }
  }

  const handleInviteUser = async () => {
    setError(null)
    
    // Check if email is already in the queue
    if (queuedUsers.some(u => u.email === inviteEmail)) {
      setError('This user is already in the invite queue')
      return
    }

    // Add to queue
    setQueuedUsers(prev => [...prev, {
      email: inviteEmail,
      firstName: inviteFirstName,
      lastName: inviteLastName,
      role: inviteRole
    }])
    
    // Clear form and show success message
    setInviteSuccess(true)
    setInviteEmail('')
    setInviteFirstName('')
    setInviteLastName('')
    
    // Clear success message after a moment
    setTimeout(() => setInviteSuccess(false), 2000)
  }

  const handleRemoveQueuedUser = (email: string) => {
    setQueuedUsers(prev => prev.filter(u => u.email !== email))
  }

  const handleComplete = async () => {
    setIsLoading(true)
    setError(null)
    try {
      // First, send invitations to all queued users
      if (queuedUsers.length > 0) {
        console.log('Sending invitations to', queuedUsers.length, 'users')
        const inviteResults = await Promise.allSettled(
          queuedUsers.map(user =>
            fetch('/auth/users', {
              method: 'POST',
              headers: { 
                'Content-Type': 'application/json',
                Authorization: `Bearer ${token}` 
              },
              body: JSON.stringify({
                username: user.email.split('@')[0],
                first_name: user.firstName,
                last_name: user.lastName,
                email: user.email,
                role: user.role,
                invite: true
              })
            }).then(async res => {
              if (!res.ok) {
                const errorData = await res.text()
                // Skip users that already exist
                if (errorData.includes('already exists') || errorData.includes('user already exists')) {
                  console.log('User already exists, skipping:', user.email)
                  return { email: user.email, success: true, skipped: true }
                }
                throw new Error(`${user.email}: ${errorData}`)
              }
              console.log('Successfully invited:', user.email)
              return { email: user.email, success: true, skipped: false }
            })
          )
        )
        
        // Check for any real failures (not just "already exists")
        const failures = inviteResults.filter(r => r.status === 'rejected')
        console.log('Invite results:', inviteResults.length, 'total,', failures.length, 'failures')
        if (failures.length > 0) {
          const errorMessages = failures.map((f: any) => f.reason?.message || 'Unknown error').join('\n')
          throw new Error(`Failed to invite some users:\n${errorMessages}`)
        }
      }

      // Then mark onboarding as completed
      console.log('Marking onboarding as completed')
      const res = await fetch('/auth/me', {
        method: 'PUT',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}` 
        },
        body: JSON.stringify({ onboarding_completed: true })
      })
      if (!res.ok) {
        const errorText = await res.text()
        console.error('Failed to mark onboarding complete:', errorText)
        throw new Error(errorText || 'Failed to complete onboarding')
      }
      console.log('Onboarding marked complete, refreshing user')
      // Refresh user context to update onboarding_completed state
      // This will cause the OnboardingCheck component to hide the wizard
      await refreshUser()
      console.log('User refreshed, wizard should close')
      setIsCompleted(true)
    } catch (err: any) {
      console.error('Error completing onboarding:', err)
      setError(err.message || 'Failed to complete onboarding')
    } finally {
      setIsLoading(false)
    }
  }

  // Don't render if completed
  if (isCompleted) {
    return null
  }

  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return (
          <div className="text-center space-y-4 py-8">
            <div className="bg-blue-100 p-4 rounded-full w-20 h-20 mx-auto flex items-center justify-center">
              <span className="text-4xl">👋</span>
            </div>
            <h2 className="text-2xl font-bold">Welcome to eRateManager!</h2>
            <p className="text-gray-600 max-w-md mx-auto">
              Let's get your system set up in just a few steps. We'll configure data refreshing, email notifications, and invite your team.
            </p>
          </div>
        )
      case 1:
        return (
          <div className="space-y-4">
            <div className="flex items-center gap-2 mb-4">
              <RefreshCw className="w-6 h-6 text-blue-600" />
              <h2 className="text-xl font-semibold">Data Refresh Settings</h2>
            </div>
            <p className="text-sm text-gray-600">
              How often should eRateManager check for new rate PDFs?
            </p>
            <div className="space-y-2">
              <Label>Refresh Interval</Label>
              <Select 
                value={refreshInterval} 
                onChange={(e) => setRefreshInterval(e.target.value)}
                options={[
                  { label: 'Every 5 Minutes', value: '300' },
                  { label: 'Every 15 Minutes', value: '900' },
                  { label: 'Every Hour', value: '3600' },
                  { label: 'Every 6 Hours', value: '21600' },
                  { label: 'Every 12 Hours', value: '43200' },
                  { label: 'Daily', value: '86400' },
                  { label: 'Weekly (Sunday at midnight)', value: '0 0 * * 0' },
                ]}
              />
            </div>
          </div>
        )
      case 2:
        return (
          <div className="space-y-4">
            <div className="flex items-center gap-2 mb-4">
              <Mail className="w-6 h-6 text-blue-600" />
              <h2 className="text-xl font-semibold">Email Configuration</h2>
            </div>
            <p className="text-sm text-gray-600">
              Configure how the system sends emails for alerts and user invites.
            </p>
            
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Provider</Label>
                <Select 
                  value={emailConfig.provider} 
                  onChange={(e) => setEmailConfig({...emailConfig, provider: e.target.value as any})}
                  options={[
                    { label: 'SMTP', value: 'smtp' },
                    { label: 'Gmail', value: 'gmail' },
                    { label: 'SendGrid', value: 'sendgrid' },
                    { label: 'Resend', value: 'resend' },
                  ]}
                />
              </div>
              <div className="space-y-2">
                <Label>From Address</Label>
                <Input 
                  value={emailConfig.from_address} 
                  onChange={(e) => setEmailConfig({...emailConfig, from_address: e.target.value})}
                />
              </div>
            </div>

            {emailConfig.provider === 'smtp' && (
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Host</Label>
                  <Input 
                    value={emailConfig.host} 
                    onChange={(e) => setEmailConfig({...emailConfig, host: e.target.value})}
                    placeholder="smtp.example.com"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Port</Label>
                  <Input 
                    type="number"
                    value={emailConfig.port} 
                    onChange={(e) => setEmailConfig({...emailConfig, port: parseInt(e.target.value)})}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Username</Label>
                  <Input 
                    value={emailConfig.username} 
                    onChange={(e) => setEmailConfig({...emailConfig, username: e.target.value})}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Password</Label>
                  <Input 
                    type="password"
                    value={emailConfig.password} 
                    onChange={(e) => setEmailConfig({...emailConfig, password: e.target.value})}
                  />
                </div>
              </div>
            )}

            {(emailConfig.provider === 'sendgrid' || emailConfig.provider === 'resend') && (
              <div className="space-y-2">
                <Label>API Key</Label>
                <Input 
                  type="password"
                  value={emailConfig.api_key} 
                  onChange={(e) => setEmailConfig({...emailConfig, api_key: e.target.value})}
                />
              </div>
            )}
          </div>
        )
      case 3:
        return (
          <div className="space-y-4">
            <div className="flex items-center gap-2 mb-4">
              <Users className="w-6 h-6 text-blue-600" />
              <h2 className="text-xl font-semibold">Invite Team Members</h2>
            </div>
            <p className="text-sm text-gray-600">
              Add users to invite. They'll receive invitation emails when you complete the setup.
            </p>
            
            {inviteSuccess && (
              <div className="bg-green-50 text-green-700 p-3 rounded-md text-sm">
                User added to invite queue!
              </div>
            )}

            {/* Queued Users List */}
            {queuedUsers.length > 0 && (
              <div className="space-y-2">
                <Label className="text-sm font-medium">Users to Invite ({queuedUsers.length})</Label>
                <div className="border rounded-md divide-y max-h-48 overflow-y-auto">
                  {queuedUsers.map((user, index) => (
                    <div key={index} className="flex items-center justify-between p-3 hover:bg-gray-50">
                      <div className="flex-1">
                        <div className="font-medium text-sm">{user.email}</div>
                        <div className="text-xs text-gray-500">
                          {user.firstName} {user.lastName} • {user.role}
                        </div>
                      </div>
                      <button
                        onClick={() => handleRemoveQueuedUser(user.email)}
                        className="text-red-500 hover:text-red-700 p-1"
                        title="Remove from queue"
                      >
                        <X className="w-4 h-4" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="space-y-4 border p-4 rounded-md">
              <div className="space-y-2">
                <Label>Email Address</Label>
                <Input 
                  type="email"
                  value={inviteEmail} 
                  onChange={(e) => setInviteEmail(e.target.value)}
                  placeholder="colleague@example.com"
                />
              </div>
              <div className="space-y-2">
                <Label>First Name</Label>
                <Input 
                  type="text"
                  value={inviteFirstName} 
                  onChange={(e) => setInviteFirstName(e.target.value)}
                  placeholder="John"
                />
              </div>
              <div className="space-y-2">
                <Label>Last Name</Label>
                <Input 
                  type="text"
                  value={inviteLastName} 
                  onChange={(e) => setInviteLastName(e.target.value)}
                  placeholder="Doe"
                />
              </div>
              <div className="space-y-2">
                <Label>Role</Label>
                <Select 
                  value={inviteRole} 
                  onChange={(e) => setInviteRole(e.target.value)}
                  options={[
                    { label: 'Viewer', value: 'viewer' },
                    { label: 'Editor', value: 'editor' },
                    { label: 'Admin', value: 'admin' },
                  ]}
                />
              </div>
              <Button 
                onClick={handleInviteUser} 
                disabled={!inviteEmail}
                variant="outline"
                className="w-full"
              >
                Add to Queue
              </Button>
            </div>
          </div>
        )
      case 4:
        return (
          <div className="space-y-6">
            <div className="text-center space-y-4">
              <div className="bg-green-100 p-4 rounded-full w-20 h-20 mx-auto flex items-center justify-center">
                <CheckCircle className="w-10 h-10 text-green-600" />
              </div>
              <h2 className="text-2xl font-bold">You're All Set!</h2>
              <p className="text-gray-600">
                Here's a summary of your configuration:
              </p>
            </div>

            {/* Configuration Summary */}
            <div className="space-y-4 text-left">
              {/* Refresh Interval */}
              <div className="border rounded-lg p-4">
                <div className="flex items-center gap-2 mb-2">
                  <RefreshCw className="w-5 h-5 text-blue-600" />
                  <h3 className="font-semibold">Data Refresh Interval</h3>
                </div>
                <p className="text-sm text-gray-600">
                  {refreshInterval === '300' && 'Every 5 minutes'}
                  {refreshInterval === '900' && 'Every 15 minutes'}
                  {refreshInterval === '1800' && 'Every 30 minutes'}
                  {refreshInterval === '3600' && 'Every hour'}
                  {refreshInterval === '21600' && 'Every 6 hours'}
                  {refreshInterval === '43200' && 'Every 12 hours'}
                  {refreshInterval === '86400' && 'Daily'}
                  {refreshInterval === '0 0 * * 0' && 'Weekly (Sunday at midnight)'}
                </p>
              </div>

              {/* Email Configuration */}
              <div className="border rounded-lg p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Mail className="w-5 h-5 text-blue-600" />
                  <h3 className="font-semibold">Email Configuration</h3>
                </div>
                <p className="text-sm text-gray-600">
                  {emailConfig.enabled ? (
                    <>
                      <span className="text-green-600 font-medium">Enabled</span>
                      <br />
                      Provider: {emailConfig.provider === 'smtp' ? 'SMTP' : emailConfig.provider}
                      <br />
                      From: {emailConfig.from_name} &lt;{emailConfig.from_address}&gt;
                    </>
                  ) : (
                    <span className="text-gray-500">Disabled (can be configured later)</span>
                  )}
                </p>
              </div>

              {/* User Invitations */}
              <div className="border rounded-lg p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Users className="w-5 h-5 text-blue-600" />
                  <h3 className="font-semibold">User Invitations</h3>
                </div>
                {queuedUsers.length > 0 ? (
                  <div className="space-y-2">
                    <p className="text-sm text-muted-foreground mb-2">
                      {queuedUsers.length} user{queuedUsers.length !== 1 ? 's' : ''} will be invited:
                    </p>
                    <div className="space-y-1 max-h-32 overflow-y-auto">
                      {queuedUsers.map((user, index) => (
                        <div key={index} className="text-sm bg-muted p-2 rounded">
                          <span className="font-medium">{user.email}</span>
                          <span className="text-muted-foreground"> • {user.role}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No users queued for invitation</p>
                )}
              </div>
            </div>

            <p className="text-sm text-gray-500 text-center">
              Click "Complete Setup" to finalize your configuration and send invitations.
            </p>
          </div>
        )
      default:
        return null
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <Card className="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <CardHeader>
          <div className="flex justify-between items-center">
            <CardTitle>Setup Wizard</CardTitle>
            <div className="text-sm text-gray-500">
              Step {currentStep + 1} of {STEPS.length}
            </div>
          </div>
          {/* Progress Bar */}
          <div className="w-full bg-gray-200 h-2 rounded-full mt-2">
            <div 
              className="bg-blue-600 h-2 rounded-full transition-all duration-300"
              style={{ width: `${((currentStep + 1) / STEPS.length) * 100}%` }}
            />
          </div>
        </CardHeader>
        
        <CardContent>
          {error && (
            <div className="bg-red-50 text-red-600 p-3 rounded-md mb-4 text-sm">
              {error}
            </div>
          )}
          {renderStepContent()}
        </CardContent>

        <CardFooter className="flex justify-between border-t pt-4">
          {currentStep > 0 ? (
            <Button variant="ghost" onClick={() => setCurrentStep(prev => prev - 1)}>
              Back
            </Button>
          ) : (
            <div />
          )}

          {currentStep < STEPS.length - 1 ? (
            <div className="flex gap-2">
              {currentStep > 0 && (
                <Button variant="ghost" onClick={() => setCurrentStep(prev => prev + 1)}>
                  Skip <SkipForward className="w-4 h-4 ml-2" />
                </Button>
              )}
              <Button onClick={handleNext} disabled={isLoading}>
                {currentStep === 0 ? 'Get Started' : 'Next'} <ArrowRight className="w-4 h-4 ml-2" />
              </Button>
            </div>
          ) : (
            <Button onClick={handleComplete} disabled={isLoading} className="bg-green-600 hover:bg-green-700">
              {isLoading ? 'Completing...' : 'Complete Setup'}
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  )
}
