import { lazy, Suspense, type ReactNode } from 'react'
import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { AdminShell } from '@/features/admin/components/AdminShell'
import { AdminIdleGuard } from '@/features/admin/components/AdminIdleGuard'
import { RequireAdminPermission } from '@/features/admin/components/RequireAdminPermission'
import { useAdminAuth } from '@/features/admin/admin-auth-context'
import { resolveSafeNextPath, useAuthContext } from '@/features/auth/hooks/use-auth'
import { useInviteOnly } from '@/lib/use-invite-only'
import ApprovePage from '@/pages/ApprovePage'
import DashboardPage from '@/pages/DashboardPage'
import ForgotPasswordPage from '@/pages/ForgotPasswordPage'
import LoginPage from '@/pages/LoginPage'
import PositionsPage from '@/pages/PositionsPage'
import RegisterPage from '@/pages/RegisterPage'
import ResetPasswordPage from '@/pages/ResetPasswordPage'
import SettingsPage from '@/pages/SettingsPage'
import SignalsPage from '@/pages/SignalsPage'
import StrategiesPage from '@/pages/StrategiesPage'
import AccountsPage from '@/pages/AccountsPage'
import AccountDetailPage from '@/pages/AccountDetailPage'
import BillingPage from '@/pages/BillingPage'
import CheckoutPage from '@/pages/CheckoutPage'
import OnboardingPage from '@/pages/OnboardingPage'
import SecuritySettingsPage from '@/pages/SecuritySettingsPage'
import LaporkanBugPage from '@/pages/LaporkanBugPage'
import RiwayatLaporanBugPage from '@/pages/RiwayatLaporanBugPage'
import DetailLaporanBugPage from '@/pages/DetailLaporanBugPage'
import ProfilPage from '@/pages/ProfilPage'
import PerangkatPage from '@/pages/PerangkatPage'
import NotifikasiPage from '@/pages/NotifikasiPage'
import { FullPageLoading } from './FullPageLoading'
import { NotFoundPage } from './NotFoundPage'
import { ScrollToTop } from './ScrollToTop'

// Halaman berat & seluruh panel admin dimuat malas (code-splitting) agar
// pengunjung halaman masuk/dashboard pelanggan tidak mengunduh bundel admin.
const AssistantPage = lazy(() => import('@/pages/AssistantPage'))
const BacktestPage = lazy(() => import('@/pages/BacktestPage'))
const BacktestHistoryPage = lazy(() => import('@/pages/BacktestHistoryPage'))
const BacktestDetailPage = lazy(() => import('@/pages/BacktestDetailPage'))
const BacktestEntryPage = lazy(() => import('@/pages/BacktestEntryPage'))
const LegalCenterPage = lazy(() => import('@/pages/public/legal/LegalCenterPage'))

const AdminLoginPage = lazy(() => import('@/pages/admin/AdminLoginPage'))
const AdminAuditPage = lazy(() => import('@/pages/admin/AdminAuditPage'))
const AdminCmsPage = lazy(() => import('@/pages/admin/AdminCmsPage'))
const AdminPageContentPage = lazy(() => import('@/pages/admin/AdminPageContentPage'))
const AdminLegalPage = lazy(() => import('@/pages/admin/AdminLegalPage'))
const AdminBrokerHealthPage = lazy(() => import('@/pages/admin/AdminBrokerHealthPage'))
const AdminEnginePage = lazy(() => import('@/pages/admin/AdminEnginePage'))
const AdminHealthPage = lazy(() => import('@/pages/admin/AdminHealthPage'))
const AdminRunnerPage = lazy(() => import('@/pages/admin/AdminRunnerPage'))
const AdminMaintenancePage = lazy(() => import('@/pages/admin/AdminMaintenancePage'))
const AdminSettingsPage = lazy(() => import('@/pages/admin/AdminSettingsPage'))
const AdminSupportPage = lazy(() => import('@/pages/admin/AdminSupportPage'))
const AdminCouponsPage = lazy(() => import('@/pages/admin/AdminCouponsPage'))
const AdminCreditCodesPage = lazy(() => import('@/pages/admin/AdminCreditCodesPage'))
const AdminInvoicesPage = lazy(() => import('@/pages/admin/AdminInvoicesPage'))
const AdminTrialsPage = lazy(() => import('@/pages/admin/AdminTrialsPage'))
const AdminHomePage = lazy(() => import('@/pages/admin/AdminHomePage'))
const AdminPlansPage = lazy(() => import('@/pages/admin/AdminPlansPage'))
const AdminPairsPage = lazy(() => import('@/pages/admin/AdminPairsPage'))
const AdminReportsPage = lazy(() => import('@/pages/admin/AdminReportsPage'))
const AdminRbacPage = lazy(() => import('@/pages/admin/AdminRbacPage'))
const AdminSignalsPage = lazy(() => import('@/pages/admin/AdminSignalsPage'))
const AdminSignalTracePage = lazy(() => import('@/pages/admin/AdminSignalTracePage'))
const AdminUserDetailPage = lazy(() => import('@/pages/admin/AdminUserDetailPage'))
const AdminUsersPage = lazy(() => import('@/pages/admin/AdminUsersPage'))
const AdminInvitesPage = lazy(() => import('@/pages/admin/AdminInvitesPage'))
const AdminBugReportsPage = lazy(() => import('@/pages/admin/AdminBugReportsPage'))

function RequireAuth({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuthContext()
  const location = useLocation()
  if (!isAuthenticated) {
    const intended = `${location.pathname}${location.search}`
    const shouldPreserve = intended && intended !== '/' && intended !== '/dashboard'
    const target = shouldPreserve ? `/login?next=${encodeURIComponent(intended)}` : '/login'
    return <Navigate to={target} replace />
  }
  return <>{children}</>
}

function RedirectIfAuthenticated({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuthContext()
  const location = useLocation()
  if (isAuthenticated) {
    const next = new URLSearchParams(location.search).get('next')
    return <Navigate to={resolveSafeNextPath(next)} replace />
  }
  return <>{children}</>
}

/**
 * HalamanPembayaran menutup halaman billing saat platform invite only.
 *
 * Route-nya sengaja tetap terdaftar, hanya dialihkan. Menghapus route berarti
 * membuka kembali pembayaran menuntut deploy ulang; mengalihkan cukup
 * mengubah flag. Alamat diketik langsung pun tidak menampilkan harga.
 */
function HalamanPembayaran({ children }: { children: ReactNode }) {
  const inviteOnly = useInviteOnly()
  if (inviteOnly) {
    return <Navigate to="/dashboard" replace />
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
      <Suspense fallback={<FullPageLoading label="Memuat halaman…" />}>
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
        {/* Alias di dalam panel supaya /app/daftar?invite=KODE juga bekerja.
            Alamat resmi yang dibagikan tetap /daftar?invite=KODE di akar situs. */}
        <Route
          path="/daftar"
          element={
            <RedirectIfAuthenticated>
              <RegisterPage />
            </RedirectIfAuthenticated>
          }
        />
        <Route
          path="/lupa-sandi"
          element={
            <RedirectIfAuthenticated>
              <ForgotPasswordPage />
            </RedirectIfAuthenticated>
          }
        />
        <Route
          path="/forgot-password"
          element={
            <RedirectIfAuthenticated>
              <ForgotPasswordPage />
            </RedirectIfAuthenticated>
          }
        />
        <Route
          path="/reset-password"
          element={
            <RedirectIfAuthenticated>
              <ResetPasswordPage />
            </RedirectIfAuthenticated>
          }
        />

        <Route
          path="/"
          element={<Navigate to="/dashboard" replace />}
        />
        <Route
          path="/dashboard"
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
          path="/akun/:id"
          element={
            <RequireAuth>
              <ShellPage>
                <AccountDetailPage />
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
          path="/asisten"
          element={
            <RequireAuth>
              <ShellPage>
                <Suspense fallback={<FullPageLoading label="Memuat ILA AI..." />}>
                  <AssistantPage />
                </Suspense>
              </ShellPage>
            </RequireAuth>
          }
        />
        <Route
          path="/backtest"
          element={
            <RequireAuth>
              <ShellPage>
                <BacktestPage />
              </ShellPage>
            </RequireAuth>
          }
        />
        {/* Riwayat didahulukan sebelum pola :id supaya "riwayat" tidak
            terbaca sebagai ID run. Dua rute terakhir bertingkat: satu run,
            lalu satu entry di dalamnya. */}
        <Route
          path="/backtest/riwayat"
          element={
            <RequireAuth>
              <ShellPage>
                <BacktestHistoryPage />
              </ShellPage>
            </RequireAuth>
          }
        />
        <Route
          path="/backtest/:id"
          element={
            <RequireAuth>
              <ShellPage>
                <BacktestDetailPage />
              </ShellPage>
            </RequireAuth>
          }
        />
        <Route
          path="/backtest/:id/:seq"
          element={
            <RequireAuth>
              <ShellPage>
                <BacktestEntryPage />
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
          path="/pengaturan/notifikasi"
          element={
            <RequireAuth>
              <ShellPage>
                <NotifikasiPage />
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
          path="/pengaturan/profil"
          element={
            <RequireAuth>
              <ShellPage>
                <ProfilPage />
              </ShellPage>
            </RequireAuth>
          }
        />
        <Route
          path="/pengaturan/perangkat"
          element={
            <RequireAuth>
              <ShellPage>
                <PerangkatPage />
              </ShellPage>
            </RequireAuth>
          }
        />

        <Route
          path="/laporkan-bug"
          element={
            <RequireAuth>
              <ShellPage>
                <LaporkanBugPage />
              </ShellPage>
            </RequireAuth>
          }
        />
        <Route
          path="/laporkan-bug/riwayat"
          element={
            <RequireAuth>
              <ShellPage>
                <RiwayatLaporanBugPage />
              </ShellPage>
            </RequireAuth>
          }
        />
        <Route
          path="/laporkan-bug/riwayat/:id"
          element={
            <RequireAuth>
              <ShellPage>
                <DetailLaporanBugPage />
              </ShellPage>
            </RequireAuth>
          }
        />
        <Route
          path="/langganan"
          element={
            <RequireAuth>
              <HalamanPembayaran>
                <ShellPage>
                  <BillingPage />
                </ShellPage>
              </HalamanPembayaran>
            </RequireAuth>
          }
        />
        <Route
          path="/langganan/bayar"
          element={
            <RequireAuth>
              <HalamanPembayaran>
                <ShellPage>
                  <CheckoutPage />
                </ShellPage>
              </HalamanPembayaran>
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
          path="/admin/undangan"
          element={
            <AdminPage>
              <AdminInvitesPage />
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
          path="/admin/pairs"
          element={
            <AdminPage>
              <AdminPairsPage />
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
          path="/admin/laporan-bug"
          element={
            <AdminPage>
              <AdminBugReportsPage />
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

        {/* Panel sinyal: daftar dan telusur perjalanan sinyal. */}
        <Route
          path="/admin/sinyal"
          element={
            <AdminPage>
              <RequireAdminPermission permission="signal.view">
                <AdminSignalsPage />
              </RequireAdminPermission>
            </AdminPage>
          }
        />
        <Route
          path="/admin/sinyal/:id"
          element={
            <AdminPage>
              <RequireAdminPermission permission="signal.trace">
                <AdminSignalTracePage />
              </RequireAdminPermission>
            </AdminPage>
          }
        />

        {/* RBAC: dua tab (role dan user) di satu halaman. */}
        <Route
          path="/admin/rbac"
          element={
            <AdminPage>
              <RequireAdminPermission permission="role.view">
                <AdminRbacPage />
              </RequireAdminPermission>
            </AdminPage>
          }
        />

        {/* Dokumen legal boleh dibaca tanpa masuk. */}
        <Route path="/legal" element={<LegalCenterPage />} />
        <Route path="/legal/:slug" element={<LegalCenterPage />} />

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
      </Suspense>
    </>
  )
}
