import { useEffect, type ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Link, Route, Routes } from 'react-router-dom'
import { FullPageLoading } from '@/app/FullPageLoading'
import { ScrollToTop } from '@/app/ScrollToTop'
import ForgotPasswordPage from '@/pages/ForgotPasswordPage'
import RegisterPage from '@/pages/RegisterPage'
import ResetPasswordPage from '@/pages/ResetPasswordPage'
import VerifyEmailPage from '@/pages/VerifyEmailPage'
import AboutPage from '@/pages/public/AboutPage'
import FeaturesPage from '@/pages/public/FeaturesPage'
import HelpCenterPage from '@/pages/public/HelpCenterPage'
import LandingPage from '@/pages/public/LandingPage'
import LegalCenterPage from '@/pages/public/legal/LegalCenterPage'
import PricingPage from '@/pages/public/PricingPage'
import { PublicLayout } from '@/pages/public/PublicLayout'

/**
 * Entri publik memakai cache sendiri dan tidak memuat AuthProvider.
 *
 * Panel pelanggan tinggal di `/app/*` dan dibuka entri Vite terpisah, sehingga
 * pengunjung tanpa akun tidak mengunduh bundel dashboard (plans/11 J5b).
 */
const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 60_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 },
  },
})

/** RedirectTo memindahkan rute akar ke entri panel tanpa menambah bundel. */
function RedirectTo({ to }: { to: string }) {
  useEffect(() => {
    window.location.replace(to)
  }, [to])
  return <FullPageLoading label="Mengalihkan…" />
}

function NotFound() {
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col items-center gap-2 px-4 py-24 text-center">
      <p className="text-lg font-semibold">Halaman tidak ditemukan</p>
      <p className="text-[13px] text-muted-foreground">Tautan mungkin sudah dipindahkan.</p>
      <Link to="/" className="mt-2 rounded-md border border-border px-3 py-1.5 text-[13px]">
        Kembali ke beranda
      </Link>
    </div>
  )
}

function PublicRoutes() {
  return (
    <Routes>
      <Route element={<PublicLayout />}>
        <Route index element={<LandingPage />} />
        <Route path="fitur" element={<FeaturesPage />} />
        <Route path="harga" element={<PricingPage />} />
        <Route path="tentang" element={<AboutPage />} />
        <Route path="bantuan" element={<HelpCenterPage />} />
        <Route path="bantuan/:topik" element={<HelpCenterPage />} />
        <Route path="legal" element={<LegalCenterPage />} />
        <Route path="legal/:slug" element={<LegalCenterPage />} />

        {/* Alur akun tanpa sesi: tautan dari surel menunjuk ke akar situs,
            bukan ke /app, sehingga rutenya harus ada di entri ini. */}
        <Route path="lupa-sandi" element={<ForgotPasswordPage />} />
        <Route path="reset-password" element={<ResetPasswordPage />} />
        <Route path="verifikasi" element={<VerifyEmailPage />} />

        {/* Tautan undangan admin berbentuk /daftar?invite=KODE dan dibagikan ke
            komunitas, jadi alamat itu harus dilayani entri ini apa adanya.
            Rute ini TIDAK boleh dialihkan ke /app/register: basename /app akan
            terpasang di depan path dan kode undangan hilang dari alamat. */}
        <Route path="daftar" element={<RegisterPage />} />

        {/* Masuk dilayani entri panel di /app/*; tautan akar diarahkan ke sana
            agar alamat yang dibagikan tetap satu bentuk. Pendaftaran mandiri
            tidak dialihkan karena sudah tertutup dan tautan undangan menang. */}
        <Route path="login" element={<RedirectTo to="/app/login" />} />
        <Route path="register" element={<RedirectTo to="/app/register" />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}

export function PublicApp({ children }: { children?: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ScrollToTop />
        <PublicRoutes />
        {children}
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export { FullPageLoading }
