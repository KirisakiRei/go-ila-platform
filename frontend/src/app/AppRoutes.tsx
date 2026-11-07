import type { ReactNode } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import ApprovePage from '@/pages/ApprovePage'
import DashboardPage from '@/pages/DashboardPage'
import LoginPage from '@/pages/LoginPage'
import PositionsPage from '@/pages/PositionsPage'
import RegisterPage from '@/pages/RegisterPage'
import SettingsPage from '@/pages/SettingsPage'
import SignalsPage from '@/pages/SignalsPage'
import StrategiesPage from '@/pages/StrategiesPage'
import AccountsPage from '@/pages/AccountsPage'
import { FullPageLoading } from './FullPageLoading'
import { NotFoundPage } from './NotFoundPage'

function RequireAuth({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuthContext()
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function RedirectIfAuthenticated({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuthContext()
  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

/** Halaman dalam kerangka aplikasi; user dijamin tersedia oleh RequireAuth. */
function ShellPage({ children }: { children: ReactNode }) {
  const { user } = useAuthContext()
  if (!user) return null
  return <AppShell user={user}>{children}</AppShell>
}

export function AppRoutes() {
  const { isLoading } = useAuthContext()

  if (isLoading) {
    return <FullPageLoading label="Memuat sesi…" />
  }

  return (
    <Routes>
      <Route
        path="/login"
        element={
          <RedirectIfAuthenticated>
            <LoginPage />
          </RedirectIfAuthenticated>
        }
      />
      <Route
        path="/register"
        element={
          <RedirectIfAuthenticated>
            <RegisterPage />
          </RedirectIfAuthenticated>
        }
      />

      <Route
        path="/"
        element={
          <RequireAuth>
            <ShellPage>
              <DashboardPage />
            </ShellPage>
          </RequireAuth>
        }
      />
      <Route
        path="/sinyal"
        element={
          <RequireAuth>
            <ShellPage>
              <SignalsPage />
            </ShellPage>
          </RequireAuth>
        }
      />
      <Route
        path="/posisi"
        element={
          <RequireAuth>
            <ShellPage>
              <PositionsPage />
            </ShellPage>
          </RequireAuth>
        }
      />
      <Route
        path="/akun"
        element={
          <RequireAuth>
            <ShellPage>
              <AccountsPage />
            </ShellPage>
          </RequireAuth>
        }
      />
      <Route
        path="/strategi"
        element={
          <RequireAuth>
            <ShellPage>
              <StrategiesPage />
            </ShellPage>
          </RequireAuth>
        }
      />
      <Route
        path="/pengaturan"
        element={
          <RequireAuth>
            <ShellPage>
              <SettingsPage />
            </ShellPage>
          </RequireAuth>
        }
      />

      {/* Deep-link dari bot Telegram wajib login; tanpa sesi diarahkan ke login. */}
      <Route
        path="/approve/:token"
        element={
          <RequireAuth>
            <ShellPage>
              <ApprovePage />
            </ShellPage>
          </RequireAuth>
        }
      />

      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
