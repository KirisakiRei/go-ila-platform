import { useState } from 'react'
import { Clock, Radio } from 'lucide-react'
import { Button, Card, CardContent } from '@/components/ui'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import {
  mapSignalsToSlots,
  pilihSlotAwal,
  semuaTimeframeMati,
  timeframeSlotDipantau,
  type TimeframeSlot,
} from '../lib/signal-slots'
import { SignalDetail } from './SignalDetail'

export function TimeframeRadarMatrix({
  signals,
  strategy,
  symbol,
  timeframeAktif,
  activeSlotKey,
  onSelectSlot,
  hideSlotTabs = false,
}: {
  signals: Signal[]
  strategy: 'ila-2.0' | 'ila-3.0'
  symbol: string
  /**
   * Pasangan timeframe yang sedang dipantau untuk metode ini.
   * `null` berarti belum diketahui, sehingga slot tidak ditandai mati.
   */
  timeframeAktif?: Set<string> | null
  /** Kunci slot aktif bila disinkronkan dengan komponen luar (mis. Chart Setup). */
  activeSlotKey?: string | null
  onSelectSlot?: (key: string) => void
  /** Sembunyikan deretan pil slot internal bila pemilih sudah ditampilkan di luar. */
  hideSlotTabs?: boolean
}) {
  // Matriks dikunci ke satu metode. Penyaringan ini pengaman tambahan supaya
  // sinyal dari metode lain tidak pernah ikut membentuk slot di matriks ini.
  const signalMetode = signals.filter((s) => s.strategy_version === strategy)
  const slots = mapSignalsToSlots(signalMetode, strategy)
  const aktif = timeframeAktif ?? null
  const matiSemua = semuaTimeframeMati(aktif)

  // Otomatis arahkan ke slot dipantau yang paling relevan (READY → WAIT_CISD → WAIT_SWEEP → aktif pertama)
  const initialKey = pilihSlotAwal(slots, aktif)
  const [internalKey, setInternalKey] = useState<string>(initialKey)

  const pickSlot = (key: string) => {
    setInternalKey(key)
    onSelectSlot?.(key)
  }

  const targetKey = activeSlotKey || internalKey
  const currentSlot =
    slots.find((s) => s.key === targetKey) ??
    slots.find((s) => s.key === initialKey) ??
    slots[0]

  return (
    <div className="space-y-4">
      {!hideSlotTabs && (
        <div className="flex flex-wrap items-center gap-1.5 border-b border-border/60 pb-3">
          {slots.map((slot) => {
            const dipantau = timeframeSlotDipantau(aktif, slot)
            const isActive = slot.key === currentSlot?.key
            const isReady = dipantau && slot.status === 'READY'
            const isWaitCisd = dipantau && slot.status === 'WAIT_CISD'
            const isWaitSweep = dipantau && slot.status === 'WAIT_SWEEP'

            return (
              <button
                key={slot.key}
                type="button"
                onClick={() => pickSlot(slot.key)}
                className={cn(
                  'group flex items-center gap-2 rounded-lg px-2.5 py-1 text-xs font-medium transition-all border',
                  isActive
                    ? 'border-foreground bg-foreground text-background shadow-xs'
                    : 'border-border bg-card text-foreground hover:bg-secondary/60',
                )}
              >
                {strategy === 'ila-2.0' && slot.cycle ? (
                  <span className="inline-flex items-baseline gap-1.5">
                    <span className="font-semibold">{slot.cycle}</span>
                    <span
                      className={cn(
                        'font-mono text-[10px]',
                        isActive ? 'text-background/75' : 'text-muted-foreground',
                      )}
                    >
                      {slot.label}
                    </span>
                  </span>
                ) : (
                  <span className="font-mono">{slot.label}</span>
                )}

                {!dipantau ? (
                  <span
                    className={cn(
                      'text-[9.5px] font-mono',
                      isActive ? 'text-background/70' : 'text-muted-foreground',
                    )}
                  >
                    OFF
                  </span>
                ) : isReady ? (
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
                    {strategy === 'ila-2.0' ? 'WAIT C2' : 'WAIT SWEEP'}
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
      )}

      {matiSemua ? (
        <SemuaTimeframeMatiState symbol={symbol} strategy={strategy} />
      ) : /*
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
 * gulirKePengaturan membuka modal pengaturan pair (atau menggulir ke
 * #pengaturan-timeframe bila dipakai di halaman lain).
 */
function gulirKePengaturan() {
  window.dispatchEvent(new CustomEvent('ila:open-timeframe-settings'))
  window.requestAnimationFrame(() => {
    const semua = document.querySelectorAll<HTMLElement>('#pengaturan-timeframe')
    const terlihat = Array.from(semua).find((el) => el.offsetParent !== null) ?? semua[0]
    terlihat?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  })
}

function SlotMatiState({ slot }: { slot?: TimeframeSlot; symbol: string }) {
  return (
    <Card className="border-dashed border-border bg-card/60">
      <CardContent className="flex flex-col items-center justify-center gap-2 py-12 text-center">
        <Radio className="h-8 w-8 text-muted-foreground/60" />
        <h4 className="text-sm font-semibold text-foreground">
          {slot?.cycle ? `${slot.cycle} (${slot.label})` : slot?.label || 'Timeframe'} tidak dipantau
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
          Radar Memantau {symbol} ·{' '}
          {slot?.cycle ? `${slot.cycle} (${slot.label})` : slot?.label || 'Timeframe'}
        </h4>
        <p className="max-w-md text-xs text-muted-foreground leading-relaxed">
          Belum ada setup struktur yang terbentuk pada cycle ini. Engine terus
          memindai C2 closure pada key level HTF &amp; konfirmasi CISD secara otomatis.
        </p>
      </CardContent>
    </Card>
  )
}
