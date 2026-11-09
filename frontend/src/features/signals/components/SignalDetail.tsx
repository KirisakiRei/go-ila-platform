import {
  CheckCircle2,
  Clock,
  Layers,
  Radio,
  TrendingDown,
  TrendingUp,
} from 'lucide-react'
import {
  Badge,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  StatusPill,
} from '@/components/ui'
import { formatDateTime, formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import { SIGNAL_LABEL, SIGNAL_TONE } from '../lib/status'
import { TimeframeBadge } from './TimeframeBadge'
import { TombolEksekusi } from './TombolEksekusi'

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="tabular font-mono text-[13px] font-medium">{value}</dd>
    </div>
  )
}

/**
 * SignalDetail menampilkan satu sinyal radar dan parameter entry secara lengkap.
 */
export function SignalDetail({ signal }: { signal: Signal }) {
  const isReady = signal.status === 'READY_TO_ENTRY' || signal.status === 'OPPORTUNITY'
  const isWaitLtf = signal.status === 'WAIT_LTF_CISD'
  const isHtfSweepDone = isWaitLtf || isReady

  const dirColor =
    signal.direction === 'BUY'
      ? 'text-emerald-600 dark:text-emerald-400'
      : signal.direction === 'SELL'
        ? 'text-rose-600 dark:text-rose-400'
        : 'text-muted-foreground'

  return (
    <Card className="overflow-hidden border-border bg-card shadow-sm">
      <CardHeader className="border-b border-border/60 bg-secondary/20 pb-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-2.5">
            <CardTitle className="text-xl font-bold tracking-tight">{signal.symbol}</CardTitle>
            <span
              className={cn(
                'inline-flex items-center gap-1 font-mono text-sm font-bold',
                dirColor,
              )}
            >
              {signal.direction === 'BUY' ? (
                <TrendingUp className="h-4 w-4" />
              ) : signal.direction === 'SELL' ? (
                <TrendingDown className="h-4 w-4" />
              ) : null}
              {signal.direction}
            </span>
            <Badge variant="outline" className="font-mono text-[10.5px]">
              {signal.strategy_version}
            </Badge>
          </div>
          <div className="flex items-center gap-2">
            <TimeframeBadge htf={signal.htf} ltf={signal.ltf} />
            <StatusPill tone={SIGNAL_TONE[signal.status]}>
              {SIGNAL_LABEL[signal.status]}
            </StatusPill>
          </div>
        </div>
        <CardDescription className="flex items-center gap-1 text-[11px] text-muted-foreground">
          <Radio className="h-3 w-3 animate-pulse text-emerald-500" />
          Analisis Pasar: {signal.strategy_version.toUpperCase()} · Terbit{' '}
          {formatDateTime(signal.published_at)}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-5 pt-5">
        {/* Visual Radar Stepper: HTF vs LTF */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Progres Radar Analisis
            </span>
            <span className="text-[11px] text-muted-foreground">
              {isReady ? '2 / 2 Selesai' : isWaitLtf ? '1 / 2 Terkonfirmasi' : '0 / 2 Memantau'}
            </span>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            {/* Step 1: HTF */}
            <div
              className={cn(
                'rounded-xl border p-3.5 transition-all',
                isHtfSweepDone
                  ? 'border-emerald-500/40 bg-emerald-500/5'
                  : 'border-border bg-secondary/15',
              )}
            >
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-1.5 font-semibold text-xs">
                  {isHtfSweepDone ? (
                    <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                  ) : (
                    <Clock className="h-4 w-4 text-muted-foreground animate-spin" />
                  )}
                  <span>Tahap 1: HTF ({signal.htf || 'HTF'})</span>
                </div>
                <Badge
                  variant={isHtfSweepDone ? 'solid' : 'outline'}
                  className={cn(
                    'text-[10px]',
                    isHtfSweepDone && 'bg-emerald-600 text-white',
                  )}
                >
                  {isHtfSweepDone ? 'Valid (Closed)' : 'Memantau'}
                </Badge>
              </div>
              <p className="mt-2 text-[12px] leading-relaxed text-muted-foreground">
                {isHtfSweepDone
                  ? 'Struktur pasar timeframe besar terkonfirmasi resmi closed.'
                  : 'Menunggu penutupan candle timeframe besar pada level kunci.'}
              </p>
              {signal.htf_sweep && signal.htf_sweep.target_dol > 0 && (
                <div className="mt-2.5 flex items-center justify-between border-t border-border/50 pt-2 text-[11px] font-mono">
                  <span className="text-muted-foreground">Target Level:</span>
                  <span className="font-semibold tabular-nums">
                    {formatNumber(signal.htf_sweep.target_dol, 5)}
                  </span>
                </div>
              )}
            </div>

            {/* Step 2: LTF */}
            <div
              className={cn(
                'rounded-xl border p-3.5 transition-all',
                isReady
                  ? 'border-emerald-500/40 bg-emerald-500/5'
                  : isWaitLtf
                    ? 'border-amber-500/50 bg-amber-500/5 shadow-sm'
                    : 'border-border/60 bg-secondary/10 opacity-75',
              )}
            >
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-1.5 font-semibold text-xs">
                  {isReady ? (
                    <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                  ) : isWaitLtf ? (
                    <Clock className="h-4 w-4 text-amber-500 animate-pulse" />
                  ) : (
                    <Layers className="h-4 w-4 text-muted-foreground" />
                  )}
                  <span>Tahap 2: LTF ({signal.ltf || 'LTF'})</span>
                </div>
                <Badge
                  variant={isReady ? 'solid' : 'outline'}
                  className={cn(
                    'text-[10px]',
                    isReady && 'bg-emerald-600 text-white',
                    isWaitLtf && 'border-amber-500 text-amber-600 dark:text-amber-400',
                  )}
                >
                  {isReady ? 'Konfirmasi Siap' : isWaitLtf ? 'Menunggu Body Close' : 'Belum Dimulai'}
                </Badge>
              </div>
              <p className="mt-2 text-[12px] leading-relaxed text-muted-foreground">
                {isReady
                  ? 'Konfirmasi momentum timeframe kecil terpenuhi. Order siap dieksekusi.'
                  : isWaitLtf
                    ? 'Struktur HTF valid. Menunggu penutupan body candle pada timeframe kecil.'
                    : 'Fase ini dimulai setelah struktur timeframe besar selesai.'}
              </p>
              {signal.cisd && signal.cisd.cisd_level > 0 && (
                <div className="mt-2.5 flex items-center justify-between border-t border-border/50 pt-2 text-[11px] font-mono">
                  <span className="text-muted-foreground">Batas Konfirmasi:</span>
                  <span className="font-semibold tabular-nums">
                    {formatNumber(signal.cisd.cisd_level, 5)}
                  </span>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Parameter Order (Hanya jika harga lengkap atau status siap) */}
        {signal.entry_price > 0 ? (
          <div className="rounded-xl border border-border bg-secondary/10 p-4">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              Parameter Setup Order
            </span>
            <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-4">
              <Field label="Entry Price" value={formatNumber(signal.entry_price, 5)} />
              <Field label="Risk : Reward" value={`1 : ${formatNumber(signal.risk_reward, 2)}`} />
              <Field label="Stop Loss" value={formatNumber(signal.stop_loss, 5)} />
              <Field label="Take Profit" value={formatNumber(signal.take_profit, 5)} />
            </dl>
          </div>
        ) : (
          <div className="flex items-center gap-2 rounded-xl border border-dashed border-border bg-secondary/15 p-3.5 text-xs text-muted-foreground">
            <Clock className="h-4 w-4 shrink-0 text-amber-500" />
            <span>
              Parameter entry, stop loss, dan take profit akan dihitung seketika
              konfirmasi penutupan candle tercapai.
            </span>
          </div>
        )}

        {/* Ringkasan Analisis & Aksi Selanjutnya */}
        {signal.summary && (
          <div className="flex flex-col gap-1 rounded-xl border border-border/60 bg-card p-3">
            <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
              Keterangan Analisis
            </span>
            <p className="text-[12.5px] leading-relaxed text-foreground/90">{signal.summary}</p>
          </div>
        )}

        {signal.reject_reason && (
          <div className="border-l-2 border-foreground pl-3 py-1">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
              Penahanan Sistem
            </span>
            <p className="text-[13px] text-foreground">{signal.reject_reason}</p>
          </div>
        )}

        {signal.next_action && (
          <div className="flex items-center gap-2 rounded-xl border border-border bg-secondary/20 px-3.5 py-2.5 text-xs">
            <span className="font-semibold text-muted-foreground">Langkah Berikutnya:</span>
            <span className="font-medium text-foreground">{signal.next_action}</span>
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between border-t border-border pt-3 text-[11px] text-muted-foreground">
          <span>Terbit: {formatDateTime(signal.published_at)}</span>
          <span>Batas Waktu: {formatDateTime(signal.expires_at)}</span>
        </div>

        {/* Tombol Eksekusi jika Status Ready/Opportunity */}
        {isReady && (
          <div className="border-t border-border pt-4">
            <TombolEksekusi signal={signal} />
          </div>
        )}
      </CardContent>
    </Card>
  )
}
