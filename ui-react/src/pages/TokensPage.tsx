import React, { useEffect, useState } from 'react'
import { getTokens, createToken, deleteToken } from '@/lib/api'
import { Token } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '@/components/Card'
import { Button } from '@/components/Button'
import { Select } from '@/components/Select'
import { Trash2, Plus, Copy, Check, AlertCircle } from 'lucide-react'
import { useAuth } from '@/context/AuthContext'
import { Navigate } from 'react-router-dom'

export function TokensPage() {
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)
  const [newTokenName, setNewTokenName] = useState('')
  const [role, setRole] = useState('editor')
  const [expiresIn, setExpiresIn] = useState('never')
  const [customExpirationDate, setCustomExpirationDate] = useState('')
  const [createdTokenValue, setCreatedTokenValue] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const { checkPermission } = useAuth()

  if (!checkPermission('tokens', 'read')) {
    return <Navigate to="/" replace />
  }

  useEffect(() => {
    loadTokens()
  }, [])

  const loadTokens = async () => {
    try {
      const data = await getTokens()
      setTokens(data)
    } catch (error) {
      console.error('Failed to load tokens', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTokenName) return

    try {
      // Use custom date if provided, otherwise use the dropdown selection
      const expirationValue = expiresIn === 'custom' ? customExpirationDate : expiresIn
      const { token, token_value } = await createToken(newTokenName, role, expirationValue)
      setTokens([...tokens, token])
      setCreatedTokenValue(token_value ?? null)
      setNewTokenName('')
      setRole('editor')
      setExpiresIn('never')
      setCustomExpirationDate('')
    } catch (error) {
      console.error('Failed to create token', error)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this token?')) return
    try {
      await deleteToken(id)
      setTokens(tokens.filter(t => t.id !== id))
    } catch (error) {
      console.error('Failed to delete token', error)
    }
  }

  const copyToClipboard = () => {
    if (createdTokenValue) {
      navigator.clipboard.writeText(createdTokenValue)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const getExpirationStatus = (token: Token): { text: string; isExpired: boolean } => {
    if (!token.expires_at) {
      return { text: 'Never expires', isExpired: false }
    }
    const expiresDate = new Date(token.expires_at)
    const isExpired = expiresDate < new Date()
    if (isExpired) {
      return { text: `Expired on ${expiresDate.toLocaleDateString()}`, isExpired: true }
    }
    return { text: `Expires ${expiresDate.toLocaleDateString()}`, isExpired: false }
  }

  const calculateExpirationDate = (duration: string): string => {
    const now = new Date()
    let expiryDate = new Date(now)

    if (duration === '24h') {
      expiryDate.setHours(expiryDate.getHours() + 24)
    } else if (duration === '7d') {
      expiryDate.setDate(expiryDate.getDate() + 7)
    } else if (duration === '30d') {
      expiryDate.setDate(expiryDate.getDate() + 30)
    } else if (duration === '90d') {
      expiryDate.setDate(expiryDate.getDate() + 90)
    }

    return expiryDate.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    })
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold tracking-tight">API Tokens</h1>
      
      <Card>
        <CardHeader>
          <CardTitle>Create New Token</CardTitle>
          <CardDescription>Generate a new API token for external access.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <label htmlFor="tokenName" className="text-sm font-medium leading-none">Token Name</label>
                <input
                  id="tokenName"
                  type="text"
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  placeholder="e.g. Home Assistant"
                  value={newTokenName}
                  onChange={(e) => setNewTokenName(e.target.value)}
                  required
                />
              </div>
              <div className="w-full">
                <Select
                  label="Role"
                  value={role}
                  onChange={(e) => setRole(e.target.value)}
                  options={[
                    { value: 'viewer', label: 'Viewer' },
                    { value: 'editor', label: 'Editor' },
                    { value: 'admin', label: 'Admin' },
                  ]}
                />
              </div>
              <div className="w-full">
                <Select
                  label="Expiration"
                  value={expiresIn}
                  onChange={(e) => setExpiresIn(e.target.value)}
                  options={[
                    { value: 'never', label: 'Never expires' },
                    { value: '24h', label: `24 hours (${calculateExpirationDate('24h')})` },
                    { value: '7d', label: `7 days (${calculateExpirationDate('7d')})` },
                    { value: '30d', label: `30 days (${calculateExpirationDate('30d')})` },
                    { value: '90d', label: `90 days (${calculateExpirationDate('90d')})` },
                    { value: 'custom', label: 'Custom date' },
                  ]}
                />
              </div>
              {expiresIn === 'custom' && (
                <div className="space-y-2">
                  <label htmlFor="customDate" className="text-sm font-medium leading-none">Expiration Date (mm/dd/yyyy)</label>
                  <input
                    id="customDate"
                    type="text"
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                    placeholder="e.g. 12/25/2026 or 12/25/2026 14:30"
                    value={customExpirationDate}
                    onChange={(e) => setCustomExpirationDate(e.target.value)}
                    required={expiresIn === 'custom'}
                  />
                  <p className="text-xs text-muted-foreground">Format: mm/dd/yyyy or mm/dd/yyyy HH:MM</p>
                </div>
              )}
            </div>
            <Button type="submit">
              <Plus className="mr-2 h-4 w-4" /> Generate
            </Button>
          </form>

          {createdTokenValue && (
            <div className="mt-4 p-4 bg-muted rounded-md border border-border">
              <p className="text-sm font-medium mb-2 text-yellow-600 dark:text-yellow-400">
                Make sure to copy your token now. You won't be able to see it again!
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 p-2 bg-background rounded border font-mono text-sm break-all">
                  {createdTokenValue}
                </code>
                <Button variant="outline" size="icon" onClick={copyToClipboard}>
                  {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Active Tokens</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p>Loading tokens...</p>
          ) : tokens.length === 0 ? (
            <p className="text-muted-foreground">No tokens found.</p>
          ) : (
            <div className="space-y-4">
              {tokens.map((token) => {
                const expStatus = getExpirationStatus(token)
                return (
                  <div
                    key={token.id}
                    className={`flex items-center justify-between p-4 border rounded-lg ${
                      expStatus.isExpired ? 'bg-red-50 border-red-200 dark:bg-red-950 dark:border-red-800' : ''
                    }`}
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <p className="font-medium">{token.name}</p>
                        {expStatus.isExpired && <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400" />}
                      </div>
                      <p className="text-sm text-muted-foreground">
                        Created: {new Date(token.created_at).toLocaleDateString()}
                      </p>
                      <p className={`text-sm ${expStatus.isExpired ? 'text-red-600 dark:text-red-400' : 'text-muted-foreground'}`}>
                        {expStatus.text}
                      </p>
                      {token.last_used_at && (
                        <p className="text-xs text-muted-foreground">
                          Last used: {new Date(token.last_used_at).toLocaleDateString()}
                        </p>
                      )}
                    </div>
                    <Button variant="destructive" size="sm" onClick={() => handleDelete(token.id)}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
