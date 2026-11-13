import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Button, PageHeader, Section, Skeleton } from '@/components/ui'
import { EmptyState } from '@/components/ui'
import { usePositions, useSystemStatus } from '@/lib/queries'
import { formatDateTime, formatMoney, formatR } from '@/lib/format'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import { PageHeaderActions } from './components/PageHeaderActions'
import { OnboardingCard, OnboardingModal } from '@/features/onboarding'
import { AccountStatusCard } from './components/AccountStatusCard'
import { DashboardGreetingBar } from './components/DashboardGreetingBar'
import { DashboardRadarHero } from './components/DashboardRadarHero'
import { DashboardQuickActions } from './components/DashboardQuickActions'
import { KondisiHariIni, KondisiHariIniRingkas } from './components/KondisiHariIni'
import { DashboardOnboardingWidget } from './components/DashboardOnboardingWidget'
import { MobileMoreDrawer } from '@/components/layout/MobileMoreDrawer'
import { POSISI_LIMIT_DASHBOARD } from '@/features/signals/lib/kondisi-pasar'

export { POSISI_LIMIT_DASHBOARD }



function RecentPositions() {
  // Limit disamakan dengan KondisiHariIni agar keduanya berbagi satu cache.
  // Kunci cache menyertakan limit, jadi angka berbeda berarti dua permintaan
  // terpisah ke /positions setiap kali data disegarkan.
  const { data, isLoading } = usePositions(POSISI_LIMIT_DASHBOARD)

  if (isLoading) return <Skeleton className="h-32 w-full" />

  if (!data || data.length === 0) {
    return (
      <EmptyState
        title="Belum ada posisi trading"
        description="Posisi akan muncul otomatis setelah sinyal dieksekusi pada akun broker Anda."
        action={
          <Button asChild variant="outline" size="sm">
            <Link to="/sinyal">Buka Signals</Link>
          </Button>
        }
      />
    )
  }

  return (
    <ul className="divide-y divide-border border border-border">
      {data.map((position) => (
        <li
          key={position.id}
          className="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 text-[13px]"
        >
          <span className="font-semibold">{position.symbol}</span>
          <span className="text-muted-foreground">{position.direction}</span>
          {position.is_dry_run && (
            <span className="text-micro uppercase tracking-wide text-muted-foreground">
              Demo
            </span>
          )}
          <span className="ml-auto tabular font-medium">
            {position.pnl_known ? (
              <>
                {formatMoney(position.profit_loss)}{' '}
                <span className="text-muted-foreground font-mono">({formatR(position.outcome_r)})</span>
              </>
            ) : (
              // Hasil belum dilaporkan broker: 0 akan terbaca sebagai "impas".
              <span className="text-muted-foreground" title="Hasil belum dilaporkan broker">
                —
              </span>
            )}
          </span>
        </li>
      ))}
    </ul>
  )
}

export default function DashboardPage() {
  const { user } = useAuthContext()
  const { data: status } = useSystemStatus()
  const [onboardingOpen, setOnboardingOpen] = useState(false)
  const [moreDrawerOpen, setMoreDrawerOpen] = useState(false)

  return (
    <>
      {/* ============================================================ */}
      {/* 1. TAMPILAN KHUSUS MOBILE (< lg) — Persis screen.png         */}
      {/* ============================================================ */}
      <div className="flex flex-col gap-4 lg:hidden">
        {/* 2.a Greeting & Status Live Broker */}
        <DashboardGreetingBar />

        {/* 2.b Card Sinyal Radar Pasar (Top Priority) */}
        <DashboardRadarHero />

        {/* 2.c Akses Cepat (SuperApp Grid 4x2) */}
        <DashboardQuickActions
          onOpenMore={() => setMoreDrawerOpen(true)}
          onOpenOnboarding={() => setOnboardingOpen(true)}
        />

        {/*
         * 2.d Ringkasan hari ini.
         *
         * Susunan mobile dipertahankan, tetapi angka yang dulu tersebar di
         * tiga kartu (status pasar, kuota, win/loss, broker) kini diringkas
         * jadi satu baris agar tidak berulang.
         */}
        <KondisiHariIniRingkas />

        {/* 2.e Widget Tipis Penyiapan Akun */}
        <DashboardOnboardingWidget onOpenOnboarding={() => setOnboardingOpen(true)} />
      </div>

      {/* ============================================================ */}
      {/* 2. TAMPILAN DESKTOP (>= lg)                                  */}
      {/* ============================================================ */}
      <div className="hidden lg:flex lg:flex-col lg:gap-6">
        <PageHeader
          title="Dashboard"
          description="Status operasional pasar, ringkasan performa trading harian, dan posisi aktif."
          actions={<PageHeaderActions />}
        />

        <AccountStatusCard />

        <OnboardingCard />

        {/* Hero Radar Pasar & Kondisi Hari Ini di Desktop (Quick Actions khusus mobile & tab portrait) */}
        <div className="grid gap-6 lg:grid-cols-12">
          <div className="lg:col-span-7">
            <DashboardRadarHero />
          </div>
          <div className="lg:col-span-5">
            <KondisiHariIni />
          </div>
        </div>

        <Section
          title="Posisi Terkini"
          description={
            status ? `Waktu server ${formatDateTime(status.server_time)}` : 'Memuat waktu server…'
          }
          actions={
            <Button asChild variant="ghost" size="sm">
              <Link to="/posisi">Semua Posisi</Link>
            </Button>
          }
        >
          <RecentPositions />
        </Section>
      </div>

      {/* Modal Onboarding jika diklik dari Checklist Akses Cepat / Widget */}
      <OnboardingModal
        forceOpen={onboardingOpen}
        onCloseForce={() => setOnboardingOpen(false)}
      />

      {/* Bottom Sheet Drawer jika dibuka dari tombol Lainnya */}
      {user && (
        <MobileMoreDrawer
          open={moreDrawerOpen}
          onClose={() => setMoreDrawerOpen(false)}
          user={user}
        />
      )}
    </>
  )
}
