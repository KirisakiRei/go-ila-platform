import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, ChevronRight } from 'lucide-react'
import { Alert, Badge, Button, PageHeader, Section, Skeleton } from '@/components/ui'
import { useTelegramLink, useTelegramUnlink } from '@/lib/queries'
import { accountFlowApi } from '@/lib/auth-api'
import { ApiError } from '@/lib/api-error'
import { useIsTabletOrMobile } from '@/lib/use-mobile-viewport'
import { PushNotificationPanel } from '@/features/notifications/PushNotificationPanel'
import { SettingsMobileView } from '@/pages/components/SettingsMobileView'

function TelegramSection() {
  const queryClient = useQueryClient()
  const link = useTelegramLink()
  const unlink = useTelegramUnlink()
  const [galat, setGalat] = useState<string | null>(null)

  const { data: onboarding, isLoading } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
  })

  const isConnected = Boolean(onboarding?.telegram_linked)

  const onLink = async () => {
    setGalat(null)
    try {
      const hasil = await link.mutateAsync()
      window.open(hasil.url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal membuat tautan Telegram.')
    }
  }

  const onUnlink = async () => {
    setGalat(null)
    try {
      await unlink.mutateAsync()
      await queryClient.invalidateQueries({ queryKey: ['me', 'onboarding'] })
      await queryClient.invalidateQueries({ queryKey: ['trial'] })
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal memutuskan Telegram.')
    }
  }

  return (
    <Section
      title="Notifikasi Telegram"
      description="Bot hanya mengirim pemberitahuan. Bot tidak pernah mengakses kredensial akun Anda."
    >
      <div className="flex flex-col gap-3 border border-border p-4">
        {isLoading ? (
          <Skeleton className="h-16 w-full" />
        ) : (
          <>
            <div className="flex items-center gap-2">
              <span className="text-[12px] font-medium text-muted-foreground">Status Bot:</span>
              {isConnected ? (
                <span className="inline-flex items-center gap-1 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:text-emerald-400">
                  <CheckCircle2 className="h-3 w-3" />
                  Terhubung
                </span>
              ) : (
                <Badge variant="outline" className="text-muted-foreground">
                  Belum Terhubung
                </Badge>
              )}
            </div>

            <p className="text-[12px] text-muted-foreground">
              {isConnected
                ? 'Akun Anda sudah terhubung ke bot resmi (@imaginaryliquiditybot). Notifikasi sinyal trading, alert risiko, dan konfirmasi entry akan otomatis dikirim ke chat ini.'
                : 'Hubungkan akun dengan bot Telegram (@imaginaryliquiditybot) untuk menerima sinyal setup, konfirmasi entry, dan peringatan darurat secara real-time.'}
            </p>

            <div className="flex flex-wrap gap-2 pt-1">
              {isConnected ? (
                <>
                  <Button
                    size="sm"
                    variant="outline"
                    isLoading={unlink.isPending}
                    onClick={() => void onUnlink()}
                  >
                    Putuskan
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    isLoading={link.isPending}
                    onClick={() => void onLink()}
                  >
                    Hubungkan Ulang / Akun Lain
                  </Button>
                </>
              ) : (
                <Button size="sm" isLoading={link.isPending} onClick={() => void onLink()}>
                  Hubungkan Telegram
                </Button>
              )}
            </div>

            {!isConnected && (
              <p className="text-[11px] text-muted-foreground">
                Tautan berlaku 5 menit dan hanya bisa dipakai sekali.
              </p>
            )}

            {galat && <Alert variant="destructive">{galat}</Alert>}
            {unlink.isSuccess && <Alert variant="success">Chat Telegram sudah diputus.</Alert>}
          </>
        )}
      </div>
    </Section>
  )
}

export default function SettingsPage() {
  const isTabletOrMobile = useIsTabletOrMobile()

  return (
    <>
      {/* Tampilan Mobile (< lg) */}
      <SettingsMobileView />

      {/* Tampilan Desktop (>= lg) */}
      {!isTabletOrMobile && (
        <div className="hidden flex-col gap-6 lg:flex">
          <PageHeader
            title="Pengaturan"
            description="Akun, saluran notifikasi, dan keamanan."
          />

          <Section
            title="Notifikasi Web Push (Browser & Aplikasi)"
            description="Terima peringatan instan di browser desktop atau ponsel hanya untuk sinyal siap entry, posisi dibuka, dan posisi ditutup."
          >
            <div className="border border-border p-4">
              <PushNotificationPanel />
            </div>
          </Section>

          <TelegramSection />

          <Section title="Akun & Keamanan" description="Pilih bagian yang ingin diatur.">
            <div className="divide-y divide-border border border-border">
              {[
                { to: '/pengaturan/profil', judul: 'Profil & Akun', isi: 'Data diri dan informasi kontak' },
                { to: '/pengaturan/keamanan', judul: 'Kata Sandi & Email', isi: 'Kata sandi, verifikasi email, dan 2FA' },
                { to: '/pengaturan/perangkat', judul: 'Perangkat & Sesi', isi: 'Perangkat yang masuk dan keluar otomatis' },
              ].map((menu) => (
                <Link
                  key={menu.to}
                  to={menu.to}
                  className="flex items-center justify-between gap-4 px-4 py-3 transition-colors duration-150 hover:bg-secondary"
                >
                  <span className="min-w-0">
                    <span className="block text-[13px] font-medium text-foreground">{menu.judul}</span>
                    <span className="block text-[12px] text-muted-foreground">{menu.isi}</span>
                  </span>
                  <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
                </Link>
              ))}
            </div>
          </Section>
        </div>
      )}
    </>
  )
}
