import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Link, Route, Routes } from 'react-router-dom'
import { FullPageLoading } from '@/app/FullPageLoading'
import LegalPage from '@/pages/LegalPage'
import AboutPage from '@/pages/public/AboutPage'
import FeaturesPage from '@/pages/public/FeaturesPage'
import HelpCenterPage from '@/pages/public/HelpCenterPage'
import LandingPage from '@/pages/public/LandingPage'
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
        <Route path="legal" element={<LegalPage />} />
        <Route path="legal/:slug" element={<LegalPage />} />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}

export function PublicApp({ children }: { children?: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <PublicRoutes />
        {children}
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export { FullPageLoading }
