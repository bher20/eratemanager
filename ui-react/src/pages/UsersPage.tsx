import { useEffect, useState } from 'react'
import { getUsers, getRoles, createUser, updateUser } from '@/lib/api'
import { User } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '@/components/Card'
import { LoadingSpinner } from '@/components/Loading'
import { Button } from '@/components/Button'
import { Select } from '@/components/Select'
import { Plus, X } from 'lucide-react'
import { useAuth } from '@/context/AuthContext'
import { Navigate } from 'react-router-dom'

export function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [newUser, setNewUser] = useState({ username: '', password: '', role: '' })
  const [error, setError] = useState('')
  const { checkPermission } = useAuth()

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
      await createUser(newUser.username, newUser.password, newUser.role)
      setIsCreateModalOpen(false)
      setNewUser({ username: '', password: '', role: roles[0] || '' })
      loadData() // Reload list
    } catch (err: any) {
      setError(err.message || 'Failed to create user')
    }
  }

  const handleRoleChange = async (userId: string, newRole: string) => {
    try {
      await updateUser(userId, newRole)
      setUsers(users.map(u => u.id === userId ? { ...u, role: newRole } : u))
    } catch (error) {
      console.error('Failed to update role', error)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold tracking-tight">Users</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add User
        </Button>
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
                  <th className="p-4 text-left font-medium">Role</th>
                  <th className="p-4 text-left font-medium">Created At</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className="border-b last:border-0">
                    <td className="p-4 font-medium">{user.username}</td>
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
                    <td className="p-4 text-muted-foreground">
                      {new Date(user.created_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
                {users.length === 0 && (
                  <tr>
                    <td colSpan={3} className="p-4 text-center text-muted-foreground">
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
                <label className="text-sm font-medium">Username</label>
                <input
                  type="text"
                  required
                  value={newUser.username}
                  onChange={e => setNewUser({ ...newUser, username: e.target.value })}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Password</label>
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
    </div>
  )
}
