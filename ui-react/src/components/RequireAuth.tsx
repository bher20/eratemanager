import { useLocation, Navigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'

export function RequireAuth({ children }: { children: JSX.Element }) {
  const { isAuthenticated, isLoading } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return <div>Loading...</div>
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return children
}
