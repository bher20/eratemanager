import { useEffect, useState } from 'react'
import { getUsers, getRoles, createUser, updateUser } from '@/lib/api'
import { User } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '@/components/Card'
import { LoadingSpinner } from '@/components/Loading'
import { Button } from '@/components/Button'
import { Select } from '@/components/Select'
import { Plus, X, ChevronDown } from 'lucide-react'
import { useAuth } from '@/context/AuthContext'
import { Navigate } from 'react-router-dom'

export function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isInviteModalOpen, setIsInviteModalOpen] = useState(false)
  const [isDropdownOpen, setIsDropdownOpen] = useState(false)
  const [newUser, setNewUser] = useState({ username: '', firstName: '', lastName: '', password: '', email: '', role: '' })
  const [inviteUser, setInviteUser] = useState({ firstName: '', lastName: '', email: '', role: '' })
  const [error, setError] = useState('')
  const [inviteSuccess, setInviteSuccess] = useState(false)
  const [resendingInvite, setResendingInvite] = useState<string | null>(null)
  const { checkPermission, token } = useAuth()

  if (!checkPermission('users', 'read')) {
    return <Navigate to="/" replace />
  }

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [usersData, rolesData] = await Promise.all([getUsers(), getRoles()])
      setUsers(usersData)
      setRoles(rolesData)
      if (rolesData.length > 0) {
        setNewUser(prev => ({ ...prev, role: rolesData[0] }))
        setInviteUser(prev => ({ ...prev, role: rolesData[0] }))
      }
    } catch (error) {
      console.error('Failed to load data', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await createUser(newUser.username, newUser.firstName, newUser.lastName, newUser.password, newUser.email, newUser.role)
      setIsCreateModalOpen(false)
      setNewUser({ username: '', firstName: '', lastName: '', password: '', email: '', role: roles[0] || '' })
      loadData() // Reload list
    } catch (err: any) {
      setError(err.message || 'Failed to create user')
    }
  }

  const handleInviteUser = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setInviteSuccess(false)
    try {
      const res = await fetch('/auth/users', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}` 
        },
        body: JSON.stringify({
          username: inviteUser.email.split('@')[0], // Simple username generation
          first_name: inviteUser.firstName,
          last_name: inviteUser.lastName,
          email: inviteUser.email,
          role: inviteUser.role,
          invite: true // Flag to use invitation flow
        })
      })
      if (!res.ok) throw new Error('Failed to invite user')
      setInviteSuccess(true)
      setInviteUser({ firstName: '', lastName: '', email: '', role: roles[0] || '' })
      setTimeout(() => {
        setIsInviteModalOpen(false)
        setInviteSuccess(false)
        loadData() // Reload list
      }, 2000)
    } catch (err: any) {
      setError(err.message || 'Failed to invite user')
    }
  }

  const handleRoleChange = async (userId: string, newRole: string) => {
    try {
      await updateUser(userId, { role: newRole })
      setUsers(users.map(u => u.id === userId ? { ...u, role: newRole } : u))
    } catch (error) {
      console.error('Failed to update role', error)
    }
  }

  const handleSkipVerificationChange = async (userId: string, skip: boolean) => {
    try {
      await updateUser(userId, { skip_email_verification: skip })
      setUsers(users.map(u => u.id === userId ? { ...u, skip_email_verification: skip } : u))
    } catch (error) {
      console.error('Failed to update skip verification', error)
    }
  }

  const handleResendInvitation = async (userId: string) => {
    setResendingInvite(userId)
    try {
      const res = await fetch(`/auth/users/${userId}/resend-invitation`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` }
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || 'Failed to resend invitation')
      }
      // Show success briefly
      setTimeout(() => setResendingInvite(null), 2000)
    } catch (error: any) {
      console.error('Failed to resend invitation', error)
      alert(error.message || 'Failed to resend invitation')
      setResendingInvite(null)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold tracking-tight">Users</h1>
        <div className="relative">
          <Button onClick={() => setIsDropdownOpen(!isDropdownOpen)}>
            <Plus className="mr-2 h-4 w-4" />
            New User
            <ChevronDown className="ml-2 h-4 w-4" />
          </Button>
          {isDropdownOpen && (
            <div className="absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-background border border-border z-10">
              <div className="py-1">
                <button
                  onClick={() => {
                    setIsInviteModalOpen(true)
                    setIsDropdownOpen(false)
                  }}
                  className="block w-full text-left px-4 py-2 text-sm hover:bg-muted"
                >
                  Invite User
                </button>
                <button
                  onClick={() => {
                    setIsCreateModalOpen(true)
                    setIsDropdownOpen(false)
                  }}
                  className="block w-full text-left px-4 py-2 text-sm hover:bg-muted"
                >
                  Create User
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Users List</CardTitle>
          <CardDescription>Manage users and their roles in the system</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="p-4 text-left font-medium">Username</th>
                  <th className="p-4 text-left font-medium">Email</th>
                  <th className="p-4 text-left font-medium">Role</th>
                  <th className="p-4 text-left font-medium">Skip Verification</th>
                  <th className="p-4 text-left font-medium">Created At</th>
                  <th className="p-4 text-left font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className="border-b last:border-0">
                    <td className="p-4 font-medium">{user.username}</td>
                    <td className="p-4">{user.email}</td>
                    <td className="p-4">
                      <select
                        value={user.role}
                        onChange={(e) => handleRoleChange(user.id, e.target.value)}
                        className="h-8 rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                      >
                        {roles.map(role => (
                          <option key={role} value={role}>{role}</option>
                        ))}
                      </select>
                    </td>
                    <td className="p-4">
                      <input
                        type="checkbox"
                        checked={user.skip_email_verification}
                        onChange={(e) => handleSkipVerificationChange(user.id, e.target.checked)}
                        className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                      />
                    </td>
                    <td className="p-4 text-muted-foreground">
                      {new Date(user.created_at).toLocaleDateString()}
                    </td>
                    <td className="p-4">
                      {!user.onboarding_completed && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleResendInvitation(user.id)}
                          disabled={resendingInvite === user.id}
                        >
                          {resendingInvite === user.id ? '✓ Sent' : 'Resend Invite'}
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
                {users.length === 0 && (
                  <tr>
                    <td colSpan={6} className="p-4 text-center text-muted-foreground">
                      No users found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {isCreateModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">Create New User</h2>
              <button onClick={() => setIsCreateModalOpen(false)} className="text-muted-foreground hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            </div>
            
            {error && (
              <div className="mb-4 rounded-md bg-destructive/15 p-3 text-sm text-destructive">
                {error}
              </div>
            )}

            <form onSubmit={handleCreateUser} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Username <span className="text-red-500">*</span></label>
                <input
                  type="text"
                  required
                  value={newUser.username}
                  onChange={e => setNewUser({ ...newUser, username: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">First Name</label>
                <input
                  type="text"
                  value={newUser.firstName}
                  onChange={e => setNewUser({ ...newUser, firstName: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Last Name</label>
                <input
                  type="text"
                  value={newUser.lastName}
                  onChange={e => setNewUser({ ...newUser, lastName: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Email <span className="text-red-500">*</span></label>
                <input
                  type="email"
                  required
                  value={newUser.email}
                  onChange={e => setNewUser({ ...newUser, email: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Password <span className="text-red-500">*</span></label>
                <input
                  type="password"
                  required
                  value={newUser.password}
                  onChange={e => setNewUser({ ...newUser, password: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Role</label>
                <Select
                  options={roles.map(r => ({ value: r, label: r }))}
                  value={newUser.role}
                  onChange={e => setNewUser({ ...newUser, role: e.target.value })}
                />
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <Button type="button" variant="outline" onClick={() => setIsCreateModalOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit">
                  Create User
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {isInviteModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">Invite User</h2>
              <button onClick={() => setIsInviteModalOpen(false)} className="text-muted-foreground hover:text-foreground">
                <X className="h-4 w-4" />
              </button>
            </div>
            
            {error && (
              <div className="mb-4 rounded-md bg-destructive/15 p-3 text-sm text-destructive">
                {error}
              </div>
            )}

            {inviteSuccess && (
              <div className="mb-4 rounded-md bg-green-50 text-green-700 p-3 text-sm">
                Invitation sent successfully!
              </div>
            )}

            <form onSubmit={handleInviteUser} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Email Address <span className="text-red-500">*</span></label>
                <input
                  type="email"
                  required
                  value={inviteUser.email}
                  onChange={e => setInviteUser({ ...inviteUser, email: e.target.value })}
                  placeholder="colleague@example.com"
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">First Name</label>
                <input
                  type="text"
                  value={inviteUser.firstName}
                  onChange={e => setInviteUser({ ...inviteUser, firstName: e.target.value })}
                  placeholder="John"
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Last Name</label>
                <input
                  type="text"
                  value={inviteUser.lastName}
                  onChange={e => setInviteUser({ ...inviteUser, lastName: e.target.value })}
                  placeholder="Doe"
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Role</label>
                <Select
                  options={roles.map(r => ({ value: r, label: r }))}
                  value={inviteUser.role}
                  onChange={e => setInviteUser({ ...inviteUser, role: e.target.value })}
                />
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <Button type="button" variant="outline" onClick={() => setIsInviteModalOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={inviteSuccess}>
                  {inviteSuccess ? 'Sent!' : 'Send Invitation'}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
