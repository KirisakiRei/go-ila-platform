import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  StatusPill,
} from '@/components/ui'
import { formatDateTime, formatNumber } from '@/lib/format'
import type { Signal } from '@/types/api'
import { SIGNAL_LABEL, SIGNAL_TONE } from '../lib/status'
import { TimeframeBadge } from './TimeframeBadge'
import { TombolEksekusi } from './TombolEksekusi'

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="tabular text-[13px] font-medium">{value}</dd>
    </div>
  )
}

/**
 * SignalDetail menampilkan satu sinyal secara lengkap.
 * Sengaja tanpa tombol eksekusi: eksekusi hanya lewat tautan konfirmasi yang
 * dikirim bot, dan halaman itu wajib sesi login.
 */
export function SignalDetail({ signal }: { signal: Signal }) {
  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle>{signal.symbol}</CardTitle>
          <span className="text-[13px] text-muted-foreground">{signal.direction}</span>
          <StatusPill tone={SIGNAL_TONE[signal.status]} className="ml-auto">
            {SIGNAL_LABEL[signal.status]}
          </StatusPill>
        </div>
        <CardDescription>
          {signal.strategy_version}
          <TimeframeBadge htf={signal.htf} ltf={signal.ltf} className="ml-1.5" />
          {' · '}engine {signal.engine_version || 'tidak diketahui'}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-4">
        <dl className="grid grid-cols-2 gap-x-4 gap-y-3">
          <Field label="Entry" value={formatNumber(signal.entry_price, 5)} />
          <Field label="Rasio" value={`1 : ${formatNumber(signal.risk_reward, 2)}`} />
          <Field label="Stop loss" value={formatNumber(signal.stop_loss, 5)} />
          <Field label="Take profit" value={formatNumber(signal.take_profit, 5)} />
        </dl>

        {signal.summary && (
          <div className="flex flex-col gap-1 border-t border-border pt-3">
            <span className="text-[11px] uppercase tracking-wide text-muted-foreground">
              Alasan
            </span>
            <p className="text-[13px] leading-relaxed">{signal.summary}</p>
          </div>
        )}

        {signal.reject_reason && (
          <div className="border-l-2 border-foreground pl-2.5">
            <span className="text-[11px] uppercase tracking-wide text-muted-foreground">
              Ditahan
            </span>
            <p className="text-[13px]">{signal.reject_reason}</p>
          </div>
        )}

        {signal.next_action && (
          <div className="flex flex-col gap-1">
            <span className="text-[11px] uppercase tracking-wide text-muted-foreground">
              Langkah berikutnya
            </span>
            <p className="text-[13px] font-medium">{signal.next_action}</p>
          </div>
        )}

        <div className="flex flex-col gap-1 border-t border-border pt-3 text-[11px] text-muted-foreground">
          <span>Terbit {formatDateTime(signal.published_at)}</span>
          <span>Kedaluwarsa {formatDateTime(signal.expires_at)}</span>
        </div>

        {/* Keputusan hanya ditawarkan selama sinyal masih bisa ditindaklanjuti,
            dan hanya untuk status yang memang menunggu keputusan user. */}
        {(signal.status === 'READY_TO_ENTRY' || signal.status === 'OPPORTUNITY') && (
          <div className="border-t border-border pt-3">
            <TombolEksekusi signal={signal} />
          </div>
        )}
      </CardContent>
    </Card>
  )
}
