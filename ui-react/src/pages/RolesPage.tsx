import { useEffect, useState } from 'react'
import { getRoles } from '@/lib/api'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '@/components/Card'
import { LoadingSpinner } from '@/components/Loading'

export function RolesPage() {
  const [roles, setRoles] = useState<string[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadRoles()
  }, [])

  const loadRoles = async () => {
    try {
      const data = await getRoles()
      setRoles(data)
    } catch (error) {
      console.error('Failed to load roles', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Roles</CardTitle>
          <CardDescription>Available roles in the system</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="p-4 text-left font-medium">Role Name</th>
                </tr>
              </thead>
              <tbody>
                {roles.map((role) => (
                  <tr key={role} className="border-b last:border-0">
                    <td className="p-4 font-medium">{role}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
