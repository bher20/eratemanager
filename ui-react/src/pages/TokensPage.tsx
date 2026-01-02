import React, { useEffect, useState } from 'react'
import { getTokens, createToken, deleteToken } from '@/lib/api'
import { Token } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '@/components/Card'
import { Button } from '@/components/Button'
import { Select } from '@/components/Select'
import { Trash2, Plus, Copy, Check } from 'lucide-react'

export function TokensPage() {
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)
  const [newTokenName, setNewTokenName] = useState('')
  const [role, setRole] = useState('editor')
  const [createdTokenValue, setCreatedTokenValue] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

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
      const { token, token_value } = await createToken(newTokenName, role)
      setTokens([...tokens, token])
      setCreatedTokenValue(token_value)
      setNewTokenName('')
      setRole('editor')
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

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold tracking-tight">API Tokens</h1>
      
      <Card>
        <CardHeader>
          <CardTitle>Create New Token</CardTitle>
          <CardDescription>Generate a new API token for external access.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="flex gap-4 items-end">
            <div className="space-y-2 flex-1">
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
            <div className="w-48">
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
              {tokens.map((token) => (
                <div key={token.id} className="flex items-center justify-between p-4 border rounded-lg">
                  <div>
                    <p className="font-medium">{token.name}</p>
                    <p className="text-sm text-muted-foreground">
                      Created: {new Date(token.created_at).toLocaleDateString()}
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
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
