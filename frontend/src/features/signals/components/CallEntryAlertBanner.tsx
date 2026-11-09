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
  // Ambil hanya sinyal yang READY_TO_ENTRY atau OPPORTUNITY
  const activeEntries = signals
    .filter((s) => s.status === 'READY_TO_ENTRY' || s.status === 'OPPORTUNITY')
    .sort((a, b) => new Date(b.published_at).getTime() - new Date(a.published_at).getTime())

  const [selectedIndex, setSelectedIndex] = useState(0)

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

  const isBuy = signal.direction === 'BUY'
  const isSell = signal.direction === 'SELL'
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
    <Card
      className={cn(
        'relative overflow-hidden border-2 shadow-md transition-all',
        isBuy
          ? 'border-emerald-500/80 bg-emerald-500/[0.03]'
          : isSell
            ? 'border-rose-500/80 bg-rose-500/[0.03]'
            : 'border-primary/60 bg-card',
      )}
    >
      {/* Top Banner Alert Bar */}
      <div
        className={cn(
          'flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3',
          isBuy
            ? 'border-emerald-500/30 bg-emerald-500/10'
            : isSell
              ? 'border-rose-500/30 bg-rose-500/10'
              : 'border-border bg-secondary/30',
        )}
      >
        <div className="flex flex-wrap items-center gap-2.5">
          <span
            className={cn(
              'inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 font-mono text-xs font-bold tracking-wide',
              isBuy
                ? 'bg-emerald-600 text-white'
                : isSell
                  ? 'bg-rose-600 text-white'
                  : 'bg-foreground text-background',
            )}
          >
            <Zap className="h-3.5 w-3.5 fill-current" />
            CALL ENTRY {signal.direction}
          </span>

          <span className="text-base font-bold tracking-tight text-foreground">
            {signal.symbol}
          </span>

          <TimeframeBadge htf={signal.htf} mtf={mtf} ltf={signal.ltf} />

          <Badge variant="outline" className="font-mono text-[10px]">
            {signal.strategy_version}
          </Badge>

          {cycle && (
            <Badge variant="outline" className="text-[10px] border-primary/20 bg-primary/5 text-primary">
              {cycle}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-3 text-xs">
          {/* Multi-signal selector if more than 1 setup is active */}
          {totalEntries > 1 && (
            <div className="flex items-center gap-1.5 border-r border-border/60 pr-3">
              <span className="text-[11px] font-medium text-muted-foreground">Setup:</span>
              {Array.from({ length: totalEntries }).map((_, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={() => onSelectIndex(i)}
                  className={cn(
                    'flex h-5 w-5 items-center justify-center rounded text-[11px] font-mono font-semibold transition-colors',
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

          {sisa.habis ? (
            <span className="inline-flex items-center gap-1 rounded bg-secondary px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
              <Clock className="h-3 w-3" />
              Batas waktu habis
            </span>
          ) : (
            <span
              className={cn(
                'inline-flex items-center gap-1 rounded px-2.5 py-0.5 font-mono text-xs font-semibold tabular-nums',
                sisa.detik <= 180
                  ? 'bg-rose-500/15 text-rose-600 dark:text-rose-400 animate-pulse'
                  : 'bg-secondary text-foreground',
              )}
            >
              <Clock className="h-3.5 w-3.5" />
              Sisa Waktu: {formatSisa(sisa.detik)}
            </span>
          )}
        </div>
      </div>

      <CardContent className="p-4 sm:p-5 space-y-4">
        {/* Parameter Order Grid */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div className="rounded-lg border border-border/80 bg-card p-3 shadow-2xs">
            <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
              Entry Price
            </span>
            <p className="mt-1 font-mono text-lg font-bold tabular-nums text-foreground">
              {formatNumber(signal.entry_price, 5)}
            </p>
            <p className="text-[11px] text-muted-foreground">Harga Pembuka Sinyal</p>
          </div>

          <div className="rounded-lg border border-border/80 bg-card p-3 shadow-2xs">
            <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
              Stop Loss
            </span>
            <p className="mt-1 font-mono text-lg font-bold tabular-nums text-rose-600 dark:text-rose-400">
              {formatNumber(signal.stop_loss, 5)}
            </p>
            <p className="text-[11px] text-muted-foreground">
              -{formatNumber(slPips, 1)} pips batas risiko
            </p>
          </div>

          <div className="rounded-lg border border-border/80 bg-card p-3 shadow-2xs">
            <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
              Take Profit (DOL)
            </span>
            <p className="mt-1 font-mono text-lg font-bold tabular-nums text-emerald-600 dark:text-emerald-400">
              {formatNumber(signal.take_profit, 5)}
            </p>
            <p className="text-[11px] text-muted-foreground">
              +{formatNumber(tpPips, 1)} pips target likuiditas
            </p>
          </div>

          <div className="rounded-lg border border-border/80 bg-card p-3 shadow-2xs">
            <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
              Risk : Reward
            </span>
            <p className="mt-1 font-mono text-lg font-bold tabular-nums text-foreground">
              1 : {formatNumber(signal.risk_reward, 2)}
            </p>
            <p className="text-[11px] text-muted-foreground">Rasio Keuntungan</p>
          </div>
        </div>

        {/* Action Controls & Info Bar */}
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border/60 pt-3.5">
          <div className="text-[11.5px] text-muted-foreground flex items-center gap-1.5 flex-wrap">
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
              disabled={!bisaDieksekusi || sibuk}
              onClick={() => void eksekusi()}
              className={cn(
                'h-8 gap-1.5 text-xs font-semibold px-4 shadow-sm',
                isBuy
                  ? 'bg-emerald-600 text-white hover:bg-emerald-700'
                  : isSell
                    ? 'bg-rose-600 text-white hover:bg-rose-700'
                    : '',
              )}
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
          <div className="flex items-center gap-2 rounded-md bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-400 border border-amber-500/20">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span>{blokir}</span>
          </div>
        )}

        {pesan && (
          <div
            className={cn(
              'rounded-md px-3 py-2 text-xs border',
              konfirmasi.isError || tolak.isError
                ? 'bg-rose-500/10 text-rose-700 dark:text-rose-400 border-rose-500/20'
                : 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border-emerald-500/20',
            )}
          >
            {pesan}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
