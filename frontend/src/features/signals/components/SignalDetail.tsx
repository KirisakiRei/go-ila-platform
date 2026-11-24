import {
  CheckCircle2,
  Clock,
  Compass,
  Crosshair,
  Layers,
  Radio,
  Target,
  TrendingDown,
  TrendingUp,
  XCircle,
  Zap,
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
import { formatDualDateTime, formatNumber, formatPrice } from '@/lib/format'
import { priceDigits } from '@/lib/pricePrecision'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import { buildIla2Timeline, type Ila2StepState } from '../lib/ila2-progress'
import { SIGNAL_LABEL, SIGNAL_TONE, getCycleBadge, getIla3Mtf, getIla3Progress } from '../lib/status'
import { TimeframeBadge } from './TimeframeBadge'
import { TombolEksekusi } from './TombolEksekusi'

type AnalysisPoint = {
  id: string
  title: string
  detail: string
  iconType: 'sweep' | 'trigger' | 'target' | 'context'
}

function parseAnalysisSummary(rawSummary: string, strategyVersion?: string): AnalysisPoint[] {
  if (!rawSummary || !rawSummary.trim()) return []

  const cleaned = rawSummary.replace(/^💎\s*SETUP[^\n:]+:\s*/gi, '').trim()
  const rawParts = cleaned
    .split(/(?:•|\n)+/)
    .map((p) => p.trim())
    .filter(Boolean)

  const points: AnalysisPoint[] = []

  for (let i = 0; i < rawParts.length; i++) {
    const part = rawParts[i]
    if (!part) continue

    // Abaikan baris redundan "Level: Entry=..." karena sudah ditampilkan di kotak parameter harga
    if (/^level:\s*entry=/i.test(part) || /^entry=/i.test(part)) {
      continue
    }

    const cleanDetail = part.replace(/^[•\s\-_]+/, '').replace(/^💎\s*/, '')
    if (!cleanDetail) continue

    if (strategyVersion === 'ila-3.0') {
      if (/htf|macro|q1/i.test(cleanDetail)) {
        points.push({
          id: `htf-${i}`,
          title: 'HTF Macro Bias & Key Level',
          detail: cleanDetail,
          iconType: 'sweep',
        })
      } else if (/mtf|reversal|retrace|internal|smt/i.test(cleanDetail)) {
        points.push({
          id: `mtf-${i}`,
          title: 'MTF Structure & Alignment',
          detail: cleanDetail,
          iconType: 'trigger',
        })
      } else if (/ltf|cisd|entry\s*1|entry\s*2|trigger|break/i.test(cleanDetail)) {
        points.push({
          id: `ltf-${i}`,
          title: 'LTF Trigger (CISD)',
          detail: cleanDetail,
          iconType: 'trigger',
        })
      } else if (/stdev|target|tp\s*1|dol|proyeksi|fibo/i.test(cleanDetail)) {
        points.push({
          id: `target-${i}`,
          title: 'Target & Risk-Reward',
          detail: cleanDetail,
          iconType: 'target',
        })
      } else {
        points.push({
          id: `point-${i}`,
          title: 'Market Structure Context',
          detail: cleanDetail,
          iconType: 'context',
        })
      }
    } else {
      if (/sweep|fvg|sibi|bisi|c2|masuk ke|menyapu|liquidity|erl|irl|htf/i.test(cleanDetail)) {
        points.push({
          id: `sweep-${i}`,
          title: 'HTF Key Level (C2 Closure)',
          detail: cleanDetail,
          iconType: 'sweep',
        })
      } else if (/cisd|trigger|break|tembus|valid|konfirmasi|reversal|ltf/i.test(cleanDetail)) {
        points.push({
          id: `trigger-${i}`,
          title: 'LTF Trigger (CISD)',
          detail: cleanDetail,
          iconType: 'trigger',
        })
      } else if (/stdev|target|tp\s*1|dol|rr|proyeksi|fibo/i.test(cleanDetail)) {
        points.push({
          id: `target-${i}`,
          title: 'Target & Risk-Reward',
          detail: cleanDetail,
          iconType: 'target',
        })
      } else {
        points.push({
          id: `point-${i}`,
          title: 'Market Structure Context',
          detail: cleanDetail,
          iconType: 'context',
        })
      }
    }
  }

  return points
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="tabular font-mono text-[13px] font-medium">{value}</dd>
    </div>
  )
}

function StepMarker({ state }: { state: Ila2StepState }) {
  if (state === 'done') {
    return <CheckCircle2 className="h-[18px] w-[18px] text-emerald-600 dark:text-emerald-400 bg-card rounded-full" />
  }
  if (state === 'active') {
    return <Clock className="h-[18px] w-[18px] text-amber-600 dark:text-amber-400 bg-card rounded-full animate-pulse" />
  }
  if (state === 'invalid') {
    return <XCircle className="h-[18px] w-[18px] text-rose-500 bg-card rounded-full" />
  }
  return <span className="block h-[18px] w-[18px] rounded-full border-2 border-border bg-card" />
}

/**
 * SignalDetail menampilkan satu sinyal radar dan parameter entry secara lengkap.
 */
export function SignalDetail({ signal }: { signal: Signal }) {
  const isILA3 = signal.strategy_version === 'ila-3.0'
  const ila3Mtf = isILA3 ? getIla3Mtf(signal.htf, signal.ltf) : undefined
  const cycleName = getCycleBadge(signal.strategy_version, signal.htf, signal.ltf)
  const ila3Prog = isILA3 ? getIla3Progress(signal.status) : null
  const ila2 = isILA3 ? null : buildIla2Timeline(signal)
  const digits = priceDigits(signal.symbol, signal.entry_price || signal.cisd?.cisd_level)

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
            {cycleName && (
              <Badge variant="outline" className="text-[10px] font-medium border-primary/20 bg-primary/5 text-primary">
                {cycleName}
              </Badge>
            )}
          </div>
          <div className="flex items-center gap-2">
            <TimeframeBadge htf={signal.htf} mtf={ila3Mtf} ltf={signal.ltf} />
            <StatusPill tone={SIGNAL_TONE[signal.status]}>
              {SIGNAL_LABEL[signal.status]}
            </StatusPill>
          </div>
        </div>
        <CardDescription className="flex items-center gap-1 text-[11px] text-muted-foreground">
          <Radio className="h-3 w-3 animate-pulse text-emerald-500" />
          Analisis Pasar: {signal.strategy_version.toUpperCase()} · Terbit{' '}
          {formatDualDateTime(signal.published_at)}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-5 pt-5">
        {/* Visual Radar Stepper: HTF vs (MTF) vs LTF */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              {isILA3 ? 'Progres Radar Analisis' : `Progres ${cycleName ?? 'Time Cycle'}`}
            </span>
            <span className="text-[11px] text-muted-foreground">
              {isILA3 ? ila3Prog?.label : `${ila2?.completedSteps ?? 0} / 3 Tahap`}
            </span>
          </div>

          {isILA3 ? (
            /* Layout Khusus ILA 3.0: 3 Steppers (HTF, MTF, LTF) */
            <div className="grid gap-3 sm:grid-cols-3">
              {/* Step 1: HTF Macro Bias */}
              <div
                className={cn(
                  'rounded-xl border p-3.5 transition-all',
                  (ila3Prog?.step ?? 0) >= 1
                    ? 'border-border bg-card'
                    : 'border-border/80 bg-muted/20',
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5 font-semibold text-xs">
                    {(ila3Prog?.step ?? 0) >= 1 ? (
                      <CheckCircle2 className="h-4 w-4 text-foreground" />
                    ) : (
                      <Clock className="h-4 w-4 text-muted-foreground animate-pulse" />
                    )}
                    <span>HTF ({signal.htf || 'HTF'})</span>
                  </div>
                  <Badge
                    variant={(ila3Prog?.step ?? 0) >= 1 ? 'default' : 'outline'}
                    className="text-[10px] font-mono"
                  >
                    {(ila3Prog?.step ?? 0) >= 1 ? 'Valid (Reached)' : 'Monitoring Level'}
                  </Badge>
                </div>
                <p className="mt-2 text-[12px] leading-relaxed text-muted-foreground">
                  {(ila3Prog?.step ?? 0) >= 1
                    ? 'HTF Key Level (ERL/Liquidity) valid disentuh/disapu. Macro bias aktif.'
                    : 'Menunggu harga berinteraksi dengan Key Level HTF atau kondisi Q1 exception.'}
                </p>
                {signal.htf_sweep && signal.htf_sweep.target_dol > 0 && (
                  <div className="mt-2.5 flex items-center justify-between border-t border-border/50 pt-2 text-[11px] font-mono">
                    <span className="text-muted-foreground">Target DOL:</span>
                    <span className="font-semibold tabular-nums">
                      {formatNumber(signal.htf_sweep.target_dol, 5)}
                    </span>
                  </div>
                )}
              </div>

              {/* Step 2: MTF Structure Alignment */}
              <div
                className={cn(
                  'rounded-xl border p-3.5 transition-all',
                  (ila3Prog?.step ?? 0) >= 2
                    ? 'border-border bg-card'
                    : (ila3Prog?.step ?? 0) === 1
                      ? 'border-border/80 bg-muted/20'
                      : 'border-border/50 bg-muted/10 opacity-60',
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5 font-semibold text-xs">
                    {(ila3Prog?.step ?? 0) >= 2 ? (
                      <CheckCircle2 className="h-4 w-4 text-foreground" />
                    ) : (ila3Prog?.step ?? 0) === 1 ? (
                      <Clock className="h-4 w-4 text-muted-foreground animate-pulse" />
                    ) : (
                      <Layers className="h-4 w-4 text-muted-foreground" />
                    )}
                    <span>MTF ({ila3Mtf})</span>
                  </div>
                  <Badge
                    variant={(ila3Prog?.step ?? 0) >= 2 ? 'default' : 'outline'}
                    className="text-[10px] font-mono"
                  >
                    {(ila3Prog?.step ?? 0) >= 2
                      ? 'Valid (Aligned)'
                      : (ila3Prog?.step ?? 0) === 1
                        ? 'Waiting Reversal/Retr'
                        : 'Pending HTF'}
                  </Badge>
                </div>
                <p className="mt-2 text-[12px] leading-relaxed text-muted-foreground">
                  {(ila3Prog?.step ?? 0) >= 2
                    ? 'Struktur internal MTF & SMT terkonfirmasi selaras dengan bias HTF.'
                    : (ila3Prog?.step ?? 0) === 1
                      ? 'HTF selesai. Menunggu konfirmasi MTF reversal (Entry 1) atau FVG retracement (Entry 2).'
                      : 'Fase MTF dimulai setelah harga berinteraksi dengan key level HTF.'}
                </p>
              </div>

              {/* Step 3: LTF Trigger */}
              <div
                className={cn(
                  'rounded-xl border p-3.5 transition-all',
                  (ila3Prog?.step ?? 0) >= 3
                    ? 'border-border bg-card'
                    : (ila3Prog?.step ?? 0) === 2
                      ? 'border-border/80 bg-muted/20'
                      : 'border-border/50 bg-muted/10 opacity-60',
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5 font-semibold text-xs">
                    {(ila3Prog?.step ?? 0) >= 3 ? (
                      <CheckCircle2 className="h-4 w-4 text-foreground" />
                    ) : (ila3Prog?.step ?? 0) === 2 ? (
                      <Clock className="h-4 w-4 text-muted-foreground animate-pulse" />
                    ) : (
                      <Layers className="h-4 w-4 text-muted-foreground" />
                    )}
                    <span>LTF ({signal.ltf || 'LTF'})</span>
                  </div>
                  <Badge
                    variant={(ila3Prog?.step ?? 0) >= 3 ? 'default' : 'outline'}
                    className="text-[10px] font-mono"
                  >
                    {(ila3Prog?.step ?? 0) >= 3
                      ? 'Setup Ready'
                      : (ila3Prog?.step ?? 0) === 2
                        ? 'Waiting CISD Close'
                        : 'Pending MTF'}
                  </Badge>
                </div>
                <p className="mt-2 text-[12px] leading-relaxed text-muted-foreground">
                  {(ila3Prog?.step ?? 0) >= 3
                    ? 'Konfirmasi CISD body-break-body pada LTF valid. Setup siap dieksekusi.'
                    : (ila3Prog?.step ?? 0) === 2
                      ? 'Struktur HTF & MTF valid. Menunggu candle LTF body close menembus CISD origin.'
                      : 'Fase trigger LTF dimulai setelah struktur MTF terkonfirmasi.'}
                </p>
                {signal.cisd && signal.cisd.cisd_level > 0 && (
                  <div className="mt-2.5 flex items-center justify-between border-t border-border/50 pt-2 text-[11px] font-mono">
                    <span className="text-muted-foreground">CISD Level:</span>
                    <span className="font-semibold tabular-nums">
                      {formatNumber(signal.cisd.cisd_level, 5)}
                    </span>
                  </div>
                )}
              </div>
            </div>
          ) : ila2 ? (
            /* Layout Vertical Stepper (Left Alignment) untuk New ILA 2.0 */
            <ol className="relative text-left rounded-xl border border-border/80 bg-secondary/10 p-4">
              {/* Step 1: HTF */}
              <li className="relative pl-7 pb-5">
                <span
                  aria-hidden="true"
                  className={cn(
                    'absolute left-[8.5px] top-5 bottom-0 w-px',
                    ila2.htf.state === 'done' ? 'bg-emerald-500/60' : 'bg-border',
                  )}
                />
                <span className="absolute left-0 top-0.5">
                  <StepMarker state={ila2.htf.state} />
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs font-bold uppercase tracking-wide text-foreground">
                    {ila2.htf.title}
                  </span>
                  <Badge
                    variant={ila2.htf.state === 'done' ? 'default' : 'outline'}
                    className="text-[10px] font-mono"
                  >
                    {ila2.htf.state === 'done'
                      ? 'C2 Valid'
                      : ila2.htf.state === 'invalid'
                        ? 'Invalid'
                        : 'Menunggu'}
                  </Badge>
                </div>
                <p className="mt-1 text-[13px] font-medium text-foreground">
                  {ila2.htf.headline}
                </p>
                {ila2.htf.details.length > 0 && (
                  <ul className="mt-1 space-y-0.5 text-xs text-muted-foreground">
                    {ila2.htf.details.map((item) => (
                      <li key={item}>• {item}</li>
                    ))}
                  </ul>
                )}
              </li>

              {/* Step 2: LTF */}
              <li className="relative pl-7 pb-5">
                <span
                  aria-hidden="true"
                  className={cn(
                    'absolute left-[8.5px] top-5 bottom-0 w-px',
                    ila2.ltf.state === 'done' ? 'bg-emerald-500/60' : 'bg-border',
                  )}
                />
                <span className="absolute left-0 top-0.5">
                  <StepMarker state={ila2.ltf.state} />
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs font-bold uppercase tracking-wide text-foreground">
                    {ila2.ltf.title}
                  </span>
                  <Badge
                    variant={ila2.ltf.state === 'done' ? 'default' : 'outline'}
                    className="text-[10px] font-mono"
                  >
                    {ila2.ltf.state === 'done'
                      ? 'CISD Valid'
                      : ila2.ltf.state === 'active'
                        ? 'Memantau'
                        : ila2.ltf.state === 'invalid'
                          ? 'Invalid'
                          : 'Menunggu HTF'}
                  </Badge>
                </div>
                <ul className="mt-1.5 space-y-1 text-xs">
                  <li className="text-foreground/90">
                    <span className="font-semibold text-muted-foreground">ERL LTF: </span>
                    {ila2.ltf.erlStatus}
                  </li>
                  <li className="text-foreground/90">
                    <span className="font-semibold text-muted-foreground">CISD: </span>
                    {ila2.ltf.cisdStatus}
                  </li>
                  {ila2.ltf.cisdLevel !== undefined && (
                    <li className="font-mono text-[11.5px] text-muted-foreground">
                      Level CISD:{' '}
                      <span className="font-semibold text-foreground tabular-nums">
                        {formatPrice(ila2.ltf.cisdLevel, digits)}
                      </span>
                    </li>
                  )}
                </ul>
              </li>

              {/* Step 3: Entry */}
              <li className="relative pl-7">
                <span className="absolute left-0 top-0.5">
                  <StepMarker state={ila2.entry.state} />
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs font-bold uppercase tracking-wide text-foreground">
                    Entry
                  </span>
                  <Badge
                    variant={ila2.entry.confirmed ? 'default' : 'outline'}
                    className="text-[10px] font-mono"
                  >
                    {ila2.entry.confirmed
                      ? 'Terkonfirmasi'
                      : ila2.entry.state === 'invalid'
                        ? 'Invalid'
                        : 'Belum ada'}
                  </Badge>
                </div>
                <p className="mt-1 text-[13px] font-medium text-foreground">
                  {ila2.entry.headline}
                </p>
                {ila2.entry.confirmed && ila2.entry.entryPrice !== undefined ? (
                  <dl className="mt-2.5 grid grid-cols-2 gap-x-4 gap-y-2.5 rounded-lg border border-border/70 bg-card p-3 sm:grid-cols-4">
                    <div>
                      <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                        Posisi
                      </dt>
                      <dd className={cn('mt-0.5 font-mono text-sm font-bold', dirColor)}>
                        {ila2.entry.direction}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                        Entry
                      </dt>
                      <dd className="mt-0.5 font-mono text-sm font-semibold tabular-nums text-foreground">
                        {formatPrice(ila2.entry.entryPrice, digits)}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                        SL
                      </dt>
                      <dd className="mt-0.5 font-mono text-sm font-semibold tabular-nums text-rose-600 dark:text-rose-400">
                        {ila2.entry.stopLoss !== undefined ? formatPrice(ila2.entry.stopLoss, digits) : '—'}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                        TP{ila2.entry.riskReward ? ` (1:${formatNumber(ila2.entry.riskReward, 0)})` : ''}
                      </dt>
                      <dd className="mt-0.5 font-mono text-sm font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">
                        {ila2.entry.takeProfit !== undefined ? formatPrice(ila2.entry.takeProfit, digits) : '—'}
                      </dd>
                    </div>
                  </dl>
                ) : ila2.entry.note ? (
                  <p className="mt-1 text-xs text-muted-foreground">{ila2.entry.note}</p>
                ) : null}
              </li>
            </ol>
          ) : null}
        </div>

        {/* Parameter Order & Konteks Analisis (khusus ILA 3.0; ILA 2.0 sudah terangkum di vertical stepper) */}
        {isILA3 && (
          signal.entry_price > 0 ? (
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
                Level Entry, Stop Loss, dan Take Profit otomatis aktif seketika
                konfirmasi LTF CISD terpenuhi.
              </span>
            </div>
          )
        )}

        {/* Ringkasan Analisis (khusus ILA 3.0) */}
        {isILA3 && signal.summary && (() => {
          const points = parseAnalysisSummary(signal.summary, signal.strategy_version)
          return (
            <div className="flex flex-col gap-2 rounded-xl border border-border/70 bg-card p-3.5 shadow-xs">
              <div className="flex items-center justify-between border-b border-border/50 pb-2">
                <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
                  <Compass className="h-3.5 w-3.5 text-primary" />
                  Konteks Analisis Setup
                </span>
                <span className="text-[10px] font-medium text-muted-foreground">
                  {signal.strategy_version === 'ila-3.0' ? 'ILA 3.0 Sequence' : 'ILA 2.0 Engine'}
                </span>
              </div>

              {points.length > 0 ? (
                <div className="flex flex-col gap-2 pt-1">
                  {points.map((pt) => {
                    const icon =
                      pt.iconType === 'sweep' ? (
                        <Crosshair className="h-3.5 w-3.5 shrink-0 text-foreground mt-0.5" />
                      ) : pt.iconType === 'trigger' ? (
                        <Zap className="h-3.5 w-3.5 shrink-0 text-foreground mt-0.5" />
                      ) : pt.iconType === 'target' ? (
                        <Target className="h-3.5 w-3.5 shrink-0 text-foreground mt-0.5" />
                      ) : (
                        <Layers className="h-3.5 w-3.5 shrink-0 text-muted-foreground mt-0.5" />
                      )

                    return (
                      <div
                        key={pt.id}
                        className="flex items-start gap-2.5 rounded-lg border border-border/50 bg-secondary/15 p-2.5 transition-colors hover:bg-secondary/25"
                      >
                        {icon}
                        <div className="flex flex-col gap-0.5">
                          <span className="text-[11px] font-semibold text-foreground/90">{pt.title}</span>
                          <span className="text-[12px] leading-relaxed text-muted-foreground">{pt.detail}</span>
                        </div>
                      </div>
                    )
                  })}
                </div>
              ) : (
                <p className="text-[12.5px] leading-relaxed text-foreground/90 pt-1">{signal.summary}</p>
              )}
            </div>
          )
        })()}

        {signal.reject_reason && (
          <div className="space-y-2.5 border-l-2 border-foreground pl-3 py-1">
            <div>
              <span className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                Penahanan Sistem
              </span>
              <p className="text-[13px] text-foreground">{signal.reject_reason}</p>
            </div>
            {signal.can_confirm && <TombolEksekusi signal={signal} />}
          </div>
        )}

        {signal.next_action && (
          <div className="flex items-center gap-2 rounded-xl border border-border bg-secondary/20 px-3.5 py-2.5 text-xs">
            <span className="font-semibold text-muted-foreground">Langkah Berikutnya:</span>
            <span className="font-medium text-foreground">{signal.next_action}</span>
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between border-t border-border pt-3 text-[11px] text-muted-foreground">
          <span>Terbit: {formatDualDateTime(signal.published_at)}</span>
          <span>Batas Waktu: {formatDualDateTime(signal.expires_at)}</span>
        </div>
      </CardContent>
    </Card>
  )
}
