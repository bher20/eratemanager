import React, { useState, useEffect } from 'react'
import { useAuth } from '@/context/AuthContext'
import { Button } from '@/components/Button'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/Card'
import { Input } from '@/components/Input'
import { Label } from '@/components/Label'
import { Select } from '@/components/Select'
import { CheckCircle, Mail, Users, RefreshCw, ArrowRight, SkipForward } from 'lucide-react'
import { EmailConfig } from '@/lib/types'

const STEPS = [
  { id: 'welcome', title: 'Welcome' },
  { id: 'refresh', title: 'Data Refresh' },
  { id: 'email', title: 'Email Setup' },
  { id: 'users', title: 'Invite Users' },
  { id: 'complete', title: 'All Set' },
]

export function OnboardingWizard() {
  const { user, token, refreshUser } = useAuth()
  const [currentStep, setCurrentStep] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

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
  const [inviteRole, setInviteRole] = useState('viewer')
  const [inviteSuccess, setInviteSuccess] = useState(false)

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
    setIsLoading(true)
    setError(null)
    try {
      const res = await fetch('/auth/users', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}` 
        },
        body: JSON.stringify({
          username: inviteEmail.split('@')[0], // Simple username generation
          email: inviteEmail,
          password: Math.random().toString(36).slice(-8), // Random password, they should reset it
          role: inviteRole
        })
      })
      if (!res.ok) throw new Error('Failed to invite user')
      setInviteSuccess(true)
      setInviteEmail('')
    } catch (err: any) {
      setError(err.message)
    } finally {
      setIsLoading(false)
    }
  }

  const handleComplete = async () => {
    setIsLoading(true)
    try {
      const res = await fetch('/auth/me', {
        method: 'PUT',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}` 
        },
        body: JSON.stringify({ onboarding_completed: true })
      })
      if (!res.ok) throw new Error('Failed to complete onboarding')
      await refreshUser()
    } catch (err: any) {
      setError(err.message)
    } finally {
      setIsLoading(false)
    }
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
                  { label: 'Every Hour', value: '3600' },
                  { label: 'Every 6 Hours', value: '21600' },
                  { label: 'Every 12 Hours', value: '43200' },
                  { label: 'Daily', value: '86400' },
                  { label: 'Weekly', value: '604800' },
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
              Add other users to the system. You can skip this and do it later.
            </p>
            
            {inviteSuccess && (
              <div className="bg-green-50 text-green-700 p-3 rounded-md text-sm">
                Invitation sent successfully! You can add another.
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
                disabled={!inviteEmail || isLoading}
                variant="outline"
                className="w-full"
              >
                {isLoading ? 'Sending...' : 'Send Invitation'}
              </Button>
            </div>
          </div>
        )
      case 4:
        return (
          <div className="text-center space-y-4 py-8">
            <div className="bg-green-100 p-4 rounded-full w-20 h-20 mx-auto flex items-center justify-center">
              <CheckCircle className="w-10 h-10 text-green-600" />
            </div>
            <h2 className="text-2xl font-bold">You're All Set!</h2>
            <p className="text-gray-600 max-w-md mx-auto">
              Configuration is complete. You can always change these settings later from the Settings menu.
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
          {currentStep > 0 && currentStep < STEPS.length - 1 ? (
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
              Go to Dashboard
            </Button>
          )}
        </CardFooter>
      </Card>
    </div>
  )
}
