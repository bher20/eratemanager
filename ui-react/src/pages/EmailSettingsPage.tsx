import { useState, useEffect } from 'react'
import { getEmailConfig, saveEmailConfig, testEmailConfig } from '@/lib/api'
import type { EmailConfig } from '@/lib/types'
import { Button } from '@/components/Button'
import { Card } from '@/components/Card'

export function EmailSettingsPage() {
  const [config, setConfig] = useState<EmailConfig>({
    provider: 'smtp',
    from_address: '',
    from_name: 'eRateManager',
    enabled: false,
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testEmail, setTestEmail] = useState('')
  const [message, setMessage] = useState<{ type: 'success' | 'error', text: string } | null>(null)

  useEffect(() => {
    loadConfig()
  }, [])

  async function loadConfig() {
    try {
      const data = await getEmailConfig()
      if (data) {
        setConfig(data)
      }
    } catch (error) {
      console.error(error)
      setMessage({ type: 'error', text: 'Failed to load email settings' })
    } finally {
      setLoading(false)
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setMessage(null)
    try {
      await saveEmailConfig(config)
      setMessage({ type: 'success', text: 'Email settings saved' })
    } catch (error) {
      console.error(error)
      setMessage({ type: 'error', text: 'Failed to save settings' })
    } finally {
      setSaving(false)
    }
  }

  async function handleTest() {
    if (!testEmail) {
      setMessage({ type: 'error', text: 'Please enter a test email address' })
      return
    }
    setTesting(true)
    setMessage(null)
    try {
      await testEmailConfig(config, testEmail)
      setMessage({ type: 'success', text: 'Test email sent' })
    } catch (error) {
      console.error(error)
      setMessage({ type: 'error', text: 'Failed to send test email: ' + (error as Error).message })
    } finally {
      setTesting(false)
    }
  }

  if (loading) return <div className="text-foreground">Loading...</div>

  return (
    <div className="space-y-6">
      {message && (
        <div className={`p-4 rounded-md ${message.type === 'success' ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400'}`}>
          {message.text}
        </div>
      )}

      <Card className="p-6">
        <h3 className="text-lg font-medium leading-6 text-foreground mb-4">Email Configuration</h3>
        <form onSubmit={handleSave} className="space-y-4">
          <div className="flex items-center space-x-2">
            <input
              type="checkbox"
              id="enabled"
              checked={config.enabled}
              onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
              className="h-4 w-4 rounded border-input bg-background text-primary focus:ring-primary"
            />
            <label htmlFor="enabled" className="text-sm font-medium text-foreground">Enable Email Notifications</label>
          </div>

          <div className="grid gap-2">
            <label htmlFor="provider" className="text-sm font-medium text-foreground">Provider</label>
            <select
              id="provider"
              value={config.provider}
              onChange={(e) => setConfig({ ...config, provider: e.target.value as any })}
              className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="smtp">SMTP</option>
              <option value="gmail">Gmail</option>
              <option value="sendgrid">Sendgrid</option>
            </select>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="grid gap-2">
              <label htmlFor="from_name" className="text-sm font-medium text-foreground">From Name</label>
              <input
                id="from_name"
                value={config.from_name}
                onChange={(e) => setConfig({ ...config, from_name: e.target.value })}
                required
                className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
            <div className="grid gap-2">
              <label htmlFor="from_address" className="text-sm font-medium text-foreground">From Address</label>
              <input
                id="from_address"
                type="email"
                value={config.from_address}
                onChange={(e) => setConfig({ ...config, from_address: e.target.value })}
                required
                className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
          </div>

          {config.provider === 'sendgrid' ? (
            <div className="grid gap-2">
              <label htmlFor="api_key" className="text-sm font-medium text-foreground">API Key</label>
              <input
                id="api_key"
                type="password"
                value={config.api_key || ''}
                onChange={(e) => setConfig({ ...config, api_key: e.target.value })}
                required
                className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
          ) : (
            <>
              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <label htmlFor="host" className="text-sm font-medium text-foreground">Host</label>
                  <input
                    id="host"
                    value={config.host || ''}
                    onChange={(e) => setConfig({ ...config, host: e.target.value })}
                    required
                    className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </div>
                <div className="grid gap-2">
                  <label htmlFor="port" className="text-sm font-medium text-foreground">Port</label>
                  <input
                    id="port"
                    type="number"
                    value={config.port || ''}
                    onChange={(e) => setConfig({ ...config, port: parseInt(e.target.value) || 0 })}
                    required
                    className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </div>
              </div>
              <div className="grid gap-2">
                <label htmlFor="encryption" className="text-sm font-medium text-foreground">Encryption</label>
                <select
                  id="encryption"
                  value={config.encryption || 'none'}
                  onChange={(e) => setConfig({ ...config, encryption: e.target.value as any })}
                  className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="none">None</option>
                  <option value="ssl">SSL/TLS (Implicit)</option>
                  <option value="tls">STARTTLS (Explicit)</option>
                </select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <label htmlFor="username" className="text-sm font-medium text-foreground">Username</label>
                  <input
                    id="username"
                    value={config.username || ''}
                    onChange={(e) => setConfig({ ...config, username: e.target.value })}
                    className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </div>
                <div className="grid gap-2">
                  <label htmlFor="password" className="text-sm font-medium text-foreground">Password</label>
                  <input
                    id="password"
                    type="password"
                    value={config.password || ''}
                    onChange={(e) => setConfig({ ...config, password: e.target.value })}
                    className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </div>
              </div>
            </>
          )}

          <div className="flex justify-end space-x-2">
            <Button type="submit" disabled={saving}>
              {saving ? 'Saving...' : 'Save Settings'}
            </Button>
          </div>
        </form>
      </Card>

      <Card className="p-6">
        <h3 className="text-lg font-medium leading-6 text-foreground mb-4">Test Configuration</h3>
        <div className="flex space-x-2">
          <input
            placeholder="Enter email address"
            value={testEmail}
            onChange={(e) => setTestEmail(e.target.value)}
            className="flex h-10 w-full rounded-md border border-input bg-background text-foreground px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          />
          <Button onClick={handleTest} disabled={testing} variant="secondary">
            {testing ? 'Sending...' : 'Send Test Email'}
          </Button>
        </div>
      </Card>
    </div>
  )
}
