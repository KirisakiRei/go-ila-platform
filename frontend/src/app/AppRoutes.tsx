import type { ReactNode } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { AdminShell } from '@/features/admin/components/AdminShell'
import { AdminIdleGuard } from '@/features/admin/components/AdminIdleGuard'
import { useAdminAuth } from '@/features/admin/admin-auth-context'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import AdminLoginPage from '@/pages/admin/AdminLoginPage'
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
import LegalCenterPage from '@/pages/public/legal/LegalCenterPage'
import AdminAuditPage from '@/pages/admin/AdminAuditPage'
import AdminCmsPage from '@/pages/admin/AdminCmsPage'
import AdminPageContentPage from '@/pages/admin/AdminPageContentPage'
import AdminLegalPage from '@/pages/admin/AdminLegalPage'
import AdminBrokerHealthPage from '@/pages/admin/AdminBrokerHealthPage'
import AdminEnginePage from '@/pages/admin/AdminEnginePage'
import AdminHealthPage from '@/pages/admin/AdminHealthPage'
import AdminRunnerPage from '@/pages/admin/AdminRunnerPage'
import AdminMaintenancePage from '@/pages/admin/AdminMaintenancePage'
import AdminSettingsPage from '@/pages/admin/AdminSettingsPage'
import AdminSupportPage from '@/pages/admin/AdminSupportPage'
import OnboardingPage from '@/pages/OnboardingPage'
import SecuritySettingsPage from '@/pages/SecuritySettingsPage'
import AdminCouponsPage from '@/pages/admin/AdminCouponsPage'
import AdminCreditCodesPage from '@/pages/admin/AdminCreditCodesPage'
import AdminInvoicesPage from '@/pages/admin/AdminInvoicesPage'
import AdminTrialsPage from '@/pages/admin/AdminTrialsPage'
import AdminHomePage from '@/pages/admin/AdminHomePage'
import AdminPlansPage from '@/pages/admin/AdminPlansPage'
import AdminReportsPage from '@/pages/admin/AdminReportsPage'
import AdminUserDetailPage from '@/pages/admin/AdminUserDetailPage'
import AdminUsersPage from '@/pages/admin/AdminUsersPage'
import { FullPageLoading } from './FullPageLoading'
import { NotFoundPage } from './NotFoundPage'
import { ScrollToTop } from './ScrollToTop'

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
 * AdminPage membungkus halaman panel admin dan menuntut sesi admin.
 *
 * Panel admin memakai cookie sesi sendiri di server. Karena itu yang diperiksa
 * di sini bukan sesi pelanggan: masuk sebagai pelanggan tidak membuka panel,
 * dan admin yang belum masuk diarahkan ke pintu admin, bukan pintu pelanggan.
 */
function AdminPage({ children }: { children: ReactNode }) {
  const { admin, isLoading } = useAdminAuth()

  if (isLoading) return <FullPageLoading label="Memeriksa sesi admin…" />
  if (!admin) return <Navigate to="/admin/login" replace />

  return (
    <>
      <AdminIdleGuard />
      <AdminShell actorEmail={admin.email}>{children}</AdminShell>
    </>
  )
}

/** RedirectIfAdmin masuk: admin yang sudah bersesi tidak perlu melihat pintunya. */
function RedirectIfAdmin({ children }: { children: ReactNode }) {
  const { admin, isLoading } = useAdminAuth()

  if (isLoading) return <FullPageLoading label="Memeriksa sesi admin…" />
  if (admin) return <Navigate to="/admin" replace />
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
    <>
      {/* Panel pelanggan juga perlu kembali ke atas saat berpindah halaman. */}
      <ScrollToTop />
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
          path="/mulai"
          element={
            <RequireAuth>
              <ShellPage>
                <OnboardingPage />
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
          path="/pengaturan/keamanan"
          element={
            <RequireAuth>
              <ShellPage>
                <SecuritySettingsPage />
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

        {/* Pintu admin: tanpa sesi admin. Tidak ada pendaftaran di sini. */}
        <Route
          path="/admin/login"
          element={
            <RedirectIfAdmin>
              <AdminLoginPage />
            </RedirectIfAdmin>
          }
        />

        {/* Panel admin. Server tetap penentu akhir: tanpa permission, 403. */}
        <Route
          path="/admin"
          element={
            <AdminPage>
              <AdminHomePage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/pengguna"
          element={
            <AdminPage>
              <AdminUsersPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/pengguna/:id"
          element={
            <AdminPage>
              <AdminUserDetailPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/paket"
          element={
            <AdminPage>
              <AdminPlansPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/kupon"
          element={
            <AdminPage>
              <AdminCouponsPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/tagihan"
          element={
            <AdminPage>
              <AdminInvoicesPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/trial"
          element={
            <AdminPage>
              <AdminTrialsPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/kode-kredit"
          element={
            <AdminPage>
              <AdminCreditCodesPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/laporan"
          element={
            <AdminPage>
              <AdminReportsPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/audit"
          element={
            <AdminPage>
              <AdminAuditPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/pengaturan-sistem"
          element={
            <AdminPage>
              <AdminSettingsPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/kesehatan"
          element={
            <AdminPage>
              <AdminHealthPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/engine"
          element={
            <AdminPage>
              <AdminEnginePage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/runner"
          element={
            <AdminPage>
              <AdminRunnerPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/kesehatan-broker"
          element={
            <AdminPage>
              <AdminBrokerHealthPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/pemeliharaan"
          element={
            <AdminPage>
              <AdminMaintenancePage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/dukungan"
          element={
            <AdminPage>
              <AdminSupportPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/konten"
          element={
            <AdminPage>
              <AdminCmsPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/konten/legal"
          element={
            <AdminPage>
              <AdminLegalPage />
            </AdminPage>
          }
        />
        <Route
          path="/admin/konten/halaman"
          element={
            <AdminPage>
              <AdminPageContentPage />
            </AdminPage>
          }
        />

        {/* Dokumen legal boleh dibaca tanpa masuk. */}
        <Route path="/legal" element={<LegalCenterPage />} />
        <Route path="/legal/:slug" element={<LegalCenterPage />} />

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </>
  )
}
