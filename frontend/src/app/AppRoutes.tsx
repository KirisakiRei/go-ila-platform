import type { ReactNode } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { AdminShell } from '@/features/admin/components/AdminShell'
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
import BillingPage from '@/pages/BillingPage'
import CheckoutPage from '@/pages/CheckoutPage'
import AdminAuditPage from '@/pages/admin/AdminAuditPage'
import AdminCouponsPage from '@/pages/admin/AdminCouponsPage'
import AdminHomePage from '@/pages/admin/AdminHomePage'
import AdminPlansPage from '@/pages/admin/AdminPlansPage'
import AdminReportsPage from '@/pages/admin/AdminReportsPage'
import AdminUserDetailPage from '@/pages/admin/AdminUserDetailPage'
import AdminUsersPage from '@/pages/admin/AdminUsersPage'
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

/**
 * AdminPage membungkus halaman panel admin.
 *
 * Panel admin terpisah dari aplikasi pelanggan: pintunya beda cookie, dan
 * akun tanpa role admin ditolak backend meski lolos di sini.
 */
function AdminPage({ children }: { children: ReactNode }) {
  const { user } = useAuthContext()
  if (!user) return null
  return <AdminShell actorEmail={user.email}>{children}</AdminShell>
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

      <Route
        path="/langganan"
        element={
          <RequireAuth>
            <ShellPage>
              <BillingPage />
            </ShellPage>
          </RequireAuth>
        }
      />
      <Route
        path="/langganan/bayar"
        element={
          <RequireAuth>
            <ShellPage>
              <CheckoutPage />
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

      {/* Panel admin. Server tetap penentu akhir: tanpa permission, 403. */}
      <Route
        path="/admin"
        element={
          <RequireAuth>
            <AdminPage>
              <AdminHomePage />
            </AdminPage>
          </RequireAuth>
        }
      />
      <Route
        path="/admin/pengguna"
        element={
          <RequireAuth>
            <AdminPage>
              <AdminUsersPage />
            </AdminPage>
          </RequireAuth>
        }
      />
      <Route
        path="/admin/pengguna/:id"
        element={
          <RequireAuth>
            <AdminPage>
              <AdminUserDetailPage />
            </AdminPage>
          </RequireAuth>
        }
      />
      <Route
        path="/admin/paket"
        element={
          <RequireAuth>
            <AdminPage>
              <AdminPlansPage />
            </AdminPage>
          </RequireAuth>
        }
      />
      <Route
        path="/admin/kupon"
        element={
          <RequireAuth>
            <AdminPage>
              <AdminCouponsPage />
            </AdminPage>
          </RequireAuth>
        }
      />
      <Route
        path="/admin/laporan"
        element={
          <RequireAuth>
            <AdminPage>
              <AdminReportsPage />
            </AdminPage>
          </RequireAuth>
        }
      />
      <Route
        path="/admin/audit"
        element={
          <RequireAuth>
            <AdminPage>
              <AdminAuditPage />
            </AdminPage>
          </RequireAuth>
        }
      />

      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
