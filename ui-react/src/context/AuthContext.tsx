import React, { createContext, useContext, useState, useEffect } from 'react'
import { User } from '@/lib/types'
import { hasPermission } from '@/lib/permissions'

interface AuthContextType {
  user: User | null
  token: string | null
  login: (token: string, user: User) => void
  logout: () => void
  refreshUser: () => Promise<void>
  isAuthenticated: boolean
  isLoading: boolean
  checkPermission: (resource: string, action: string) => boolean
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const storedToken = localStorage.getItem('token')
    if (storedToken) {
      setToken(storedToken)
      // Always fetch fresh user data from server to get latest state
      fetch('/auth/me', {
        headers: { Authorization: `Bearer ${storedToken}` }
      })
        .then(res => {
          if (res.ok) return res.json()
          throw new Error('Failed to fetch user')
        })
        .then(updatedUser => {
          localStorage.setItem('user', JSON.stringify(updatedUser))
          setUser(updatedUser)
        })
        .catch(err => {
          console.error('Failed to refresh user on load:', err)
          // Fall back to localStorage if fetch fails
          const storedUser = localStorage.getItem('user')
          if (storedUser) {
            setUser(JSON.parse(storedUser))
          }
        })
        .finally(() => {
          setIsLoading(false)
        })
    } else {
      setIsLoading(false)
    }
  }, [])

  const login = (newToken: string, newUser: User) => {
    localStorage.setItem('token', newToken)
    localStorage.setItem('user', JSON.stringify(newUser))
    setToken(newToken)
    setUser(newUser)
  }

  const logout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setToken(null)
    setUser(null)
  }

  const refreshUser = async () => {
    if (!token) return
    try {
      const response = await fetch('/auth/me', {
        headers: { Authorization: `Bearer ${token}` }
      })
      if (response.ok) {
        const updatedUser = await response.json()
        localStorage.setItem('user', JSON.stringify(updatedUser))
        setUser(updatedUser)
      }
    } catch (error) {
      console.error('Failed to refresh user', error)
    }
  }

  const checkPermission = (resource: string, action: string) => {
    return hasPermission(user, resource, action)
  }

  return (
    <AuthContext.Provider value={{ user, token, login, logout, refreshUser, isAuthenticated: !!token, isLoading, checkPermission }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
