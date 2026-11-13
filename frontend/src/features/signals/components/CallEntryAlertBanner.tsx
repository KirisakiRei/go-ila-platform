import { useEffect, useState } from 'react'
import {
  AlertTriangle,
  Check,
  Clock,
  Loader2,
  X,
  Zap,
} from 'lucide-react'
import { Badge, Button, Card, CardContent } from '@/components/ui'
import { formatDualDateTime, formatNumber } from '@/lib/format'
import { useConfirmApproval, useDeclineApproval, useIssueApproval } from '@/lib/queries'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import { getCycleBadge, getIla3Mtf } from '../lib/status'
import { TimeframeBadge } from './TimeframeBadge'

function sisaWaktu(expiresAt: string, sekarang: number) {
  const selesai = new Date(expiresAt).getTime()
  if (!Number.isFinite(selesai)) return { habis: true, detik: 0 }
  const detik = Math.floor((selesai - sekarang) / 1000)
  return { habis: detik <= 0, detik: Math.max(0, detik) }
}

function formatSisa(detik: number) {
  const menit = Math.floor(detik / 60)
  const sisaDetik = detik % 60
  return `${menit}m ${String(sisaDetik).padStart(2, '0')}s`
}

export function CallEntryAlertBanner({
  signals,
  onSelesai,
}: {
  signals: Signal[]
  onSelesai?: () => void
}) {
  const [sekarang, setSekarang] = useState(() => Date.now())

  useEffect(() => {
    const timer = window.setInterval(() => setSekarang(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [])

  // Hanya tampilkan sinyal yang READY_TO_ENTRY atau OPPORTUNITY dan BELUM expired
  const activeEntries = signals
    .filter((s) => {
      if (s.status !== 'READY_TO_ENTRY' && s.status !== 'OPPORTUNITY') {
        return false
      }
      const expiredTime = new Date(s.expires_at).getTime()
      return Number.isFinite(expiredTime) && expiredTime > sekarang
    })
    .sort((a, b) => new Date(b.published_at).getTime() - new Date(a.published_at).getTime())

  const [selectedIndex, setSelectedIndex] = useState(0)

  // Jika tidak ada sinyal aktif yang belum expired, card alert TIDAK ditampilkan
  if (activeEntries.length === 0) {
    return null
  }

  // Pastikan index aman bila jumlah sinyal berubah
  const safeIndex = selectedIndex < activeEntries.length ? selectedIndex : 0
  const activeSignal = activeEntries[safeIndex]

  if (!activeSignal) {
    return null
  }

  return (
    <SingleCallEntryCard
      signal={activeSignal}
      totalEntries={activeEntries.length}
      currentIndex={safeIndex}
      onSelectIndex={setSelectedIndex}
      onSelesai={onSelesai}
    />
  )
}

function SingleCallEntryCard({
  signal,
  totalEntries,
  currentIndex,
  onSelectIndex,
  onSelesai,
}: {
  signal: Signal
  totalEntries: number
  currentIndex: number
  onSelectIndex: (idx: number) => void
  onSelesai?: () => void
}) {
  const issue = useIssueApproval()
  const konfirmasi = useConfirmApproval()
  const tolak = useDeclineApproval()

  const [sekarang, setSekarang] = useState(() => Date.now())
  useEffect(() => {
    const sisa = sisaWaktu(signal.expires_at, Date.now())
    if (sisa.habis) return
    const timer = window.setInterval(() => setSekarang(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [signal.expires_at])

  const sisa = sisaWaktu(signal.expires_at, sekarang)
  const sibuk = issue.isPending || konfirmasi.isPending || tolak.isPending
  const bisaDieksekusi = signal.can_confirm && !sisa.habis

  const isILA3 = signal.strategy_version === 'ila-3.0'
  const mtf = isILA3 ? getIla3Mtf(signal.htf, signal.ltf) : undefined
  const cycle = isILA3 ? getCycleBadge(signal.strategy_version, signal.htf, signal.ltf) : null

  // Hitung selisih pip perkiraan
  const pipsMultiplier = signal.symbol.toUpperCase().includes('XAU') ? 10 : 10000
  const slPips = Math.abs(signal.entry_price - signal.stop_loss) * pipsMultiplier
  const tpPips = Math.abs(signal.take_profit - signal.entry_price) * pipsMultiplier

  const eksekusi = async () => {
    const hasil = await issue.mutateAsync(signal.signal_id)
    if (!hasil.can_confirm) return
    await konfirmasi.mutateAsync(hasil.token)
    onSelesai?.()
  }

  const lewati = async () => {
    const hasil = await issue.mutateAsync(signal.signal_id)
    if (!hasil.token) return
    await tolak.mutateAsync(hasil.token)
    onSelesai?.()
  }

  const blokir = issue.data && !issue.data.can_confirm ? issue.data.blocked_reason : ''
  const pesan = konfirmasi.data?.message ?? konfirmasi.error?.message ?? tolak.data?.message ?? tolak.error?.message

  return (
    <Card className="relative overflow-hidden border border-border bg-card shadow-sm">
      {/* Top Banner Alert Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border bg-muted/40 px-4 py-3">
        <div className="flex flex-wrap items-center gap-2.5">
          <Badge
            variant="default"
            className="font-mono text-xs font-semibold gap-1.5 px-2.5 py-1 tracking-wider"
          >
            <Zap className="h-3.5 w-3.5 fill-current" />
            CALL ENTRY · {signal.direction}
          </Badge>

          <span className="text-base font-bold tracking-tight text-foreground font-mono">
            {signal.symbol}
          </span>

          <TimeframeBadge htf={signal.htf} mtf={mtf} ltf={signal.ltf} />

          <Badge variant="outline" className="font-mono text-micro">
            {signal.strategy_version}
          </Badge>

          {cycle && (
            <Badge variant="outline" className="text-micro">
              {cycle}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-3 text-xs">
          {/* Multi-signal selector if more than 1 setup is active */}
          {totalEntries > 1 && (
            <div className="flex items-center gap-1.5 border-r border-border/60 pr-3">
              <span className="text-micro font-medium text-muted-foreground">Setup:</span>
              {Array.from({ length: totalEntries }).map((_, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={() => onSelectIndex(i)}
                  className={cn(
                    'flex h-5 w-5 items-center justify-center rounded text-micro font-mono font-semibold transition-colors',
                    i === currentIndex
                      ? 'bg-foreground text-background'
                      : 'bg-secondary text-muted-foreground hover:bg-secondary/80',
                  )}
                >
                  {i + 1}
                </button>
              ))}
            </div>
          )}

          <span
            className={cn(
              'inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 font-mono text-xs font-medium tabular-nums border border-border bg-background',
              sisa.detik <= 180 && 'border-foreground font-bold animate-pulse',
            )}
          >
            <Clock className="h-3.5 w-3.5" />
            Sisa Waktu: {formatSisa(sisa.detik)}
          </span>
        </div>
      </div>

      <CardContent className="p-4 sm:p-5 space-y-4">
        {/* Parameter Order Grid */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div className="rounded-lg border border-border bg-muted/20 p-3">
            <span className="text-micro font-semibold uppercase tracking-wider text-muted-foreground">
              Entry Price
            </span>
            <p className="mt-1 font-mono text-lg font-bold tabular-nums text-foreground">
              {formatNumber(signal.entry_price, 5)}
            </p>
            <p className="text-micro text-muted-foreground">Harga Pembuka Sinyal</p>
          </div>

          <div className="rounded-lg border border-border bg-muted/20 p-3">
            <span className="text-micro font-semibold uppercase tracking-wider text-muted-foreground">
              Stop Loss
            </span>
            <p className="mt-1 font-mono text-lg font-bold tabular-nums text-foreground">
              {formatNumber(signal.stop_loss, 5)}
            </p>
            <p className="text-micro text-muted-foreground">
              -{formatNumber(slPips, 1)} pips batas risiko
            </p>
          </div>

          <div className="rounded-lg border border-border bg-muted/20 p-3">
            <span className="text-micro font-semibold uppercase tracking-wider text-muted-foreground">
              Take Profit (DOL)
            </span>
            <p className="mt-1 font-mono text-lg font-bold tabular-nums text-foreground">
              {formatNumber(signal.take_profit, 5)}
            </p>
            <p className="text-micro text-muted-foreground">
              +{formatNumber(tpPips, 1)} pips target likuiditas
            </p>
          </div>

          <div className="rounded-lg border border-border bg-muted/20 p-3">
            <span className="text-micro font-semibold uppercase tracking-wider text-muted-foreground">
              Risk : Reward
            </span>
            <p className="mt-1 font-mono text-lg font-bold tabular-nums text-foreground">
              1 : {formatNumber(signal.risk_reward, 2)}
            </p>
            <p className="text-micro text-muted-foreground">Rasio Keuntungan</p>
          </div>
        </div>

        {/* Action Controls & Info Bar */}
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-3.5">
          <div className="text-caption text-muted-foreground flex items-center gap-1.5 flex-wrap">
            <span>Terbit: {formatDualDateTime(signal.published_at)}</span>
            <span>·</span>
            <span>Konfirmasi: LTF CISD Valid</span>
          </div>

          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={sisa.habis || sibuk}
              onClick={() => void lewati()}
              className="h-8 gap-1.5 text-xs font-medium"
            >
              <X className="h-3.5 w-3.5" />
              Lewati
            </Button>

            <Button
              size="sm"
              variant="default"
              disabled={!bisaDieksekusi || sibuk}
              onClick={() => void eksekusi()}
              className="h-8 gap-1.5 text-xs font-semibold px-4 shadow-sm"
            >
              {konfirmasi.isPending || issue.isPending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Check className="h-3.5 w-3.5" />
              )}
              Eksekusi Order Sekarang
            </Button>
          </div>
        </div>

        {/* Warning / Error Notifications */}
        {blokir && (
          <div className="flex items-center gap-2 rounded-md bg-muted px-3 py-2 text-xs text-foreground border border-border">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span>{blokir}</span>
          </div>
        )}

        {pesan && (
          <div className="rounded-md bg-muted px-3 py-2 text-xs border border-border text-foreground">
            {pesan}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
