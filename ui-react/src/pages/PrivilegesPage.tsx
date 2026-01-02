import { useEffect, useState } from 'react'
import { getPrivileges } from '@/lib/api'
import { Privilege } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '@/components/Card'
import { LoadingSpinner } from '@/components/Loading'

export function PrivilegesPage() {
  const [privileges, setPrivileges] = useState<Privilege[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadPrivileges()
  }, [])

  const loadPrivileges = async () => {
    try {
      const data = await getPrivileges()
      setPrivileges(data)
    } catch (error) {
      console.error('Failed to load privileges', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) return <LoadingSpinner />

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Privileges</CardTitle>
          <CardDescription>System access control policies</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="p-4 text-left font-medium">Role</th>
                  <th className="p-4 text-left font-medium">Resource</th>
                  <th className="p-4 text-left font-medium">Action</th>
                </tr>
              </thead>
              <tbody>
                {privileges.map((priv, i) => (
                  <tr key={i} className="border-b last:border-0">
                    <td className="p-4 font-medium">{priv.role}</td>
                    <td className="p-4 font-mono text-xs">{priv.resource}</td>
                    <td className="p-4 font-mono text-xs">{priv.action}</td>
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
