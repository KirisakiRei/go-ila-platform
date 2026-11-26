import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  BarChart3,
  Bot,
  Bug,
  CreditCard,
  FlaskConical,
  LogOut,
  MessageSquareText,
  Radio,
  Settings,
  ShieldCheck,
  Sparkles,
  Wallet,
  X,
} from 'lucide-react'
import { Badge, Button, MobileMenuGroup, MobileMenuRow } from '@/components/ui'
import { useSignOut } from '@/features/auth/hooks/use-auth'
import type { Tier, User } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

type MobileMoreDrawerProps = {
  open: boolean
  onClose: () => void
  user: User
  /** Saat invite only, menu Subscription disembunyikan. */
  inviteOnly?: boolean
}

export function MobileMoreDrawer({ open, onClose, user, inviteOnly = false }: MobileMoreDrawerProps) {
  const signOut = useSignOut()
  const navigate = useNavigate()

  useEffect(() => {
    if (!open) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose])

  if (!open) return null

  const go = (path: string) => {
    onClose()
    navigate(path)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center lg:hidden">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 backdrop-blur-sm transition-opacity"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Bottom Sheet Modal */}
      <div className="relative z-10 flex max-h-[85dvh] w-full max-w-lg flex-col overflow-hidden rounded-t-3xl border-t border-border bg-card shadow-2xl animate-in slide-in-from-bottom duration-200">
        {/* Handle Bar */}
        <div className="flex w-full justify-center pt-3 pb-1">
          <div className="h-1.5 w-10 rounded-full bg-muted-foreground/30" />
        </div>

        {/* Drawer Header */}
        <div className="flex items-center justify-between gap-2 border-b border-border px-4 py-3">
          <div className="min-w-0">
            <h3 className="truncate text-base font-bold tracking-tight text-foreground">
              Semua Fitur & Menu
            </h3>
            <p className="truncate text-xs text-muted-foreground">
              Akses cepat seluruh modul operasional ILA Platform
            </p>
          </div>
          <button
            type="button"
            aria-label="Tutup menu"
            onClick={onClose}
            className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-secondary text-muted-foreground transition-colors hover:bg-secondary/80 hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Drawer Body Links */}
        <div className="space-y-4 overflow-y-auto px-4 py-4 pb-[calc(1.75rem+env(safe-area-inset-bottom))]">
          <MobileMenuGroup title="Trading & Ekosistem">
            <MobileMenuRow
              icon={Radio}
              label="Signals & Radar"
              description="Pantauan struktur pasar & sinyal ILA realtime"
              onClick={() => go('/sinyal')}
            />
            <MobileMenuRow
              icon={BarChart3}
              label="Posisi Trading"
              description="Jurnal eksekusi, riwayat order, & evaluasi setup"
              onClick={() => go('/posisi')}
            />
            <MobileMenuRow
              icon={BarChart3}
              label="Jurnal Manual"
              description="Catatan trade mandiri, modal, & evaluasi"
              onClick={() => go('/jurnal')}
            />
            <MobileMenuRow
              icon={Wallet}
              label="Akun Broker"
              description="Kelola koneksi cTrader & Match-Trader"
              onClick={() => go('/akun')}
            />
            <MobileMenuRow
              icon={Bot}
              label="Watchlist Pair"
              description="Pilih instrumen trading yang dipantau"
              onClick={() => go('/strategi')}
            />
            <MobileMenuRow
              icon={MessageSquareText}
              label="ILA AI"
              description="Asisten analisis New ILA 2.0 & data akun"
              onClick={() => go('/asisten')}
            />
            <MobileMenuRow
              icon={FlaskConical}
              label="Backtest"
              description="Simulasi historis aturan trading ILA"
              onClick={() => go('/backtest')}
            />
          </MobileMenuGroup>

          <MobileMenuGroup title="Akun & Layanan">
            {!inviteOnly && (
              <MobileMenuRow
                icon={CreditCard}
                label="Subscription & Billing"
                description="Kelola paket langganan & riwayat tagihan"
                onClick={() => go('/langganan')}
              />
            )}
            <MobileMenuRow
              icon={Settings}
              label="Pengaturan"
              description="Profil, notifikasi push/Telegram, & preferensi"
              onClick={() => go('/pengaturan')}
            />
            <MobileMenuRow
              icon={ShieldCheck}
              label="Keamanan Akun"
              description="Kata sandi, verifikasi 2FA, & sesi perangkat"
              onClick={() => go('/pengaturan/keamanan')}
            />
            <MobileMenuRow
              icon={Sparkles}
              label="Onboarding"
              description="Daftar langkah persiapan awal akun"
              onClick={() => go('/mulai')}
            />
            <MobileMenuRow
              icon={Bug}
              label="Laporkan Bug"
              description="Kirim laporan kendala ke tim teknis"
              onClick={() => go('/laporkan-bug')}
            />
          </MobileMenuGroup>

          {/* Sesi Pengguna & Tombol Keluar */}
          <div className="space-y-3 rounded-2xl border border-border bg-secondary/30 p-3.5">
            <div className="flex items-center justify-between gap-2">
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-bold text-foreground">{user.name}</p>
                <p className="truncate text-[11px] text-muted-foreground">{user.email}</p>
              </div>
              <Badge variant="muted" className="shrink-0">
                {TIER_LABEL[user.tier]}
              </Badge>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="w-full gap-2 text-xs text-rose-600 hover:text-rose-700 dark:text-rose-400 dark:hover:text-rose-300"
              onClick={() => {
                onClose()
                signOut()
              }}
            >
              <LogOut className="h-3.5 w-3.5" />
              Keluar dari Sesi
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
