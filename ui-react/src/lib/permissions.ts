import { User } from './types'

// Default policies matching the backend defaults
// In a real production app, these might be fetched from the backend
export const DEFAULT_POLICIES: Record<string, string[]> = {
  admin: ['*'],
  editor: ['rates:read', 'rates:write', 'providers:read', 'providers:write', 'tokens:read', 'tokens:write'],
  viewer: ['rates:read', 'providers:read', 'tokens:read', 'tokens:write'],
}

export function hasPermission(user: User | null, resource: string, action: string): boolean {
  if (!user) return false
  
  // Admin bypass
  if (user.role === 'admin') return true

  const permissions = DEFAULT_POLICIES[user.role] || []
  
  // Check for wildcard
  if (permissions.includes('*')) return true
  
  // Check specific permission
  const permission = `${resource}:${action}`
  return permissions.includes(permission)
}
