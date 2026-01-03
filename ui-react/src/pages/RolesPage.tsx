import { useEffect, useState } from 'react'
import { getRoles, createRole, getPrivileges, addPolicy, removePolicy } from '@/lib/api'
import { Privilege } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '@/components/Card'
import { LoadingSpinner } from '@/components/Loading'
import { Button } from '@/components/Button'
import { Plus, X, Trash2, Shield, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

export function RolesPage() {
  const [roles, setRoles] = useState<string[]>([])
  const [privileges, setPrivileges] = useState<Privilege[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedRole, setSelectedRole] = useState<string | null>(null)
  
  // Modal states
  const [isAddRoleOpen, setIsAddRoleOpen] = useState(false)
  const [isAddPolicyOpen, setIsAddPolicyOpen] = useState(false)
  
  // Form states
  const [newRole, setNewRole] = useState('')
  const [newPolicy, setNewPolicy] = useState({ resource: '', action: '' })
  const [error, setError] = useState('')

  // New Role Policies state
  const [newRolePolicies, setNewRolePolicies] = useState<{ resource: string; action: string }[]>([])
  const [newRolePolicy, setNewRolePolicy] = useState({ resource: '', action: '' })

  const KNOWN_RESOURCES = [
    '*',
    'rates',
    'providers',
    'users',
    'roles',
    'tokens',
    'system'
  ]

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [rolesData, privData] = await Promise.all([getRoles(), getPrivileges()])
      setRoles(rolesData)
      setPrivileges(privData)
    } catch (error) {
      console.error('Failed to load data', error)
    } finally {
      setLoading(false)
    }
  }

  const handleAddRole = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    // Include any pending policy in the inputs
    const policiesToSubmit = [...newRolePolicies]
    if (newRolePolicy.resource && newRolePolicy.action) {
      policiesToSubmit.push(newRolePolicy)
    }

    try {
      await createRole(newRole, policiesToSubmit)
      setIsAddRoleOpen(false)
      setNewRole('')
      setNewRolePolicies([])
      setNewRolePolicy({ resource: '', action: '' })
      loadData()
    } catch (err: any) {
      setError(err.message || 'Failed to create role')
    }
  }

  const addPolicyToNewRole = () => {
    if (!newRolePolicy.resource || !newRolePolicy.action) return
    setNewRolePolicies([...newRolePolicies, newRolePolicy])
    setNewRolePolicy({ resource: '', action: '' })
  }

  const removePolicyFromNewRole = (index: number) => {
    setNewRolePolicies(newRolePolicies.filter((_, i) => i !== index))
  }

  const handleAddPolicy = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedRole) return
    setError('')
    try {
      await addPolicy(selectedRole, newPolicy.resource, newPolicy.action)
      setIsAddPolicyOpen(false)
      setNewPolicy({ resource: '', action: '' })
      loadData()
    } catch (err: any) {
      setError(err.message || 'Failed to add policy')
    }
  }

  const handleRemovePolicy = async (priv: Privilege) => {
    if (!confirm(`Remove permission for ${priv.role} to ${priv.action} ${priv.resource}?`)) return
    try {
      await removePolicy(priv.role, priv.resource, priv.action)
      loadData()
    } catch (error) {
      console.error('Failed to remove policy', error)
    }
  }

  if (loading) return <LoadingSpinner />

  const selectedRolePolicies = privileges.filter(p => p.role === selectedRole)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Roles & Policies</h1>
          <p className="text-muted-foreground">Manage roles and their access permissions</p>
        </div>
        <Button onClick={() => setIsAddRoleOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Role
        </Button>
      </div>

      <div className="grid gap-6 md:grid-cols-12">
        {/* Roles List */}
        <div className="md:col-span-4">
          <Card>
            <CardHeader>
              <CardTitle>Roles</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <div className="divide-y">
                {roles.map((role) => (
                  <button
                    key={role}
                    onClick={() => setSelectedRole(role)}
                    className={cn(
                      "flex w-full items-center justify-between p-4 text-sm font-medium transition-colors hover:bg-muted/50",
                      selectedRole === role ? "bg-muted" : ""
                    )}
                  >
                    <div className="flex items-center gap-3">
                      <Shield className="h-4 w-4 text-muted-foreground" />
                      {role}
                    </div>
                    <ChevronRight className={cn(
                      "h-4 w-4 text-muted-foreground transition-transform",
                      selectedRole === role ? "rotate-90" : ""
                    )} />
                  </button>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Policies List */}
        <div className="md:col-span-8">
          {selectedRole ? (
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <div className="space-y-1">
                  <CardTitle>{selectedRole} Policies</CardTitle>
                  <CardDescription>
                    Access permissions for the {selectedRole} role
                  </CardDescription>
                </div>
                <Button size="sm" onClick={() => setIsAddPolicyOpen(true)}>
                  <Plus className="mr-2 h-4 w-4" />
                  Add Policy
                </Button>
              </CardHeader>
              <CardContent>
                <div className="rounded-md border mt-4">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="p-4 text-left font-medium">Resource</th>
                        <th className="p-4 text-left font-medium">Action</th>
                        <th className="p-4 text-right font-medium"></th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedRolePolicies.map((priv, i) => (
                        <tr key={i} className="border-b last:border-0">
                          <td className="p-4 font-mono">{priv.resource}</td>
                          <td className="p-4 font-mono">{priv.action}</td>
                          <td className="p-4 text-right">
                            <Button 
                              variant="ghost" 
                              size="sm" 
                              onClick={() => handleRemovePolicy(priv)}
                              className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </td>
                        </tr>
                      ))}
                      {selectedRolePolicies.length === 0 && (
                        <tr>
                          <td colSpan={3} className="p-8 text-center text-muted-foreground">
                            No policies defined for this role
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </CardContent>
            </Card>
          ) : (
            <div className="flex h-full min-h-[300px] items-center justify-center rounded-lg border border-dashed p-8 text-center animate-in fade-in-50">
              <div className="max-w-[420px]">
                <Shield className="mx-auto h-12 w-12 text-muted-foreground/50" />
                <h3 className="mt-4 text-lg font-semibold">Select a Role</h3>
                <p className="mt-2 text-sm text-muted-foreground">
                  Select a role from the list to view and manage its permissions.
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Add Role Modal */}
      {isAddRoleOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-lg border bg-card p-6 shadow-lg">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold">Add Role</h2>
              <button onClick={() => setIsAddRoleOpen(false)} className="text-muted-foreground hover:text-foreground">
                <X className="h-5 w-5" />
              </button>
            </div>
            {error && <div className="mb-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}
            <form onSubmit={handleAddRole} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Role Name</label>
                <input
                  type="text"
                  value={newRole}
                  onChange={(e) => setNewRole(e.target.value)}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  placeholder="e.g. manager"
                  required
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Initial Policies</label>
                <div className="rounded-md border p-3 space-y-3">
                  <div className="flex gap-2">
                    <div className="flex-1">
                      <input
                        type="text"
                        list="new-role-resource-list"
                        value={newRolePolicy.resource}
                        onChange={(e) => setNewRolePolicy({ ...newRolePolicy, resource: e.target.value })}
                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        placeholder="Select or type resource..."
                      />
                      <datalist id="new-role-resource-list">
                        {KNOWN_RESOURCES.map((res) => (
                          <option key={res} value={res} />
                        ))}
                      </datalist>
                    </div>
                    <div className="w-32">
                      <select
                        value={newRolePolicy.action}
                        onChange={(e) => setNewRolePolicy({ ...newRolePolicy, action: e.target.value })}
                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      >
                        <option value="">Action</option>
                        <option value="read">read</option>
                        <option value="write">write</option>
                        <option value="*">*</option>
                      </select>
                    </div>
                    <Button 
                      type="button" 
                      size="sm"
                      variant="secondary"
                      onClick={addPolicyToNewRole}
                      disabled={!newRolePolicy.resource || !newRolePolicy.action}
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </div>

                  {newRolePolicies.length > 0 && (
                    <div className="space-y-2">
                      {newRolePolicies.map((p, i) => (
                        <div key={i} className="flex items-center justify-between rounded-md bg-muted px-3 py-2 text-sm">
                          <div className="flex gap-2">
                            <span className="font-mono font-medium">{p.resource}</span>
                            <span className="text-muted-foreground">/</span>
                            <span className="font-mono">{p.action}</span>
                          </div>
                          <button
                            type="button"
                            onClick={() => removePolicyFromNewRole(i)}
                            className="text-muted-foreground hover:text-destructive"
                          >
                            <X className="h-4 w-4" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>

              <div className="flex justify-end gap-3 pt-2">
                <Button type="button" variant="outline" onClick={() => setIsAddRoleOpen(false)}>Cancel</Button>
                <Button type="submit">Create Role</Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Policy Modal */}
      {isAddPolicyOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-lg border bg-card p-6 shadow-lg">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold">Add Policy for {selectedRole}</h2>
              <button onClick={() => setIsAddPolicyOpen(false)} className="text-muted-foreground hover:text-foreground">
                <X className="h-5 w-5" />
              </button>
            </div>
            {error && <div className="mb-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}
            <form onSubmit={handleAddPolicy} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Resource</label>
                <input
                  type="text"
                  list="resource-list"
                  value={newPolicy.resource}
                  onChange={(e) => setNewPolicy({ ...newPolicy, resource: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  placeholder="Select or type resource..."
                  required
                />
                <datalist id="resource-list">
                  {KNOWN_RESOURCES.map((res) => (
                    <option key={res} value={res} />
                  ))}
                </datalist>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Action</label>
                <select
                  value={newPolicy.action}
                  onChange={(e) => setNewPolicy({ ...newPolicy, action: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  required
                >
                  <option value="">Select action...</option>
                  <option value="read">read</option>
                  <option value="write">write</option>
                  <option value="*">* (all)</option>
                </select>
              </div>
              <div className="flex justify-end gap-3">
                <Button type="button" variant="outline" onClick={() => setIsAddPolicyOpen(false)}>Cancel</Button>
                <Button type="submit">Add Policy</Button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
