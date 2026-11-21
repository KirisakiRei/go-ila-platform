import { useState } from 'react'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui'
import { cn } from '@/lib/utils'

type StepId = 'cycle' | 'htf' | 'ltf' | 'entry'
type HtfKind = 'irl' | 'erl'
type CisdKind = 'k1' | 'k2' | 'k3'
type Direction = 'bearish' | 'bullish'

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
        className="stroke-foreground dark:stroke-zinc-400"
        strokeWidth="1.25"
      />
      <rect
        x={x}
        y={bodyTop}
        width={w}
        height={bodyH}
        rx="1"
        className={
          bearish
            ? 'fill-foreground stroke-foreground dark:fill-zinc-800 dark:stroke-zinc-400'
            : 'fill-card stroke-foreground dark:fill-zinc-100 dark:stroke-zinc-100'
        }
        strokeWidth="1.25"
      />
    </g>
  )
}

function CycleDiagramSvg({ dir }: { dir: Direction }) {
  const sell = dir === 'bearish'
  const C2 = sell
    ? { open: 114, close: 126, high: 66, low: 136 }
    : { open: 100, close: 88, high: 80, low: 140 }
  const sweepY = sell ? 66 : 140
  const C3 = sell
    ? { open: 126, close: 138, high: 116, low: 142 }
    : { open: 88, close: 76, high: 72, low: 98 }
  const outside = sell
    ? { open: 124, close: 108, high: 96, low: 132 }
    : { open: 112, close: 128, high: 104, low: 140 }

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
        fill={sell ? 'rgba(244,63,94,0.09)' : 'rgba(16,185,129,0.09)'}
        stroke={sell ? 'rgba(244,63,94,0.45)' : 'rgba(16,185,129,0.45)'}
        strokeWidth="1"
      />
      <Candle x={165} {...C2} />
      <circle cx="176" cy={sweepY} r="3" className={sell ? 'fill-rose-600' : 'fill-emerald-600'} />
      <text
        x="176"
        y="154"
        textAnchor="middle"
        className={
          sell
            ? 'fill-rose-600 dark:fill-rose-400 text-[10px] font-bold'
            : 'fill-emerald-700 dark:fill-emerald-400 text-[10px] font-bold'
        }
      >
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
      <Candle x={232} {...C3} />
      <text x="270" y="88" className="fill-emerald-700 dark:fill-emerald-400 text-[10.5px] font-bold">
        C3 Berjalan
      </text>
      <text x="270" y="103" className="fill-muted-foreground text-[9.5px]">
        Jendela cari CISD
      </text>
      <text x="270" y="116" className="fill-muted-foreground text-[9.5px]">
        di LTF (mis. M15)
      </text>
      <Candle x={436} {...outside} muted />
      <text x="494" y="100" textAnchor="middle" className="fill-muted-foreground text-[10px] font-semibold">
        Di luar cycle
      </text>
      <text x="494" y="114" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[9.5px] font-semibold">
        Setup tidak berlaku
      </text>
    </svg>
  )
}

function HtfIrlSvg({ dir }: { dir: Direction }) {
  const sell = dir === 'bearish'
  return (
    <svg
      viewBox="0 0 600 225"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label={
        sell
          ? 'Ilustrasi HTF C2 Closure pada IRL SIBI'
          : 'Ilustrasi HTF C2 Closure pada IRL BISI'
      }
    >
      {sell ? (
        <>
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
          <Candle x={62} open={44} close={62} high={26} low={72} />
          <Candle x={102} open={62} close={156} high={52} low={168} />
          <Candle x={142} open={156} close={174} high={138} low={186} />
          <Candle x={190} open={174} close={112} high={96} low={182} />
          <text x="201" y="198" textAnchor="middle" className="fill-muted-foreground text-[9px]">
            Masuk zona
          </text>

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
        </>
      ) : (
        <>
          {/* Zona BISI (FVG Bullish): antara Low C3 (y=96) dan High C1 (y=150) */}
          <rect
            x="84"
            y="96"
            width="318"
            height="54"
            fill="rgba(16,185,129,0.14)"
            stroke="rgba(16,185,129,0.45)"
            strokeWidth="1"
          />
          <text x="394" y="145" textAnchor="end" className="fill-emerald-700 dark:fill-emerald-400 text-[10px] font-bold">
            Zona BISI (IRL)
          </text>
          <Candle x={62} open={182} close={162} high={150} low={196} />
          <Candle x={102} open={162} close={62} high={52} low={172} />
          <Candle x={142} open={62} close={46} high={34} low={96} />
          <Candle x={190} open={46} close={112} high={36} low={124} />
          <text x="201" y="28" textAnchor="middle" className="fill-muted-foreground text-[9px]">
            Masuk zona
          </text>

          <line
            x1="201"
            y1="124"
            x2="386"
            y2="124"
            className="stroke-foreground"
            strokeDasharray="4 3"
            strokeWidth="1.2"
          />
          <text x="386" y="119" textAnchor="end" className="fill-foreground text-[9.5px] font-semibold">
            Acuan C2
          </text>

          <Candle x={244} open={114} close={102} high={88} low={154} />
          <circle cx="255" cy="154" r="3" className="fill-emerald-600" />
          <text x="255" y="78" textAnchor="middle" className="fill-emerald-700 dark:fill-emerald-400 text-[10.5px] font-bold">
            C2 Valid
          </text>
          <line x1="258" y1="156" x2="308" y2="188" className="stroke-emerald-500" strokeWidth="1" />
          <text x="313" y="191" className="fill-foreground text-[10px] font-semibold">
            Wick sentuh / lewati acuan di BISI
          </text>
          <text x="313" y="204" className="fill-muted-foreground text-[9.5px]">
            Body close tetap di atas batas bawah zona
          </text>
        </>
      )}

      {/* Jendela C3 */}
      <rect
        x="414"
        y={sell ? 64 : 24}
        width="164"
        height="138"
        rx="7"
        fill="rgba(16,185,129,0.08)"
        stroke="rgba(16,185,129,0.55)"
        strokeDasharray="4 3"
        strokeWidth="1.2"
      />
      {sell ? (
        <Candle x={436} open={128} close={168} high={116} low={180} />
      ) : (
        <Candle x={436} open={134} close={94} high={84} low={146} />
      )}
      <text x="472" y={sell ? 102 : 62} className="fill-emerald-700 dark:fill-emerald-400 text-[11px] font-bold">
        C3 Berjalan
      </text>
      <text x="472" y={sell ? 118 : 78} className="fill-muted-foreground text-[9.5px]">
        Cari {sell ? '-CISD' : '+CISD'} di LTF
      </text>
      <text x="472" y={sell ? 132 : 92} className="fill-muted-foreground text-[9.5px]">
        sebelum C3 close.
      </text>
      <text x="472" y={sell ? 152 : 112} className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        C3 close tanpa setup
      </text>
      <text x="472" y={sell ? 164 : 124} className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        = Invalid
      </text>
    </svg>
  )
}

function HtfErlSvg({ dir }: { dir: Direction }) {
  const sell = dir === 'bearish'
  return (
    <svg
      viewBox="0 0 600 225"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label={
        sell
          ? 'Ilustrasi HTF C2 Closure pada ERL Swing High'
          : 'Ilustrasi HTF C2 Closure pada ERL Swing Low'
      }
    >
      {sell ? (
        <>
          <Candle x={54} open={142} close={112} high={96} low={152} muted />
          <Candle x={94} open={112} close={90} high={68} low={120} />
          <Candle x={134} open={92} close={126} high={84} low={136} muted />
          <text x="105" y="56" textAnchor="middle" className="fill-foreground text-[10px] font-bold">
            Swing High (ERL)
          </text>
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
        </>
      ) : (
        <>
          <Candle x={54} open={94} close={124} high={84} low={140} muted />
          <Candle x={94} open={124} close={146} high={116} low={168} />
          <Candle x={134} open={144} close={110} high={100} low={152} muted />
          <text x="105" y="184" textAnchor="middle" className="fill-foreground text-[10px] font-bold">
            Swing Low (ERL)
          </text>
          <line
            x1="105"
            y1="168"
            x2="386"
            y2="168"
            className="stroke-emerald-600 dark:stroke-emerald-400"
            strokeDasharray="5 3"
            strokeWidth="1.35"
          />
          <text x="386" y="183" textAnchor="end" className="fill-emerald-600 dark:fill-emerald-400 text-[9.5px] font-bold">
            Level ERL HTF
          </text>
          <Candle x={182} open={110} close={90} high={82} low={118} muted />
          <Candle x={222} open={92} close={132} high={82} low={142} muted />
          <Candle x={268} open={134} close={118} high={106} low={190} />
          <circle cx="279" cy="190" r="3" className="fill-emerald-600" />
          <text x="279" y="96" textAnchor="middle" className="fill-emerald-700 dark:fill-emerald-400 text-[10.5px] font-bold">
            C2 Valid
          </text>
          <line x1="282" y1="191" x2="324" y2="201" className="stroke-emerald-500" strokeWidth="1" />
          <text x="330" y="199" className="fill-foreground text-[10px] font-semibold">
            Wick sapu di bawah Swing Low
          </text>
          <text x="330" y="212" className="fill-muted-foreground text-[9.5px]">
            Body C2 tetap close di atas level ERL
          </text>
        </>
      )}

      {/* Jendela C3 */}
      <rect
        x="414"
        y={sell ? 64 : 24}
        width="164"
        height="138"
        rx="7"
        fill="rgba(16,185,129,0.08)"
        stroke="rgba(16,185,129,0.55)"
        strokeDasharray="4 3"
        strokeWidth="1.2"
      />
      {sell ? (
        <Candle x={436} open={118} close={164} high={108} low={176} />
      ) : (
        <Candle x={436} open={134} close={94} high={84} low={146} />
      )}
      <text x="472" y={sell ? 102 : 62} className="fill-emerald-700 dark:fill-emerald-400 text-[11px] font-bold">
        C3 Berjalan
      </text>
      <text x="472" y={sell ? 118 : 78} className="fill-muted-foreground text-[9.5px]">
        Cari {sell ? '-CISD' : '+CISD'} di LTF
      </text>
      <text x="472" y={sell ? 132 : 92} className="fill-muted-foreground text-[9.5px]">
        selama C3 aktif.
      </text>
      <text x="472" y={sell ? 152 : 112} className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        Body close tembus ERL
      </text>
      <text x="472" y={sell ? 164 : 124} className="fill-rose-600 dark:fill-rose-400 text-[9px] font-semibold">
        = FTM (bukan C2)
      </text>
    </svg>
  )
}

function LtfCisdSvg({ kind, dir }: { kind: CisdKind; dir: Direction }) {
  const sell = dir === 'bearish'
  const kondisi =
    kind === 'k1' ? 'Kondisi 1' : kind === 'k2' ? 'Kondisi 2' : 'Kondisi 3'
  return (
    <svg
      viewBox="0 0 600 235"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label={`Ilustrasi LTF ${sell ? '-CISD' : '+CISD'} ${kondisi}`}
    >
      <text x="20" y="22" className="fill-muted-foreground text-[10px] font-semibold">
        Di dalam C3 HTF · Contoh Arah {sell ? 'Bearish (-CISD)' : 'Bullish (+CISD)'}
      </text>

      {sell ? (
        <>
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
        </>
      ) : (
        <>
          {/* ERL Swing Low LTF */}
          <Candle x={36} open={102} close={124} high={92} low={136} muted />
          <Candle x={68} open={124} close={140} high={116} low={162} />
          <Candle x={100} open={138} close={110} high={100} low={148} muted />
          <text x="79" y="180" textAnchor="middle" className="fill-foreground text-[9.5px] font-bold">
            ERL LTF
          </text>
          <line
            x1="79"
            y1="162"
            x2="372"
            y2="162"
            className="stroke-emerald-600 dark:stroke-emerald-400"
            strokeDasharray="4 3"
            strokeWidth="1.2"
          />
          <text x="172" y="176" className="fill-emerald-600 dark:fill-emerald-400 text-[9px] font-semibold">
            Swing Low LTF
          </text>
        </>
      )}

      {sell ? (
        <>
          {/* Kondisi 1: ERL valid (wick sweep) + close tembus CISD (open candle #1) */}
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

          {/* Kondisi 2: FTM (body close lewat ERL) + BISI menjadi iFVG */}
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
                Level CISD (Open Bullish #1)
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

          {/* Kondisi 3: FTM tanpa BISI/iFVG + SWL FTM; level CISD = LOW bullish #1 */}
          {kind === 'k3' && (
            <>
              <Candle x={140} open={126} close={156} high={120} low={170} />
              <Candle x={196} open={148} close={118} high={102} low={158} />
              <Candle x={234} open={118} close={92} high={78} low={124} />
              <Candle x={272} open={92} close={58} high={46} low={108} />
              <text x="285" y="36" textAnchor="middle" className="fill-amber-700 dark:fill-amber-400 text-[9.5px] font-bold">
                FTM tanpa BISI/iFVG
              </text>

              <line x1="207" y1="158" x2="500" y2="158" className="stroke-foreground" strokeDasharray="5 3" strokeWidth="1.35" />
              <text x="500" y="151" textAnchor="end" className="fill-foreground text-[9.5px] font-bold">
                Level CISD (Low Bullish #1)
              </text>

              <line x1="146" y1="170" x2="452" y2="170" className="stroke-amber-600 dark:stroke-amber-400" strokeDasharray="4 3" strokeWidth="1.1" />
              <text x="146" y="184" textAnchor="start" className="fill-amber-700 dark:fill-amber-400 text-[9px] font-bold">
                SWL FTM
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
        </>
      ) : (
        <>
          {/* Bullish: mirror vertikal dari bearish (yMid = 118) */}
          {/* Kondisi 1 */}
          {kind === 'k1' && (
            <>
              <Candle x={200} open={90} close={120} high={82} low={126} />
              <Candle x={238} open={120} close={144} high={114} low={152} />
              <Candle x={276} open={144} close={156} high={138} low={180} />
              <circle cx="287" cy="180" r="3" className="fill-emerald-600" />
              <text x="287" y="196" textAnchor="middle" className="fill-emerald-700 dark:fill-emerald-400 text-[9.5px] font-bold">
                Wick kena ERL (Valid)
              </text>

              <line x1="200" y1="90" x2="540" y2="90" className="stroke-foreground" strokeDasharray="5 3" strokeWidth="1.35" />
              <text x="540" y="103" textAnchor="end" className="fill-foreground text-[9.5px] font-bold">
                Level CISD (Open Bearish #1)
              </text>

              <Candle x={324} open={154} close={110} high={104} low={158} />
              <Candle x={364} open={110} close={68} high={58} low={116} />
              <circle cx="375" cy="68" r="3.5" className="fill-emerald-600" />
              <text x="444" y="64" className="fill-emerald-700 dark:fill-emerald-400 text-[10px] font-bold">
                Close tembus CISD
              </text>
              <text x="444" y="77" className="fill-muted-foreground text-[9.5px]">
                +CISD Kondisi 1 Valid
              </text>
            </>
          )}

          {/* Kondisi 2: FTM + SIBI menjadi iFVG */}
          {kind === 'k2' && (
            <>
              <Candle x={198} open={80} close={102} high={72} low={110} />
              <Candle x={234} open={102} close={148} high={96} low={154} />
              <Candle x={270} open={148} close={178} high={138} low={190} />

              <rect
                x="210"
                y="110"
                width="210"
                height="28"
                fill="rgba(244,63,94,0.14)"
                stroke="rgba(244,63,94,0.5)"
                strokeWidth="1"
              />
              <text x="414" y="127" textAnchor="end" className="fill-rose-600 dark:fill-rose-400 text-[9px] font-bold">
                SIBI → iFVG
              </text>

              <text x="281" y="204" textAnchor="middle" className="fill-amber-700 dark:fill-amber-400 text-[9.5px] font-bold">
                Body close di bawah ERL (FTM)
              </text>

              <line x1="198" y1="80" x2="540" y2="80" className="stroke-foreground" strokeDasharray="5 3" strokeWidth="1.35" />
              <text x="540" y="93" textAnchor="end" className="fill-foreground text-[9.5px] font-bold">
                Level CISD (Open Bearish #1)
              </text>

              <Candle x={320} open={176} close={100} high={94} low={182} />
              <Candle x={360} open={100} close={58} high={48} low={106} />
              <circle cx="371" cy="58" r="3.5" className="fill-emerald-600" />
              <text x="442" y="54" className="fill-emerald-700 dark:fill-emerald-400 text-[10px] font-bold">
                iFVG + Close tembus CISD
              </text>
              <text x="442" y="67" className="fill-muted-foreground text-[9.5px]">
                +CISD Kondisi 2 Valid
              </text>
            </>
          )}

          {/* Kondisi 3: FTM tanpa SIBI/iFVG + SWH FTM; level CISD = HIGH bearish #1 */}
          {kind === 'k3' && (
            <>
              <Candle x={140} open={110} close={80} high={66} low={116} />
              <Candle x={196} open={88} close={118} high={78} low={134} />
              <Candle x={234} open={118} close={144} high={112} low={158} />
              <Candle x={272} open={144} close={178} high={128} low={190} />
              <text x="285" y="204" textAnchor="middle" className="fill-amber-700 dark:fill-amber-400 text-[9.5px] font-bold">
                FTM tanpa SIBI/iFVG
              </text>

              <line x1="207" y1="78" x2="540" y2="78" className="stroke-foreground" strokeDasharray="5 3" strokeWidth="1.35" />
              <text x="540" y="91" textAnchor="end" className="fill-foreground text-[9.5px] font-bold">
                Level CISD (High Bearish #1)
              </text>

              <line x1="146" y1="66" x2="452" y2="66" className="stroke-amber-600 dark:stroke-amber-400" strokeDasharray="4 3" strokeWidth="1.1" />
              <text x="146" y="60" textAnchor="start" className="fill-amber-700 dark:fill-amber-400 text-[9px] font-bold">
                SWH FTM
              </text>

              <Candle x={322} open={176} close={106} high={100} low={180} />
              <Candle x={362} open={106} close={48} high={38} low={112} />
              <circle cx="373" cy="48" r="3.5" className="fill-emerald-600" />
              <text x="444" y="44" className="fill-emerald-700 dark:fill-emerald-400 text-[10px] font-bold">
                SWH FTM + Tembus CISD
              </text>
              <text x="444" y="57" className="fill-muted-foreground text-[9.5px]">
                +CISD Kondisi 3 Valid
              </text>
            </>
          )}
        </>
      )}
    </svg>
  )
}

function EntryRiskSvg({ dir }: { dir: Direction }) {
  const sell = dir === 'bearish'
  return (
    <svg
      viewBox="0 0 600 232"
      className="w-full h-auto rounded-lg border border-border/70 bg-background"
      role="img"
      aria-label={
        sell
          ? 'Ilustrasi Entry pada close konfirmasi CISD, SL di puncak rangkaian, dan TP RR 1:2'
          : 'Ilustrasi Entry pada close konfirmasi CISD, SL di dasar rangkaian, dan TP RR 1:2'
      }
    >
      {sell ? (
        <>
          {/* Level CISD = 86 (Open Bullish #1), Peak High = 40 (SL), Close Konfirmasi = 98 (Entry) */}
          <Candle x={58} open={86} close={68} high={62} low={94} />
          <Candle x={94} open={68} close={54} high={40} low={74} />
          <Candle x={130} open={56} close={78} high={52} low={82} />
          <Candle x={166} open={78} close={98} high={74} low={104} />

          <line x1="58" y1="86" x2="212" y2="86" className="stroke-muted-foreground" strokeDasharray="3 3" strokeWidth="1.1" />
          <text x="58" y="114" className="fill-muted-foreground text-[9px]">
            Level CISD
          </text>

          {/* Risk 1R (y=40..98) & Reward 2R (y=98..214) */}
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

          <line x1="105" y1="40" x2="468" y2="40" className="stroke-rose-600 dark:stroke-rose-400" strokeDasharray="5 3" strokeWidth="1.35" />
          <text x="476" y="44" className="fill-rose-600 dark:fill-rose-400 text-[10.5px] font-bold">
            SL (Puncak)
          </text>
          <text x="340" y="73" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[10px] font-bold">
            Risk · 1R
          </text>
          <line x1="177" y1="98" x2="468" y2="98" className="stroke-foreground" strokeWidth="1.35" />
          <circle cx="212" cy="98" r="4" className="fill-foreground" />
          <text x="476" y="102" className="fill-foreground text-[10.5px] font-bold">
            Entry (Close CISD)
          </text>
          <Candle x={242} open={98} close={134} high={90} low={142} />
          <Candle x={282} open={134} close={122} high={116} low={140} />
          <Candle x={322} open={122} close={172} high={118} low={180} />
          <Candle x={362} open={172} close={214} high={166} low={220} />
          <text x="428" y="162" textAnchor="middle" className="fill-emerald-700 dark:fill-emerald-400 text-[11px] font-bold">
            Reward · 2R
          </text>
          <line x1="212" y1="214" x2="468" y2="214" className="stroke-emerald-600 dark:stroke-emerald-400" strokeDasharray="5 3" strokeWidth="1.35" />
          <text x="476" y="218" className="fill-emerald-600 dark:fill-emerald-400 text-[10.5px] font-bold">
            TP (RR 1 : 2)
          </text>
        </>
      ) : (
        <>
          {/* Bullish: mirror vertikal (yMid = 116). Level CISD = 146 (Open Bearish #1),
              Swing Low = 192 (SL), Close Konfirmasi = 134 (Entry) */}
          <Candle x={58} open={146} close={164} high={138} low={170} />
          <Candle x={94} open={164} close={178} high={158} low={192} />
          <Candle x={130} open={176} close={154} high={150} low={180} />
          <Candle x={166} open={154} close={134} high={128} low={158} />

          <line x1="58" y1="146" x2="212" y2="146" className="stroke-muted-foreground" strokeDasharray="3 3" strokeWidth="1.1" />
          <text x="58" y="130" className="fill-muted-foreground text-[9px]">
            Level CISD
          </text>

          {/* Reward 2R (y=18..134) & Risk 1R (y=134..192) */}
          <rect
            x="212"
            y="18"
            width="256"
            height="116"
            fill="rgba(16,185,129,0.12)"
            stroke="rgba(16,185,129,0.45)"
            strokeWidth="1"
          />
          <rect
            x="212"
            y="134"
            width="256"
            height="58"
            fill="rgba(244,63,94,0.13)"
            stroke="rgba(244,63,94,0.45)"
            strokeWidth="1"
          />

          <line x1="105" y1="192" x2="468" y2="192" className="stroke-rose-600 dark:stroke-rose-400" strokeDasharray="5 3" strokeWidth="1.35" />
          <text x="476" y="196" className="fill-rose-600 dark:fill-rose-400 text-[10.5px] font-bold">
            SL (Dasar)
          </text>
          <text x="340" y="168" textAnchor="middle" className="fill-rose-600 dark:fill-rose-400 text-[10px] font-bold">
            Risk · 1R
          </text>
          <line x1="177" y1="134" x2="468" y2="134" className="stroke-foreground" strokeWidth="1.35" />
          <circle cx="212" cy="134" r="4" className="fill-foreground" />
          <text x="476" y="138" className="fill-foreground text-[10.5px] font-bold">
            Entry (Close CISD)
          </text>
          <Candle x={242} open={134} close={98} high={90} low={142} />
          <Candle x={282} open={98} close={110} high={92} low={116} />
          <Candle x={322} open={110} close={60} high={52} low={114} />
          <Candle x={362} open={60} close={18} high={12} low={66} />
          <text x="428" y="52" textAnchor="middle" className="fill-emerald-700 dark:fill-emerald-400 text-[11px] font-bold">
            Reward · 2R
          </text>
          <line x1="212" y1="18" x2="468" y2="18" className="stroke-emerald-600 dark:stroke-emerald-400" strokeDasharray="5 3" strokeWidth="1.35" />
          <text x="476" y="22" className="fill-emerald-600 dark:fill-emerald-400 text-[10.5px] font-bold">
            TP (RR 1 : 2)
          </text>
        </>
      )}
    </svg>
  )
}

/**
 * Ila2FlowGuide menampilkan alur entry New ILA 2.0 dalam 4 tahap interaktif
 * lengkap dengan ilustrasi candlestick SVG (dua arah: bearish & bullish).
 */
export function Ila2FlowGuide() {
  const [step, setStep] = useState<StepId>('cycle')
  const [htfKind, setHtfKind] = useState<HtfKind>('irl')
  const [cisdKind, setCisdKind] = useState<CisdKind>('k1')
  const [dir, setDir] = useState<Direction>('bearish')

  const idx = STEPS.findIndex((s) => s.id === step)
  const sell = dir === 'bearish'

  return (
    <div className="space-y-4">
      {/* Toggle arah setup */}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-xs font-semibold text-foreground">Arah Setup:</span>
        <div className="inline-flex rounded-lg border border-border bg-secondary/40 p-0.5">
          {(
            [
              { id: 'bearish', label: 'Bearish · Sell (-CISD)' },
              { id: 'bullish', label: 'Bullish · Buy (+CISD)' },
            ] as const
          ).map((opt) => (
            <button
              key={opt.id}
              type="button"
              onClick={() => setDir(opt.id)}
              className={cn(
                'rounded-md px-2.5 py-1 text-[11px] font-semibold transition-all',
                dir === opt.id
                  ? 'bg-card text-foreground shadow-xs'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

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
          <CycleDiagramSvg dir={dir} />
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
          {htfKind === 'irl' ? <HtfIrlSvg dir={dir} /> : <HtfErlSvg dir={dir} />}
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
                  { id: 'k3', label: sell ? 'Kondisi 3 · FTM + SWL' : 'Kondisi 3 · FTM + SWH' },
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
          <LtfCisdSvg kind={cisdKind} dir={dir} />
          <p className="text-xs leading-relaxed text-muted-foreground">
            {cisdKind === 'k1' && (
              <>
                <strong className="text-foreground">Kondisi 1 (ERL Valid + Wick):</strong>{' '}
                {sell
                  ? 'puncak rangkaian menyapu Swing High LTF dengan wick (tanpa FTM)'
                  : 'dasar rangkaian menyapu Swing Low LTF dengan wick (tanpa FTM)'}
                , lalu candle pembalikan close menembus{' '}
                <strong className="text-foreground">Level CISD</strong> (open candle pertama
                rangkaian).
              </>
            )}
            {cisdKind === 'k2' && (
              <>
                <strong className="text-foreground">Kondisi 2 (FTM + iFVG):</strong> ERL LTF sudah
                FTM (body close lewat), tetapi ada FVG di dalam rangkaian (
                {sell ? 'BISI' : 'SIBI'}) yang berubah menjadi{' '}
                <strong className="text-foreground">iFVG</strong> dan close menembus Level CISD
                (open candle pertama rangkaian).
              </>
            )}
            {cisdKind === 'k3' && (
              <>
                <strong className="text-foreground">Kondisi 3 (FTM + {sell ? 'SWL' : 'SWH'} FTM):</strong>{' '}
                ERL LTF sudah FTM{' '}
                {sell ? (
                  <>
                    dan rangkaian tidak membentuk BISI/iFVG, sehingga menunggu swing lawan (SWL) ikut
                    FTM
                  </>
                ) : (
                  <>
                    dan rangkaian tidak membentuk SIBI/iFVG, sehingga menunggu swing lawan (SWH) ikut
                    FTM
                  </>
                )}{' '}
                bersamaan dengan close menembus Level CISD. Level CISD diambil dari{' '}
                <strong className="text-foreground">
                  {sell ? 'low candle bullish pertama' : 'high candle bearish pertama'}
                </strong>{' '}
                pada rangkaian, bukan open.
              </>
            )}
          </p>
        </div>
      )}

      {/* Konten Step 4: Entry, SL, TP */}
      {step === 'entry' && (
        <div className="space-y-3">
          <EntryRiskSvg dir={dir} />
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
                {sell ? 'Puncak' : 'Dasar'} rangkaian
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
