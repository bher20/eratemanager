import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import { 
  DashboardPage, 
  ElectricPage, 
  WaterPage, 
  LoginPage, 
  TokensPage, 
  OnboardingPage,
  ProfilePage,
  ForgotPasswordPage,
  ResetPasswordPage,
  VerifyEmailPage,
  EmailVerificationRequiredPage,
  SetupAccountPage
} from '@/pages'
import { SettingsPage } from '@/pages/SettingsPage'
import { AuthProvider, useAuth } from '@/context/AuthContext'
import { RequireAuth } from '@/components/RequireAuth'
import { useEffect, useState } from 'react'
import { getAuthStatus } from '@/lib/api'
import { LoadingSpinner } from '@/components/Loading'

import { OnboardingWizard } from '@/components/OnboardingWizard'

function RequireEmailVerification({ children }: { children: JSX.Element }) {
  const { user, isLoading } = useAuth()

  if (isLoading) {
    return <LoadingSpinner />
  }

  if (user && user.role !== 'admin' && !user.email_verified && !user.skip_email_verification) {
    return <Navigate to="/email-verification-required" replace />
  }

  return children
}

function OnboardingCheck({ children }: { children: JSX.Element }) {
  const { user, isLoading } = useAuth()
  
  // Show wizard if user is admin and hasn't completed onboarding
  // Wait for loading to finish to ensure we have the latest user state from server
  // We render the children (app content) behind the modal
  return (
    <>
      {children}
      {!isLoading && user && user.role === 'admin' && !user.onboarding_completed && (
        <OnboardingWizard />
      )}
    </>
  )
}

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
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
        <Route path="/setup-account" element={<SetupAccountPage />} />
        <Route path="/verify-email" element={<VerifyEmailPage />} />
        <Route path="/email-verification-required" element={
          <RequireAuth>
            <EmailVerificationRequiredPage />
          </RequireAuth>
        } />
        <Route
          path="*"
          element={
            !initialized ? (
              <Navigate to="/onboarding" />
            ) : (
              <RequireAuth>
                <RequireEmailVerification>
                  <OnboardingCheck>
                    <Layout>
                      <Routes>
                        <Route path="/" element={<DashboardPage />} />
                        <Route path="/electric" element={<ElectricPage />} />
                        <Route path="/water" element={<WaterPage />} />
                        <Route path="/profile" element={<ProfilePage />} />
                        <Route path="/tokens" element={<TokensPage />} />
                        
                        {/* Settings Routes */}
                        <Route path="/settings" element={<SettingsPage />} />
                      </Routes>
                    </Layout>
                  </OnboardingCheck>
                </RequireEmailVerification>
              </RequireAuth>
            )
          }
        />
      </Routes>
    </AuthProvider>
  )
}

export default App
