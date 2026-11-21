import { useState } from 'react'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui'
import { cn } from '@/lib/utils'

type StepId = 'cycle' | 'htf' | 'ltf' | 'entry'
type HtfKind = 'irl' | 'erl'
type CisdKind = 'k1' | 'k2' | 'k3'

const STEPS: Array<{ id: StepId; num: number; label: string }> = [
  { id: 'cycle', num: 1, label: 'Time Cycle' },
  { id: 'htf', num: 2, label: 'HTF (C2)' },
  { id: 'ltf', num: 3, label: 'LTF (CISD)' },
  { id: 'entry', num: 4, label: 'Entry' },
]

const CYCLES = [
  { name: 'Monthly Cycle', htf: 'W', ltf: 'H4', note: 'Swing jangka menengah' },
  { name: 'Weekly Cycle', htf: 'D1', ltf: 'H1', note: 'Pergerakan mingguan' },
  { name: 'Daily Cycle', htf: 'H4', ltf: 'M15', note: 'Intraday harian (NY)' },
  { name: '90M Cycle', htf: 'H1', ltf: 'M5', note: 'Sesi 6 jam (NY)' },
  { name: 'Micro Cycle', htf: 'M15', ltf: 'M1', note: 'Sesi 90 menit (NY)' },
] as const

type CandleProps = {
  x: number
  w?: number
  open: number
  close: number
  high: number
  low: number
  muted?: boolean
}

/**
 * Satu candlestick gaya minimalis (mengikuti gaya referensi New ILA 2.0):
 * - Bearish (close > open pada koordinat Y SVG): isi gelap solid
 * - Bullish (close < open pada koordinat Y SVG): isi terang dengan border gelap
 */
function Candle({ x, w = 22, open, close, high, low, muted = false }: CandleProps) {
  const cx = x + w / 2
  const bearish = close > open
  const bodyTop = Math.min(open, close)
  const bodyH = Math.max(Math.abs(close - open), 3)
  return (
    <g opacity={muted ? 0.4 : 1}>
      <line
        x1={cx}
        y1={high}
        x2={cx}
        y2={low}
        className="stroke-foreground"
        strokeWidth="1.25"
      />
      <rect
        x={x}
        y={bodyTop}
        width={w}
        height={bodyH}
        rx="1"
        className={bearish ? 'fill-foreground stroke-foreground' : 'fill-card stroke-foreground'}
        strokeWidth="1.25"
      />
    </g>
  )
}

function CycleDiagramSvg() {
  return (
    <svg
      viewBox="0 0 600 185"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label="Ilustrasi Time Cycle: C2 valid dan C3 wajib berada di dalam satu cycle yang sama"
    >
      <rect
        x="24"
        y="28"
        width="368"
        height="132"
        rx="8"
        className="fill-secondary/30 stroke-foreground/40"
        strokeDasharray="5 4"
        strokeWidth="1.2"
      />
      <text x="36" y="47" className="fill-foreground text-[11px] font-bold">
        1 Time Cycle Aktif (mis. Daily Cycle · H4)
      </text>

      <rect
        x="408"
        y="28"
        width="168"
        height="132"
        rx="8"
        className="fill-muted/30 stroke-border"
        strokeDasharray="4 4"
        strokeWidth="1"
      />
      <text x="420" y="47" className="fill-muted-foreground text-[10.5px] font-semibold">
        Cycle Berikutnya
      </text>

      <Candle x={54} open={106} close={82} high={74} low={116} muted />
      <Candle x={108} open={84} close={120} high={78} low={130} muted />

      <rect
        x="154"
        y="58"
        width="44"
        height="88"
        rx="5"
        fill="rgba(244,63,94,0.09)"
        stroke="rgba(244,63,94,0.45)"
        strokeWidth="1"
      />
      <Candle x={165} open={114} close={126} high={66} low={136} />
      <text x="176" y="154" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[10px] font-bold">
        C2 Valid
      </text>

      <rect
        x="218"
        y="58"
        width="154"
        height="88"
        rx="5"
        fill="rgba(16,185,129,0.09)"
        stroke="rgba(16,185,129,0.55)"
        strokeWidth="1.2"
      />
      <Candle x={232} open={126} close={138} high={116} low={142} />
      <text x="270" y="88" className="fill-emerald-700 dark:fill-emerald-400 text-[10.5px] font-bold">
        C3 Berjalan
      </text>
      <text x="270" y="103" className="fill-muted-foreground text-[9.5px]">
        Jendela cari CISD
      </text>
      <text x="270" y="116" className="fill-muted-foreground text-[9.5px]">
        di LTF (mis. M15)
      </text>

      <Candle x={436} open={124} close={108} high={96} low={132} muted />
      <text x="494" y="100" textAnchor="middle" className="fill-muted-foreground text-[10px] font-semibold">
        Di luar cycle
      </text>
      <text x="494" y="114" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[9.5px] font-semibold">
        Setup tidak berlaku
      </text>
    </svg>
  )
}

function HtfIrlSvg() {
  return (
    <svg
      viewBox="0 0 600 225"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label="Ilustrasi HTF C2 Closure pada IRL SIBI"
    >
      {/* Zona SIBI (FVG Bearish): antara Low C1 (y=72) dan High C3 (y=138) */}
      <rect
        x="84"
        y="72"
        width="318"
        height="66"
        fill="rgba(244,63,94,0.14)"
        stroke="rgba(244,63,94,0.45)"
        strokeWidth="1"
      />
      <text x="394" y="86" textAnchor="end" className="fill-rose-600 dark:fill-rose-400 text-[10px] font-bold">
        Zona SIBI (IRL)
      </text>

      {/* 3 Candle pembentuk SIBI */}
      <Candle x={62} open={44} close={62} high={26} low={72} />
      <text x="73" y="21" textAnchor="middle" className="fill-muted-foreground text-[9px] font-mono">
        1
      </text>
      <Candle x={102} open={62} close={156} high={52} low={168} />
      <text x="113" y="45" textAnchor="middle" className="fill-muted-foreground text-[9px] font-mono">
        2
      </text>
      <Candle x={142} open={156} close={174} high={138} low={186} />
      <text x="153" y="198" textAnchor="middle" className="fill-muted-foreground text-[9px] font-mono">
        3
      </text>

      {/* Candle naik masuk ke dalam zona SIBI (menggeser acuan) */}
      <Candle x={190} open={174} close={112} high={96} low={182} />
      <text x="201" y="198" textAnchor="middle" className="fill-muted-foreground text-[9px]">
        Masuk zona
      </text>

      {/* Garis Acuan dari High candle yang masuk zona (y=96) */}
      <line
        x1="201"
        y1="96"
        x2="386"
        y2="96"
        className="stroke-foreground"
        strokeDasharray="4 3"
        strokeWidth="1.2"
      />
      <text x="386" y="110" textAnchor="end" className="fill-foreground text-[9.5px] font-semibold">
        Acuan C2
      </text>

      {/* Candle C2 Valid: wick melewati acuan (y=68) & body close di bawah acuan */}
      <Candle x={244} open={112} close={128} high={68} low={146} />
      <circle cx="255" cy="68" r="3" className="fill-rose-600" />
      <text x="255" y="160" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[10.5px] font-bold">
        C2 Valid
      </text>

      <line x1="258" y1="66" x2="308" y2="38" className="stroke-rose-500" strokeWidth="1" />
      <text x="313" y="36" className="fill-foreground text-[10px] font-semibold">
        Wick sentuh / lewati acuan di SIBI
      </text>
      <text x="313" y="49" className="fill-muted-foreground text-[9.5px]">
        Body close tetap di bawah batas atas zona
      </text>

      {/* Jendela C3 */}
      <rect
        x="414"
        y="64"
        width="164"
        height="138"
        rx="7"
        fill="rgba(16,185,129,0.08)"
        stroke="rgba(16,185,129,0.55)"
        strokeDasharray="4 3"
        strokeWidth="1.2"
      />
      <Candle x={436} open={128} close={168} high={116} low={180} />
      <text x="472" y="102" className="fill-emerald-700 dark:fill-emerald-400 text-[11px] font-bold">
        C3 Berjalan
      </text>
      <text x="472" y="118" className="fill-muted-foreground text-[9.5px]">
        Cari -CISD di LTF
      </text>
      <text x="472" y="132" className="fill-muted-foreground text-[9.5px]">
        sebelum C3 close.
      </text>
      <text x="472" y="152" className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        C3 close tanpa setup
      </text>
      <text x="472" y="164" className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        = Invalid
      </text>
    </svg>
  )
}

function HtfErlSvg() {
  return (
    <svg
      viewBox="0 0 600 225"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label="Ilustrasi HTF C2 Closure pada ERL Swing High"
    >
      {/* 3-candle Swing High */}
      <Candle x={54} open={142} close={112} high={96} low={152} muted />
      <Candle x={94} open={112} close={90} high={68} low={120} />
      <Candle x={134} open={92} close={126} high={84} low={136} muted />
      <text x="105" y="56" textAnchor="middle" className="fill-foreground text-[10px] font-bold">
        Swing High (ERL)
      </text>

      {/* Garis ERL Swing High */}
      <line
        x1="105"
        y1="68"
        x2="386"
        y2="68"
        className="stroke-rose-600 dark:stroke-rose-400"
        strokeDasharray="5 3"
        strokeWidth="1.35"
      />
      <text x="386" y="61" textAnchor="end" className="fill-rose-600 dark:fill-rose-400 text-[9.5px] font-bold">
        Level ERL HTF
      </text>

      <Candle x={182} open={126} close={146} high={118} low={158} muted />
      <Candle x={222} open={144} close={104} high={94} low={150} muted />

      {/* C2 Valid: wick sweep di atas ERL (y=46), body close di bawah ERL */}
      <Candle x={268} open={102} close={118} high={46} low={130} />
      <circle cx="279" cy="46" r="3" className="fill-rose-600" />
      <text x="279" y="148" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[10.5px] font-bold">
        C2 Valid
      </text>

      <line x1="282" y1="45" x2="324" y2="28" className="stroke-rose-500" strokeWidth="1" />
      <text x="330" y="28" className="fill-foreground text-[10px] font-semibold">
        Wick sapu di atas Swing High
      </text>
      <text x="330" y="41" className="fill-muted-foreground text-[9.5px]">
        Body C2 tetap close di bawah level ERL
      </text>

      {/* Jendela C3 */}
      <rect
        x="414"
        y="64"
        width="164"
        height="138"
        rx="7"
        fill="rgba(16,185,129,0.08)"
        stroke="rgba(16,185,129,0.55)"
        strokeDasharray="4 3"
        strokeWidth="1.2"
      />
      <Candle x={436} open={118} close={164} high={108} low={176} />
      <text x="472" y="102" className="fill-emerald-700 dark:fill-emerald-400 text-[11px] font-bold">
        C3 Berjalan
      </text>
      <text x="472" y="118" className="fill-muted-foreground text-[9.5px]">
        Cari -CISD di LTF
      </text>
      <text x="472" y="132" className="fill-muted-foreground text-[9.5px]">
        selama C3 aktif.
      </text>
      <text x="472" y="152" className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        Body close tembus ERL
      </text>
      <text x="472" y="164" className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        = FTM (bukan C2)
      </text>
    </svg>
  )
}

function LtfCisdSvg({ kind }: { kind: CisdKind }) {
  return (
    <svg
      viewBox="0 0 600 235"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label={`Ilustrasi LTF -CISD ${kind === 'k1' ? 'Kondisi 1' : kind === 'k2' ? 'Kondisi 2' : 'Kondisi 3'}`}
    >
      <text x="20" y="22" className="fill-muted-foreground text-[10px] font-semibold">
        Di dalam C3 HTF · Contoh Arah Bearish (-CISD)
      </text>

      {/* ERL Swing High LTF */}
      <Candle x={36} open={134} close={112} high={100} low={144} muted />
      <Candle x={68} open={112} close={96} high={74} low={120} />
      <Candle x={100} open={98} close={126} high={90} low={136} muted />
      <text x="79" y="64" textAnchor="middle" className="fill-foreground text-[9.5px] font-bold">
        ERL LTF
      </text>

      <line
        x1="79"
        y1="74"
        x2="372"
        y2="74"
        className="stroke-rose-600 dark:stroke-rose-400"
        strokeDasharray="4 3"
        strokeWidth="1.2"
      />
      <text x="172" y="67" className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        Swing High LTF
      </text>

      {kind === 'k3' && (
        <>
          <Candle x={142} open={126} close={154} high={120} low={168} />
          <line
            x1="153"
            y1="168"
            x2="452"
            y2="168"
            className="stroke-amber-600 dark:stroke-amber-400"
            strokeDasharray="4 3"
            strokeWidth="1.2"
          />
          <text x="153" y="182" textAnchor="middle" className="fill-amber-700 dark:fill-amber-400 text-[9px] font-bold">
            SWL
          </text>
        </>
      )}

      {kind === 'k1' && (
        <>
          <Candle x={200} open={146} close={116} high={110} low={154} />
          <Candle x={238} open={116} close={92} high={84} low={122} />
          <Candle x={276} open={92} close={80} high={56} low={98} />
          <circle cx="287" cy="56" r="3" className="fill-rose-600" />
          <text x="287" y="45" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[9.5px] font-bold">
            Wick kena ERL (Valid)
          </text>

          <line x1="200" y1="146" x2="495" y2="146" className="stroke-foreground" strokeDasharray="5 3" strokeWidth="1.35" />
          <text x="495" y="139" textAnchor="end" className="fill-foreground text-[9.5px] font-bold">
            Level CISD (Open Bullish #1)
          </text>

          <Candle x={324} open={82} close={126} high={78} low={132} />
          <Candle x={364} open={126} close={168} high={120} low={178} />
          <circle cx="375" cy="168" r="3.5" className="fill-emerald-600" />
          <text x="444" y="172" className="fill-emerald-700 dark:fill-emerald-400 text-[10px] font-bold">
            Close tembus CISD
          </text>
          <text x="444" y="185" className="fill-muted-foreground text-[9.5px]">
            -CISD Kondisi 1 Valid
          </text>
        </>
      )}

      {kind === 'k2' && (
        <>
          <Candle x={198} open={156} close={134} high={126} low={164} />
          <Candle x={234} open={134} close={88} high={82} low={140} />
          <Candle x={270} open={88} close={58} high={46} low={98} />

          <rect
            x="210"
            y="98"
            width="210"
            height="28"
            fill="rgba(16,185,129,0.14)"
            stroke="rgba(16,185,129,0.5)"
            strokeWidth="1"
          />
          <text x="414" y="115" textAnchor="end" className="fill-emerald-700 dark:fill-emerald-400 text-[9px] font-bold">
            BISI → iFVG
          </text>

          <text x="281" y="38" textAnchor="middle" className="fill-amber-700 dark:fill-amber-400 text-[9.5px] font-bold">
            Body close di atas ERL (FTM)
          </text>

          <line x1="198" y1="156" x2="508" y2="156" className="stroke-foreground" strokeDasharray="5 3" strokeWidth="1.35" />
          <text x="508" y="149" textAnchor="end" className="fill-foreground text-[9.5px] font-bold">
            Level CISD
          </text>

          <Candle x={320} open={60} close={136} high={54} low={142} />
          <Candle x={360} open={136} close={178} high={130} low={188} />
          <circle cx="371" cy="178" r="3.5" className="fill-emerald-600" />
          <text x="442" y="180" className="fill-emerald-700 dark:fill-emerald-400 text-[10px] font-bold">
            iFVG + Close tembus CISD
          </text>
          <text x="442" y="193" className="fill-muted-foreground text-[9.5px]">
            -CISD Kondisi 2 Valid
          </text>
        </>
      )}

      {kind === 'k3' && (
        <>
          <Candle x={198} open={150} close={112} high={104} low={158} />
          <Candle x={236} open={112} close={82} high={78} low={118} />
          <Candle x={274} open={82} close={58} high={46} low={88} />
          <text x="285" y="38" textAnchor="middle" className="fill-amber-700 dark:fill-amber-400 text-[9.5px] font-bold">
            FTM tanpa BISI/iFVG
          </text>

          <line x1="198" y1="150" x2="500" y2="150" className="stroke-foreground" strokeDasharray="5 3" strokeWidth="1.35" />
          <text x="500" y="143" textAnchor="end" className="fill-foreground text-[9.5px] font-bold">
            Level CISD
          </text>

          <Candle x={322} open={60} close={130} high={56} low={136} />
          <Candle x={362} open={130} close={188} high={124} low={198} />
          <circle cx="373" cy="188" r="3.5" className="fill-emerald-600" />
          <text x="444" y="188" className="fill-emerald-700 dark:fill-emerald-400 text-[10px] font-bold">
            SWL FTM + Tembus CISD
          </text>
          <text x="444" y="201" className="fill-muted-foreground text-[9.5px]">
            -CISD Kondisi 3 Valid
          </text>
        </>
      )}
    </svg>
  )
}

function EntryRiskSvg() {
  return (
    <svg
      viewBox="0 0 600 232"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label="Ilustrasi Entry pada close konfirmasi CISD, SL di puncak rangkaian, dan TP RR 1:2"
    >
      {/* Rangkaian menuju puncak & candle konfirmasi CISD:
          Open Bullish #1 = 86 (Level CISD), Peak High = 40 (SL), Close Konfirmasi = 98 (Entry) */}
      <Candle x={58} open={86} close={68} high={62} low={94} />
      <Candle x={94} open={68} close={54} high={40} low={74} />
      <Candle x={130} open={56} close={78} high={52} low={82} />
      <Candle x={166} open={78} close={98} high={74} low={104} />

      {/* Level CISD (y=86) */}
      <line x1="58" y1="86" x2="212" y2="86" className="stroke-muted-foreground" strokeDasharray="3 3" strokeWidth="1.1" />
      <text x="58" y="114" className="fill-muted-foreground text-[9px]">
        Level CISD
      </text>

      {/* Kotak Risk 1R (y=40..98 -> tinggi 58) & Reward 2R (y=98..214 -> tinggi 116) */}
      <rect
        x="212"
        y="40"
        width="256"
        height="58"
        fill="rgba(244,63,94,0.13)"
        stroke="rgba(244,63,94,0.45)"
        strokeWidth="1"
      />
      <rect
        x="212"
        y="98"
        width="256"
        height="116"
        fill="rgba(16,185,129,0.12)"
        stroke="rgba(16,185,129,0.45)"
        strokeWidth="1"
      />

      {/* Garis SL */}
      <line x1="105" y1="40" x2="468" y2="40" className="stroke-rose-600 dark:stroke-rose-400" strokeDasharray="5 3" strokeWidth="1.35" />
      <text x="476" y="44" className="fill-rose-600 dark:fill-rose-400 text-[10.5px] font-bold">
        SL (Puncak)
      </text>
      <text x="340" y="73" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[10px] font-bold">
        Risk · 1R
      </text>

      {/* Garis Entry tepat di close konfirmasi CISD (y=98) */}
      <line x1="177" y1="98" x2="468" y2="98" className="stroke-foreground" strokeWidth="1.35" />
      <circle cx="212" cy="98" r="4" className="fill-foreground" />
      <text x="476" y="102" className="fill-foreground text-[10.5px] font-bold">
        Entry (Close CISD)
      </text>

      {/* Candle lanjutan menuju TP */}
      <Candle x={242} open={98} close={134} high={90} low={142} />
      <Candle x={282} open={134} close={122} high={116} low={140} />
      <Candle x={322} open={122} close={172} high={118} low={180} />
      <Candle x={362} open={172} close={214} high={166} low={220} />

      <text x="428" y="162" textAnchor="middle" className="fill-emerald-700 dark:fill-emerald-400 text-[11px] font-bold">
        Reward · 2R
      </text>

      {/* Garis TP */}
      <line x1="212" y1="214" x2="468" y2="214" className="stroke-emerald-600 dark:stroke-emerald-400" strokeDasharray="5 3" strokeWidth="1.35" />
      <text x="476" y="218" className="fill-emerald-600 dark:fill-emerald-400 text-[10.5px] font-bold">
        TP (RR 1 : 2)
      </text>
    </svg>
  )
}

/**
 * Ila2FlowGuide menampilkan alur entry New ILA 2.0 dalam 4 tahap interaktif
 * lengkap dengan ilustrasi candlestick SVG.
 */
export function Ila2FlowGuide() {
  const [step, setStep] = useState<StepId>('cycle')
  const [htfKind, setHtfKind] = useState<HtfKind>('irl')
  const [cisdKind, setCisdKind] = useState<CisdKind>('k1')

  const idx = STEPS.findIndex((s) => s.id === step)

  return (
    <div className="space-y-4">
      {/* Stepper Tabs */}
      <div
        role="tablist"
        aria-label="Tahapan alur entry ILA 2.0"
        className="grid grid-cols-2 gap-1.5 sm:grid-cols-4"
      >
        {STEPS.map((s) => {
          const active = s.id === step
          return (
            <button
              key={s.id}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => setStep(s.id)}
              className={cn(
                'flex items-center justify-center gap-1.5 rounded-lg border px-2.5 py-2 text-xs font-semibold transition-all',
                active
                  ? 'border-foreground bg-foreground text-background shadow-xs'
                  : 'border-border bg-card text-muted-foreground hover:border-foreground/40 hover:text-foreground',
              )}
            >
              <span className="font-mono text-[10.5px] opacity-80">{s.num}.</span>
              <span>{s.label}</span>
            </button>
          )
        })}
      </div>

      {/* Konten Step 1: Time Cycle */}
      {step === 'cycle' && (
        <div className="space-y-3">
          <CycleDiagramSvg />
          <div className="overflow-hidden rounded-lg border border-border">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-border bg-secondary/40 text-[10.5px] uppercase tracking-wider text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 font-semibold">Time Cycle</th>
                  <th className="px-3 py-2 font-semibold">HTF → LTF</th>
                  <th className="hidden px-3 py-2 font-semibold sm:table-cell">Karakter</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/70">
                {CYCLES.map((c) => (
                  <tr key={c.name}>
                    <td className="px-3 py-2 font-semibold text-foreground">{c.name}</td>
                    <td className="px-3 py-2 font-mono text-[11.5px] text-foreground">
                      {c.htf} → {c.ltf}
                    </td>
                    <td className="hidden px-3 py-2 text-muted-foreground sm:table-cell">
                      {c.note}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-xs leading-relaxed text-muted-foreground">
            Aturan inti: candle <strong className="text-foreground">C2</strong> dan{' '}
            <strong className="text-foreground">C3</strong> wajib berada dalam satu time cycle yang
            sama. Jika lewat ke cycle berikutnya, setup dianggap tidak berlaku.
          </p>
        </div>
      )}

      {/* Konten Step 2: HTF (C2 Closure) */}
      {step === 'htf' && (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-xs font-semibold text-foreground">Jenis Key Level HTF:</span>
            <div className="inline-flex rounded-lg border border-border bg-secondary/40 p-0.5">
              {(
                [
                  { id: 'irl', label: 'IRL (SIBI / BISI)' },
                  { id: 'erl', label: 'ERL (Swing High / Low)' },
                ] as const
              ).map((opt) => (
                <button
                  key={opt.id}
                  type="button"
                  onClick={() => setHtfKind(opt.id)}
                  className={cn(
                    'rounded-md px-2.5 py-1 text-[11px] font-semibold transition-all',
                    htfKind === opt.id
                      ? 'bg-card text-foreground shadow-xs'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          {htfKind === 'irl' ? <HtfIrlSvg /> : <HtfErlSvg />}

          <ul className="space-y-1.5 text-xs leading-relaxed text-muted-foreground">
            {htfKind === 'irl' ? (
              <>
                <li>
                  • <strong className="text-foreground">IRL (SIBI / BISI)</strong>: gap 3-candle
                  pada HTF. Wick <strong className="text-foreground">C2</strong> menyentuh/melewati
                  acuan lalu body close valid (tidak menembus batas luar zona).
                </li>
                <li>
                  • Jika body close menembus batas luar zona → berubah menjadi{' '}
                  <strong className="text-foreground">iFVG</strong> (invalid).
                </li>
              </>
            ) : (
              <>
                <li>
                  • <strong className="text-foreground">ERL (Swing High / Low)</strong>: level
                  swing 3-candle HTF. Wick <strong className="text-foreground">C2</strong> menyapu
                  level lalu body close kembali di dalam.
                </li>
                <li>
                  • Jika body close menembus level ERL → menjadi{' '}
                  <strong className="text-foreground">FTM</strong> (invalid sebagai C2).
                </li>
              </>
            )}
            <li>
              • Setelah C2 valid close, sistem memantau LTF selama candle{' '}
              <strong className="text-foreground">C3</strong> berjalan. Bila C3 close tanpa setup →
              invalid.
            </li>
          </ul>
        </div>
      )}

      {/* Konten Step 3: LTF (ERL & CISD) */}
      {step === 'ltf' && (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-xs font-semibold text-foreground">Syarat Konfirmasi CISD:</span>
            <div className="inline-flex flex-wrap rounded-lg border border-border bg-secondary/40 p-0.5">
              {(
                [
                  { id: 'k1', label: 'Kondisi 1 · ERL Valid' },
                  { id: 'k2', label: 'Kondisi 2 · FTM + iFVG' },
                  { id: 'k3', label: 'Kondisi 3 · FTM + SWL' },
                ] as const
              ).map((opt) => (
                <button
                  key={opt.id}
                  type="button"
                  onClick={() => setCisdKind(opt.id)}
                  className={cn(
                    'rounded-md px-2 py-1 text-[11px] font-semibold transition-all',
                    cisdKind === opt.id
                      ? 'bg-card text-foreground shadow-xs'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <LtfCisdSvg kind={cisdKind} />

          <p className="text-xs leading-relaxed text-muted-foreground">
            {cisdKind === 'k1' && (
              <>
                <strong className="text-foreground">Kondisi 1 (ERL Valid + Wick):</strong> puncak
                rangkaian menyapu Swing High/Low LTF dengan wick (tanpa FTM), lalu candle pembalikan
                close menembus <strong className="text-foreground">Level CISD</strong> (open candle
                pertama rangkaian).
              </>
            )}
            {cisdKind === 'k2' && (
              <>
                <strong className="text-foreground">Kondisi 2 (FTM + iFVG):</strong> ERL LTF sudah
                FTM (body close lewat), tetapi ada FVG di dalam rangkaian yang berubah menjadi{' '}
                <strong className="text-foreground">iFVG</strong> dan close menembus Level CISD.
              </>
            )}
            {cisdKind === 'k3' && (
              <>
                <strong className="text-foreground">Kondisi 3 (FTM + SWL/SWH FTM):</strong> ERL LTF
                sudah FTM dan tidak ada iFVG di rangkaian, sehingga menunggu swing lawan (SWL/SWH)
                ikut FTM bersamaan dengan close menembus Level CISD.
              </>
            )}
          </p>
        </div>
      )}

      {/* Konten Step 4: Entry, SL, TP */}
      {step === 'entry' && (
        <div className="space-y-3">
          <EntryRiskSvg />
          <div className="grid gap-2 sm:grid-cols-3">
            <div className="rounded-lg border border-border bg-secondary/25 p-2.5">
              <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                Entry
              </p>
              <p className="mt-0.5 text-xs font-semibold text-foreground">
                Close konfirmasi CISD
              </p>
            </div>
            <div className="rounded-lg border border-rose-500/30 bg-rose-500/[0.04] p-2.5">
              <p className="text-[10px] font-semibold uppercase tracking-wider text-rose-600 dark:text-rose-400">
                Stop Loss (SL)
              </p>
              <p className="mt-0.5 text-xs font-semibold text-foreground">
                Puncak / dasar rangkaian
              </p>
            </div>
            <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/[0.04] p-2.5">
              <p className="text-[10px] font-semibold uppercase tracking-wider text-emerald-600 dark:text-emerald-400">
                Take Profit (TP)
              </p>
              <p className="mt-0.5 text-xs font-semibold text-foreground">
                Target tetap RR 1 : 2
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Navigasi Sebelumnya / Lanjut */}
      <div className="flex items-center justify-between border-t border-border pt-3">
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={idx <= 0}
          onClick={() => setStep(STEPS[idx - 1]!.id)}
          className="gap-1 text-xs"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Sebelumnya
        </Button>
        <span className="font-mono text-[11px] text-muted-foreground">
          {idx + 1} / {STEPS.length}
        </span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={idx >= STEPS.length - 1}
          onClick={() => setStep(STEPS[idx + 1]!.id)}
          className="gap-1 text-xs"
        >
          Lanjut
          <ArrowRight className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  )
}
