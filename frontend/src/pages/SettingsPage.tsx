import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { PageHeader, Section } from '@/components/ui'
import { useIsTabletOrMobile } from '@/lib/use-mobile-viewport'
import { SettingsMobileView } from '@/pages/components/SettingsMobileView'

const MENU_PENGATURAN = [
  { to: '/pengaturan/notifikasi', judul: 'Notifikasi', isi: 'Telegram dan Web Push' },
  { to: '/pengaturan/profil', judul: 'Profil & Akun', isi: 'Data diri dan informasi kontak' },
  { to: '/pengaturan/keamanan', judul: 'Kata Sandi & Email', isi: 'Kata sandi, verifikasi email, dan 2FA' },
  { to: '/pengaturan/perangkat', judul: 'Perangkat & Sesi', isi: 'Perangkat yang masuk dan keluar otomatis' },
] as const

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
            description="Notifikasi, profil akun, dan keamanan."
          />

          <Section title="Menu Pengaturan">
            <div className="divide-y divide-border rounded-xl border border-border bg-card overflow-hidden">
              {MENU_PENGATURAN.map((menu) => (
                <Link
                  key={menu.to}
                  to={menu.to}
                  className="flex items-center justify-between gap-4 px-4 py-3.5 transition-colors duration-150 hover:bg-secondary/60"
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
