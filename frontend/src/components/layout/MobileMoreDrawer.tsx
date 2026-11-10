import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import {
  BarChart3,
  Bot,
  CreditCard,
  LogOut,
  Radio,
  Settings,
  ShieldCheck,
  Sparkles,
  Wallet,
  X,
} from 'lucide-react'
import { Badge, Button } from '@/components/ui'
import { useSignOut } from '@/features/auth/hooks/use-auth'
import type { Tier, User } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

type MobileMoreDrawerProps = {
  open: boolean
  onClose: () => void
  user: User
}

export function MobileMoreDrawer({ open, onClose, user }: MobileMoreDrawerProps) {
  const signOut = useSignOut()

  useEffect(() => {
    if (!open) return
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center lg:hidden">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 backdrop-blur-sm transition-opacity"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Bottom Sheet Modal */}
      <div className="relative z-10 flex max-h-[85vh] w-full max-w-lg flex-col overflow-hidden rounded-t-3xl border-t border-border bg-card shadow-2xl animate-in slide-in-from-bottom duration-200">
        {/* Handle Bar */}
        <div className="flex w-full justify-center pt-3 pb-1">
          <div className="h-1.5 w-10 rounded-full bg-muted-foreground/30" />
        </div>

        {/* Drawer Header */}
        <div className="flex items-center justify-between border-b border-border px-5 py-3">
          <div>
            <h3 className="text-base font-bold tracking-tight text-foreground">Semua Fitur & Menu</h3>
            <p className="text-xs text-muted-foreground">Akses cepat seluruh modul operasional ILA Platform</p>
          </div>
          <button
            type="button"
            aria-label="Tutup menu"
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-muted-foreground hover:bg-secondary/80 hover:text-foreground transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Drawer Body Links */}
        <div className="overflow-y-auto p-5 space-y-5 pb-8">
          {/* Kelompok 1: Trading & Ekosistem */}
          <div className="space-y-2">
            <span className="text-[10.5px] font-bold uppercase tracking-wider text-muted-foreground">
              Trading & Ekosistem
            </span>
            <div className="grid grid-cols-2 gap-2.5">
              <Link
                to="/sinyal"
                onClick={onClose}
                className="flex items-center gap-2.5 rounded-xl border border-border/80 bg-secondary/30 p-2.5 hover:bg-secondary/60 transition-colors"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-card text-foreground shadow-xs">
                  <Radio className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-foreground">Signals & Radar</p>
                  <p className="truncate text-[10.5px] text-muted-foreground">Setup ILA Realtime</p>
                </div>
              </Link>

              <Link
                to="/posisi"
                onClick={onClose}
                className="flex items-center gap-2.5 rounded-xl border border-border/80 bg-secondary/30 p-2.5 hover:bg-secondary/60 transition-colors"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-card text-foreground shadow-xs">
                  <BarChart3 className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-foreground">Posisi Trading</p>
                  <p className="truncate text-[10.5px] text-muted-foreground">Riwayat & Order</p>
                </div>
              </Link>

              <Link
                to="/akun"
                onClick={onClose}
                className="flex items-center gap-2.5 rounded-xl border border-border/80 bg-secondary/30 p-2.5 hover:bg-secondary/60 transition-colors"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-card text-foreground shadow-xs">
                  <Wallet className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-foreground">Akun Broker</p>
                  <p className="truncate text-[10.5px] text-muted-foreground">Match-Trader</p>
                </div>
              </Link>

              <Link
                to="/strategi"
                onClick={onClose}
                className="flex items-center gap-2.5 rounded-xl border border-border/80 bg-secondary/30 p-2.5 hover:bg-secondary/60 transition-colors"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-card text-foreground shadow-xs">
                  <Bot className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-foreground">Watchlist Pair</p>
                  <p className="truncate text-[10.5px] text-muted-foreground">Kelola Instrumen</p>
                </div>
              </Link>
            </div>
          </div>

          {/* Kelompok 2: Akun & Layanan */}
          <div className="space-y-2">
            <span className="text-[10.5px] font-bold uppercase tracking-wider text-muted-foreground">
              Akun & Layanan
            </span>
            <div className="grid grid-cols-2 gap-2.5">
              <Link
                to="/langganan"
                onClick={onClose}
                className="flex items-center gap-2.5 rounded-xl border border-border/80 bg-secondary/30 p-2.5 hover:bg-secondary/60 transition-colors"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-card text-foreground shadow-xs">
                  <CreditCard className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-foreground">Subscription</p>
                  <p className="truncate text-[10.5px] text-muted-foreground">Paket & Billing</p>
                </div>
              </Link>

              <Link
                to="/pengaturan/keamanan"
                onClick={onClose}
                className="flex items-center gap-2.5 rounded-xl border border-border/80 bg-secondary/30 p-2.5 hover:bg-secondary/60 transition-colors"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-card text-foreground shadow-xs">
                  <ShieldCheck className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-foreground">Keamanan</p>
                  <p className="truncate text-[10.5px] text-muted-foreground">Password & 2FA</p>
                </div>
              </Link>

              <Link
                to="/mulai"
                onClick={onClose}
                className="flex items-center gap-2.5 rounded-xl border border-border/80 bg-secondary/30 p-2.5 hover:bg-secondary/60 transition-colors"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-card text-foreground shadow-xs">
                  <Sparkles className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-foreground">Onboarding</p>
                  <p className="truncate text-[10.5px] text-muted-foreground">Checklist Akun</p>
                </div>
              </Link>

              <Link
                to="/pengaturan"
                onClick={onClose}
                className="flex items-center gap-2.5 rounded-xl border border-border/80 bg-secondary/30 p-2.5 hover:bg-secondary/60 transition-colors"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-card text-foreground shadow-xs">
                  <Settings className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-xs font-semibold text-foreground">Pengaturan</p>
                  <p className="truncate text-[10.5px] text-muted-foreground">Preferensi Akun</p>
                </div>
              </Link>
            </div>
          </div>

          {/* Sesi Pengguna & Tombol Keluar */}
          <div className="rounded-xl border border-border bg-secondary/30 p-3.5 space-y-3">
            <div className="flex items-center justify-between">
              <div className="min-w-0">
                <p className="truncate text-xs font-bold text-foreground">{user.name}</p>
                <p className="truncate text-[11px] text-muted-foreground">{user.email}</p>
              </div>
              <Badge variant="muted">{TIER_LABEL[user.tier]}</Badge>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="w-full gap-2 text-xs text-rose-600 hover:text-rose-700"
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
