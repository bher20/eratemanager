import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import { 
  DashboardPage, 
  ElectricPage, 
  WaterPage, 
  GeneralSettingsPage, 
  UsersPage,
  RolesPage,
  PrivilegesPage,
  LoginPage, 
  TokensPage, 
  OnboardingPage 
} from '@/pages'
import { AuthProvider } from '@/context/AuthContext'
import { RequireAuth } from '@/components/RequireAuth'
import { useEffect, useState } from 'react'
import { getAuthStatus } from '@/lib/api'
import { LoadingSpinner } from '@/components/Loading'

function App() {
  const [initialized, setInitialized] = useState<boolean | null>(null)

  useEffect(() => {
    checkInitialization()
  }, [])

  const checkInitialization = async () => {
    try {
      const { initialized } = await getAuthStatus()
      setInitialized(initialized)
    } catch (error) {
      console.error('Failed to check initialization status', error)
      // If check fails, assume initialized to prevent unauthorized onboarding access
      setInitialized(true)
    }
  }

  if (initialized === null) {
    return (
      <div className="flex h-screen items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  return (
    <AuthProvider>
      <Routes>
        <Route 
          path="/onboarding" 
          element={initialized ? <Navigate to="/login" /> : <OnboardingPage onComplete={() => setInitialized(true)} />} 
        />
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="*"
          element={
            !initialized ? (
              <Navigate to="/onboarding" />
            ) : (
              <RequireAuth>
                <Layout>
                  <Routes>
                    <Route path="/" element={<DashboardPage />} />
                    <Route path="/electric" element={<ElectricPage />} />
                    <Route path="/water" element={<WaterPage />} />
                    <Route path="/tokens" element={<TokensPage />} />
                    
                    {/* Settings Routes */}
                    <Route path="/settings" element={<Navigate to="/settings/general" replace />} />
                    <Route path="/settings/general" element={<GeneralSettingsPage />} />
                    <Route path="/settings/users" element={<UsersPage />} />
                    <Route path="/settings/rbac/roles" element={<RolesPage />} />
                    <Route path="/settings/rbac/privileges" element={<PrivilegesPage />} />
                  </Routes>
                </Layout>
              </RequireAuth>
            )
          }
        />
      </Routes>
    </AuthProvider>
  )
}

export default App
