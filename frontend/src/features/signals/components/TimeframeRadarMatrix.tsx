import { useState } from 'react'
import {
  Clock,
  Grid2X2,
  Radio,
  Square,
  TrendingDown,
  TrendingUp,
} from 'lucide-react'
import { Badge, Button, Card, CardContent, CardDescription, CardHeader, StatusPill } from '@/components/ui'
import { formatDualDateTime, formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import { mapSignalsToSlots, semuaTimeframeMati, timeframeSlotDipantau, type TimeframeSlot } from '../lib/signal-slots'
import { SIGNAL_LABEL, SIGNAL_TONE, getCycleBadge, getIla3Mtf, getIla3Progress } from '../lib/status'
import { SignalDetail } from './SignalDetail'
import { TimeframeBadge } from './TimeframeBadge'

export function TimeframeRadarMatrix({
  signals,
  strategy,
  symbol,
  timeframeAktif,
}: {
  signals: Signal[]
  strategy: 'ila-2.0' | 'ila-3.0'
  symbol: string
  /**
   * Pasangan timeframe yang sedang dipantau untuk metode ini.
   * `null` berarti belum diketahui, sehingga slot tidak ditandai mati.
   */
  timeframeAktif?: Set<string> | null
}) {
  // Matriks dikunci ke satu metode. Penyaringan ini pengaman tambahan supaya
  // sinyal dari metode lain tidak pernah ikut membentuk slot di matriks ini.
  const signalMetode = signals.filter((s) => s.strategy_version === strategy)
  const slots = mapSignalsToSlots(signalMetode, strategy)
  const aktif = timeframeAktif ?? null
  const matiSemua = semuaTimeframeMati(aktif)

  // Otomatis arahkan ke slot yang statusnya READY jika ada, atau slot pertama
  const initialKey = slots.find((s) => s.status === 'READY')?.key ?? slots[0]?.key ?? ''
  const [activeKey, setActiveKey] = useState<string>(initialKey)
  const [viewMode, setViewMode] = useState<'tab' | 'grid'>('tab')

  // Pastikan activeKey valid di antara slots saat ini
  const currentSlot = slots.find((s) => s.key === activeKey) ?? slots[0]

  return (
    <div className="space-y-4">
      {/* Header Matrix Controls */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b border-border/60 pb-3">
        {/* Slot Tabs */}
        <div className="flex flex-wrap items-center gap-1.5">
          {slots.map((slot) => {
            const isActive = slot.key === currentSlot?.key && viewMode === 'tab'
            const isReady = slot.status === 'READY'
            const isWaitCisd = slot.status === 'WAIT_CISD'
            const isWaitSweep = slot.status === 'WAIT_SWEEP'

            return (
              <button
                key={slot.key}
                type="button"
                onClick={() => {
                  setActiveKey(slot.key)
                  if (viewMode === 'grid') setViewMode('tab')
                }}
                className={cn(
                  'group flex items-center gap-2 rounded-lg px-2.5 py-1 text-xs font-medium transition-all border',
                  isActive
                    ? 'border-foreground bg-foreground text-background shadow-xs'
                    : 'border-border bg-card text-foreground hover:bg-secondary/60',
                )}
              >
                <span className="font-mono">{slot.label}</span>

                {/* Status Dot / Pill */}
                {isReady ? (
                  <span
                    className={cn(
                      'inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[9.5px] font-bold font-mono',
                      isActive ? 'bg-background text-foreground' : 'bg-foreground text-background',
                    )}
                  >
                    READY
                  </span>
                ) : isWaitCisd ? (
                  <span
                    className={cn(
                      'inline-flex items-center rounded px-1.5 py-0.5 text-[9.5px] font-mono',
                      isActive ? 'bg-background/20 text-background' : 'bg-muted text-muted-foreground',
                    )}
                  >
                    WAIT CISD
                  </span>
                ) : isWaitSweep ? (
                  <span
                    className={cn(
                      'inline-flex items-center rounded px-1.5 py-0.5 text-[9.5px] font-mono',
                      isActive ? 'bg-background/20 text-background' : 'bg-muted text-muted-foreground',
                    )}
                  >
                    WAIT SWEEP
                  </span>
                ) : (
                  <span
                    className={cn(
                      'text-[9.5px] font-mono',
                      isActive ? 'text-background/70' : 'text-muted-foreground',
                    )}
                  >
                    SCAN
                  </span>
                )}
              </button>
            )
          })}
        </div>

        {/* View Switcher: Tab Focus vs Grid 2x2 */}
        <div className="flex items-center gap-1 self-end sm:self-auto shrink-0">
          <Button
            size="sm"
            variant={viewMode === 'tab' ? 'default' : 'outline'}
            className="h-7 px-2.5 text-[11px] gap-1 font-medium"
            onClick={() => setViewMode('tab')}
            title="Tampilan Fokus 1 Timeframe"
          >
            <Square className="h-3 w-3" />
            Fokus
          </Button>

          <Button
            size="sm"
            variant={viewMode === 'grid' ? 'default' : 'outline'}
            className="h-7 px-2.5 text-[11px] gap-1 font-medium"
            onClick={() => setViewMode('grid')}
            title="Tampilan Pantauan 4 Timeframe Sekaligus"
          >
            <Grid2X2 className="h-3 w-3" />
            Grid 4 TF
          </Button>
        </div>
      </div>

      {/* Main Content Area */}
      {matiSemua ? (
        <SemuaTimeframeMatiState symbol={symbol} strategy={strategy} />
      ) : viewMode === 'tab' ? (
        /*
         * Urutan pemeriksaan disengaja: "tidak dipantau" didahulukan atas sinyal
         * terakhir. Slot yang dimatikan masih menyimpan sinyal lama di database,
         * dan menampilkannya lebih dulu membuat radar terbaca sedang memindai
         * padahal engine sudah berhenti mengevaluasinya.
         */
        currentSlot && !timeframeSlotDipantau(aktif, currentSlot) ? (
          <SlotMatiState slot={currentSlot} symbol={symbol} />
        ) : currentSlot?.latestSignal ? (
          <SignalDetail signal={currentSlot.latestSignal} />
        ) : (
          <SlotEmptyState slot={currentSlot} symbol={symbol} />
        )
      ) : (
        /* Grid 2x2 CCTV Mode */
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {slots.map((slot) => (
            <SlotGridCard
              key={slot.key}
              slot={slot}
              symbol={symbol}
              dipantau={timeframeSlotDipantau(aktif, slot)}
              onSelect={() => {
                setActiveKey(slot.key)
                setViewMode('tab')
              }}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function SemuaTimeframeMatiState({
  symbol,
  strategy,
}: {
  symbol: string
  strategy: 'ila-2.0' | 'ila-3.0'
}) {
  // Kondisi ini bukan "menunggu setup": tidak ada satu pun pasangan timeframe
  // yang dipantau, jadi engine memang tidak memindai apa pun. Dibedakan supaya
  // user tidak menunggu sinyal yang tidak akan pernah datang.
  return (
    <Card className="border-dashed border-border bg-card/60">
      <CardContent className="flex flex-col items-center justify-center gap-2 py-12 text-center">
        <Radio className="h-8 w-8 text-muted-foreground/60" />
        <h4 className="text-sm font-semibold text-foreground">
          Timeframe {strategy} belum dipantau untuk {symbol}
        </h4>
        <p className="max-w-md text-xs leading-relaxed text-muted-foreground">
          Seluruh pasangan timeframe metode ini sedang dimatikan, sehingga tidak ada
          yang dievaluasi. Aktifkan minimal satu pasangan timeframe agar pemantauan
          dimulai kembali.
        </p>
        <Button size="sm" variant="outline" className="mt-1" onClick={gulirKePengaturan}>
          Aktifkan pemantauan
        </Button>
      </CardContent>
    </Card>
  )
}

/**
 * gulirKePengaturan membawa pengguna ke saklar timeframe di halaman yang sama.
 *
 * Panel pengaturan ada dua: satu untuk layar lebar, satu untuk mobile, dan
 * keduanya membawa id yang sama agar satu tombol berlaku di semua ukuran layar.
 * Karena yang tersembunyi tetap ada di DOM, elemen yang benar-benar tampil
 * dipilih lebih dulu — kalau tidak, gulir berhenti di elemen tak terlihat dan
 * pengguna merasa tombolnya tidak bekerja.
 */
function gulirKePengaturan() {
  const semua = document.querySelectorAll<HTMLElement>('#pengaturan-timeframe')
  const terlihat = Array.from(semua).find((el) => el.offsetParent !== null) ?? semua[0]
  terlihat?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function SlotMatiState({ slot }: { slot?: TimeframeSlot; symbol: string }) {
  // Slot ini sengaja tidak dipantau. Dibedakan dari "sedang memindai" supaya
  // user tidak menunggu setup yang tidak akan pernah muncul.
  return (
    <Card className="border-dashed border-border bg-card/60">
      <CardContent className="flex flex-col items-center justify-center gap-2 py-12 text-center">
        <Radio className="h-8 w-8 text-muted-foreground/60" />
        <h4 className="text-sm font-semibold text-foreground">
          {slot?.label || 'Timeframe'} tidak dipantau
        </h4>
        <p className="max-w-md text-xs leading-relaxed text-muted-foreground">
          Pasangan timeframe ini dimatikan pada pengaturan timeframe akun, jadi engine
          tidak mengevaluasinya. Data terakhir yang tampil bisa berasal dari sebelum
          pemantauan dimatikan.
        </p>
        <Button size="sm" variant="outline" className="mt-1" onClick={gulirKePengaturan}>
          Aktifkan pemantauan
        </Button>
      </CardContent>
    </Card>
  )
}

function SlotEmptyState({ slot, symbol }: { slot?: TimeframeSlot; symbol: string }) {
  return (
    <Card className="border-dashed border-border bg-card/60">
      <CardContent className="py-12 text-center flex flex-col items-center justify-center gap-2">
        <Clock className="h-8 w-8 text-muted-foreground/60 animate-pulse" />
        <h4 className="text-sm font-semibold text-foreground">
          Radar Memantau {symbol} · {slot?.label || 'Timeframe'}
        </h4>
        <p className="max-w-md text-xs text-muted-foreground leading-relaxed">
          Belum ada setup struktur yang terbentuk pada timeframe ini. Engine terus
          memindai pembentukan liquidity sweep & candle close secara otomatis.
        </p>
      </CardContent>
    </Card>
  )
}

function SlotGridCard({
  slot,
  symbol,
  dipantau,
  onSelect,
}: {
  slot: TimeframeSlot
  symbol: string
  /** false berarti slot ini sengaja tidak dipindai engine. */
  dipantau: boolean
  onSelect: () => void
}) {
  const signal = slot.latestSignal

  // Slot yang dimatikan diperlakukan sama seperti di tampilan fokus: status
  // "tidak dipantau" didahulukan atas sinyal lama yang masih tersimpan.
  if (!dipantau) {
    return (
      <Card className="flex flex-col justify-between border-dashed border-border/80 bg-secondary/10 p-4">
        <div className="flex items-center justify-between border-b border-border/50 pb-2.5">
          <div className="flex items-center gap-1.5 font-mono text-xs font-semibold">
            <span>{slot.label}</span>
            {slot.cycle && (
              <Badge variant="outline" className="text-[9.5px]">
                {slot.cycle}
              </Badge>
            )}
          </div>
          <span className="font-mono text-[10px] uppercase text-muted-foreground">Tidak dipantau</span>
        </div>
        <div className="flex flex-col items-center gap-2 py-6 text-center">
          <p className="text-xs text-muted-foreground">
            Pasangan {slot.label} dimatikan pada pengaturan timeframe akun.
          </p>
          <Button size="sm" variant="outline" onClick={onSelect}>
            Lihat detail
          </Button>
        </div>
        <div className="border-t border-border/40 pt-2 text-right text-[10.5px] text-muted-foreground">
          {symbol} · Tidak dipantau
        </div>
      </Card>
    )
  }

  if (!signal) {
    return (
      <Card className="border-dashed border-border/80 bg-secondary/10 flex flex-col justify-between p-4">
        <div className="flex items-center justify-between border-b border-border/50 pb-2.5">
          <div className="flex items-center gap-1.5 font-mono text-xs font-semibold">
            <span>{slot.label}</span>
            {slot.cycle && (
              <Badge variant="outline" className="text-[9.5px]">
                {slot.cycle}
              </Badge>
            )}
          </div>
          <span className="text-[10px] font-mono text-muted-foreground uppercase">Scanning</span>
        </div>
        <div className="py-8 text-center text-xs text-muted-foreground">
          Menunggu pembentukan setup pada {slot.label}
        </div>
        <div className="text-[10.5px] text-muted-foreground text-right border-t border-border/40 pt-2">
          {symbol} · Scanning
        </div>
      </Card>
    )
  }

  const isBuy = signal.direction === 'BUY'
  const isSell = signal.direction === 'SELL'
  const isILA3 = signal.strategy_version === 'ila-3.0'
  const mtf = isILA3 ? getIla3Mtf(signal.htf, signal.ltf) : undefined
  const cycle = isILA3 ? getCycleBadge(signal.strategy_version, signal.htf, signal.ltf) : null
  const ila3Prog = isILA3 ? getIla3Progress(signal.status) : null

  const isReady = signal.status === 'READY_TO_ENTRY' || signal.status === 'OPPORTUNITY'
  const isWaitLtf = signal.status === 'WAIT_LTF_CISD'
  const isHtfSweepDone = isWaitLtf || isReady

  return (
    <Card
      className={cn(
        'group flex flex-col justify-between border transition-all hover:border-foreground/50 hover:shadow-sm cursor-pointer',
        isReady
          ? 'border-emerald-500/60 bg-emerald-500/[0.02]'
          : 'border-border bg-card',
      )}
      onClick={onSelect}
    >
      <CardHeader className="border-b border-border/60 bg-secondary/15 pb-3">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span
              className={cn(
                'inline-flex items-center gap-1 font-mono text-xs font-bold',
                isBuy
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : isSell
                    ? 'text-rose-600 dark:text-rose-400'
                    : 'text-muted-foreground',
              )}
            >
              {isBuy ? <TrendingUp className="h-3.5 w-3.5" /> : isSell ? <TrendingDown className="h-3.5 w-3.5" /> : null}
              {signal.direction}
            </span>
            <TimeframeBadge htf={signal.htf} mtf={mtf} ltf={signal.ltf} />
            <Badge variant="outline" className="font-mono text-[9.5px]">
              {signal.strategy_version}
            </Badge>
          </div>
          <StatusPill tone={SIGNAL_TONE[signal.status]} className="text-[10px]">
            {SIGNAL_LABEL[signal.status]}
          </StatusPill>
        </div>
        <CardDescription className="flex items-center gap-1 text-[10.5px] text-muted-foreground pt-1">
          <Radio className="h-3 w-3 text-emerald-500 animate-pulse" />
          {cycle ? `${cycle} · ` : ''}Terbit {formatDualDateTime(signal.published_at)}
        </CardDescription>
      </CardHeader>

      <CardContent className="p-3.5 space-y-3">
        {/* Stepper Ringkas Grid */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between text-[10px] text-muted-foreground font-semibold uppercase">
            <span>Progress Radar</span>
            <span>
              {isILA3
                ? ila3Prog?.label
                : isReady
                  ? '2/2 Ready'
                  : isWaitLtf
                    ? '1/2 Wait CISD'
                    : '0/2 Sweep'}
            </span>
          </div>

          {isILA3 ? (
            <div className="grid grid-cols-3 gap-1.5 text-[10.5px]">
              <div
                className={cn(
                  'rounded border px-2 py-1 text-center font-mono',
                  (ila3Prog?.step ?? 0) >= 1
                    ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold'
                    : 'border-border bg-secondary/40 text-muted-foreground',
                )}
              >
                HTF: {signal.htf}
              </div>
              <div
                className={cn(
                  'rounded border px-2 py-1 text-center font-mono',
                  (ila3Prog?.step ?? 0) >= 2
                    ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold'
                    : (ila3Prog?.step ?? 0) === 1
                      ? 'border-amber-500/50 bg-amber-500/10 text-amber-600 font-semibold'
                      : 'border-border bg-secondary/40 text-muted-foreground',
                )}
              >
                MTF: {mtf}
              </div>
              <div
                className={cn(
                  'rounded border px-2 py-1 text-center font-mono',
                  (ila3Prog?.step ?? 0) >= 3
                    ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold'
                    : (ila3Prog?.step ?? 0) === 2
                      ? 'border-amber-500/50 bg-amber-500/10 text-amber-600 font-semibold'
                      : 'border-border bg-secondary/40 text-muted-foreground',
                )}
              >
                LTF: {signal.ltf}
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-2 text-[10.5px]">
              <div
                className={cn(
                  'rounded border px-2.5 py-1 text-center font-mono',
                  isHtfSweepDone
                    ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold'
                    : 'border-border bg-secondary/40 text-muted-foreground',
                )}
              >
                HTF: {signal.htf}
              </div>
              <div
                className={cn(
                  'rounded border px-2.5 py-1 text-center font-mono',
                  isReady
                    ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 font-semibold'
                    : isWaitLtf
                      ? 'border-amber-500/50 bg-amber-500/10 text-amber-600 font-semibold'
                      : 'border-border bg-secondary/40 text-muted-foreground',
                )}
              >
                LTF: {signal.ltf}
              </div>
            </div>
          )}
        </div>

        {/* Parameter Order jika Siap */}
        {signal.entry_price > 0 && (
          <div className="grid grid-cols-3 gap-1.5 rounded-lg border border-border/70 bg-secondary/20 p-2 text-[10.5px] font-mono">
            <div>
              <span className="text-[9.5px] uppercase text-muted-foreground">Entry</span>
              <p className="font-semibold tabular-nums">{formatNumber(signal.entry_price, 5)}</p>
            </div>
            <div>
              <span className="text-[9.5px] uppercase text-muted-foreground">SL</span>
              <p className="font-semibold tabular-nums text-rose-600">{formatNumber(signal.stop_loss, 4)}</p>
            </div>
            <div className="text-right">
              <span className="text-[9.5px] uppercase text-muted-foreground">TP</span>
              <p className="font-semibold tabular-nums text-emerald-600">{formatNumber(signal.take_profit, 4)}</p>
            </div>
          </div>
        )}

        <div className="flex items-center justify-between text-[11px] text-muted-foreground border-t border-border/50 pt-2">
          <span className="truncate max-w-[200px]">
            {signal.summary ? signal.summary.slice(0, 40) + '…' : 'Konteks setup pasar'}
          </span>
          <span className="text-primary font-medium group-hover:underline">Buka Detail →</span>
        </div>
      </CardContent>
    </Card>
  )
}
