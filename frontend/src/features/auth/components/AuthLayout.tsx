import type { ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'
import { Logo } from '@/components/brand/Logo'
import { useApplyUserDashboardTheme } from '@/lib/theme'

type CandleBar = {
  x: number
  bodyTop: number
  bodyHeight: number
  wickTop: number
  wickBottom: number
  markerY?: number
}

const LEFT_CANDLES: CandleBar[] = [
  { x: 68, bodyTop: 538, bodyHeight: 52, wickTop: 496, wickBottom: 625 },
  { x: 128, bodyTop: 486, bodyHeight: 66, wickTop: 446, wickBottom: 584 },
  { x: 188, bodyTop: 510, bodyHeight: 48, wickTop: 470, wickBottom: 592 },
  { x: 248, bodyTop: 402, bodyHeight: 72, wickTop: 364, wickBottom: 510, markerY: 410 },
  { x: 308, bodyTop: 428, bodyHeight: 52, wickTop: 390, wickBottom: 512 },
  { x: 368, bodyTop: 318, bodyHeight: 70, wickTop: 282, wickBottom: 424, markerY: 346 },
]

const RIGHT_CANDLES: CandleBar[] = [
  { x: 1068, bodyTop: 504, bodyHeight: 64, wickTop: 464, wickBottom: 602 },
  { x: 1128, bodyTop: 534, bodyHeight: 48, wickTop: 498, wickBottom: 614 },
  { x: 1188, bodyTop: 416, bodyHeight: 72, wickTop: 378, wickBottom: 524, markerY: 422 },
  { x: 1248, bodyTop: 436, bodyHeight: 46, wickTop: 398, wickBottom: 516 },
  { x: 1308, bodyTop: 330, bodyHeight: 66, wickTop: 292, wickBottom: 434 },
  { x: 1368, bodyTop: 266, bodyHeight: 60, wickTop: 228, wickBottom: 360, markerY: 310 },
]

const MOBILE_CANDLES: CandleBar[] = [
  { x: 34, bodyTop: 86, bodyHeight: 28, wickTop: 64, wickBottom: 132 },
  { x: 96, bodyTop: 68, bodyHeight: 34, wickTop: 48, wickBottom: 118 },
  { x: 158, bodyTop: 78, bodyHeight: 26, wickTop: 58, wickBottom: 120 },
  { x: 220, bodyTop: 46, bodyHeight: 38, wickTop: 26, wickBottom: 98, markerY: 50 },
  { x: 282, bodyTop: 56, bodyHeight: 28, wickTop: 38, wickBottom: 102 },
  { x: 344, bodyTop: 28, bodyHeight: 34, wickTop: 12, wickBottom: 78, markerY: 34 },
]

function DesktopBackdrop() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none fixed inset-0 hidden overflow-hidden select-none sm:block"
    >
      <svg
        viewBox="0 0 1440 900"
        preserveAspectRatio="xMidYMid slice"
        className="h-full w-full text-zinc-900 dark:text-white"
        fill="none"
      >
        {/* Tiga garis struktur horizontal penuh viewport */}
        <line x1="0" y1="207" x2="1440" y2="207" stroke="currentColor" strokeOpacity="0.07" strokeWidth="1" />
        <line x1="0" y1="450" x2="1440" y2="450" stroke="currentColor" strokeOpacity="0.06" strokeWidth="1" />
        <line x1="0" y1="693" x2="1440" y2="693" stroke="currentColor" strokeOpacity="0.07" strokeWidth="1" />

        {/* Label sudut latar desktop */}
        <g
          className="hidden lg:block"
          fill="currentColor"
          fillOpacity="0.22"
          fontSize="10"
          fontFamily="ui-monospace, SFMono-Regular, Menlo, monospace"
          letterSpacing="1.6"
        >
          <text x="36" y="198">ERL SWEEP</text>
          <text x="1404" y="198" textAnchor="end">CISD · M5</text>
          <text x="36" y="683">IRL · FVG</text>
          <text x="1404" y="683" textAnchor="end">TIME CYCLE</text>
        </g>

        {/* Garis penghubung struktur & candle sisi kiri */}
        <g className="hidden md:block">
          <path
            d="M 0 612 L 68 544 L 128 570 L 188 518 L 248 416 L 308 456 L 368 346 L 470 384"
            stroke="currentColor"
            strokeOpacity="0.11"
            strokeWidth="1"
          />
          {LEFT_CANDLES.map((bar) => (
            <g key={bar.x}>
              <line
                x1={bar.x}
                y1={bar.wickTop}
                x2={bar.x}
                y2={bar.wickBottom}
                stroke="currentColor"
                strokeOpacity="0.12"
                strokeWidth="1"
              />
              <rect
                x={bar.x - 7}
                y={bar.bodyTop}
                width="14"
                height={bar.bodyHeight}
                rx="1"
                fill="currentColor"
                fillOpacity="0.045"
                stroke="currentColor"
                strokeOpacity="0.15"
                strokeWidth="1"
              />
              {bar.markerY !== undefined && (
                <circle
                  cx={bar.x}
                  cy={bar.markerY}
                  r="2.4"
                  fill="none"
                  stroke="currentColor"
                  strokeOpacity="0.32"
                  strokeWidth="1"
                />
              )}
            </g>
          ))}
        </g>

        {/* Garis penghubung struktur & candle sisi kanan */}
        <g className="hidden md:block">
          <path
            d="M 970 542 L 1068 504 L 1128 534 L 1188 422 L 1248 456 L 1308 352 L 1368 310 L 1440 264"
            stroke="currentColor"
            strokeOpacity="0.11"
            strokeWidth="1"
          />
          {RIGHT_CANDLES.map((bar) => (
            <g key={bar.x}>
              <line
                x1={bar.x}
                y1={bar.wickTop}
                x2={bar.x}
                y2={bar.wickBottom}
                stroke="currentColor"
                strokeOpacity="0.12"
                strokeWidth="1"
              />
              <rect
                x={bar.x - 7}
                y={bar.bodyTop}
                width="14"
                height={bar.bodyHeight}
                rx="1"
                fill="currentColor"
                fillOpacity="0.045"
                stroke="currentColor"
                strokeOpacity="0.15"
                strokeWidth="1"
              />
              {bar.markerY !== undefined && (
                <circle
                  cx={bar.x}
                  cy={bar.markerY}
                  r="2.4"
                  fill="none"
                  stroke="currentColor"
                  strokeOpacity="0.32"
                  strokeWidth="1"
                />
              )}
            </g>
          ))}
        </g>
      </svg>
    </div>
  )
}

function MobileBackdrop() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-x-0 bottom-0 h-44 overflow-hidden select-none sm:hidden"
    >
      <svg
        viewBox="0 0 380 150"
        preserveAspectRatio="xMidYMax slice"
        className="h-full w-full text-zinc-900 dark:text-white"
        fill="none"
      >
        <line x1="0" y1="36" x2="380" y2="36" stroke="currentColor" strokeOpacity="0.05" strokeWidth="1" />
        <line x1="0" y1="108" x2="380" y2="108" stroke="currentColor" strokeOpacity="0.05" strokeWidth="1" />
        <path
          d="M 0 118 L 34 92 L 96 102 L 158 82 L 220 50 L 282 70 L 344 34 L 380 22"
          stroke="currentColor"
          strokeOpacity="0.08"
          strokeWidth="1"
        />
        {MOBILE_CANDLES.map((bar) => (
          <g key={bar.x}>
            <line
              x1={bar.x}
              y1={bar.wickTop}
              x2={bar.x}
              y2={bar.wickBottom}
              stroke="currentColor"
              strokeOpacity="0.08"
              strokeWidth="1"
            />
            <rect
              x={bar.x - 5}
              y={bar.bodyTop}
              width="10"
              height={bar.bodyHeight}
              rx="1"
              fill="currentColor"
              fillOpacity="0.03"
              stroke="currentColor"
              strokeOpacity="0.1"
              strokeWidth="1"
            />
            {bar.markerY !== undefined && (
              <circle
                cx={bar.x}
                cy={bar.markerY}
                r="2"
                fill="none"
                stroke="currentColor"
                strokeOpacity="0.22"
                strokeWidth="1"
              />
            )}
          </g>
        ))}
      </svg>
    </div>
  )
}

/**
 * AuthLayout memusatkan formulir autentikasi dalam satu layar penuh (`100dvh`)
 * tanpa scroll halaman, dengan latar candlestick monokrom yang menyesuaikan
 * tema terang/gelap serta proporsi khusus perangkat mobile.
 *
 * Tautan ke beranda dan dokumen legal memakai anchor biasa karena situs publik
 * dilayani oleh bundel terpisah di luar router `/app`.
 */
export function AuthLayout({ children, footer }: { children: ReactNode; footer: ReactNode }) {
  useApplyUserDashboardTheme()

  return (
    <div className="relative flex h-[100dvh] w-full justify-center overflow-hidden bg-white text-foreground dark:bg-[#0c0c0e] sm:bg-[#f6f6f7] dark:sm:bg-[#09090b]">
      <DesktopBackdrop />

      <div className="relative z-10 flex h-full w-full max-w-[480px] flex-col overflow-hidden bg-white dark:bg-[#0c0c0e] sm:border-x sm:border-zinc-200/85 dark:sm:border-white/[0.08]">
        <header className="relative z-10 flex shrink-0 items-center justify-between gap-3 border-b border-zinc-200/80 px-5 pt-[max(1rem,env(safe-area-inset-top))] pb-4 dark:border-white/[0.08] sm:px-10 sm:py-6">
          <a href="/" className="group inline-flex min-w-0 items-center gap-2.5">
            <Logo
              alt=""
              className="h-5 w-auto shrink-0 transition-transform duration-150 group-hover:scale-105 sm:h-6"
            />
            <span className="truncate text-[14px] font-semibold tracking-tight text-foreground sm:text-[15px]">
              Imaginary Liquidity
            </span>
          </a>

          <a
            href="/"
            className="inline-flex shrink-0 items-center gap-1.5 rounded-md py-1 text-[12.5px] font-medium text-muted-foreground transition-colors hover:text-foreground sm:text-[13px]"
          >
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
            Beranda
          </a>
        </header>

        <main className="relative flex min-h-0 flex-1 flex-col overflow-y-auto scrollbar-none px-5 py-5 sm:px-10 sm:py-8">
          <MobileBackdrop />
          <div className="relative z-10 my-auto w-full">
            {children}
            {footer && (
              <div className="mt-6 text-center text-[13px] text-muted-foreground sm:mt-7">
                {footer}
              </div>
            )}
          </div>
        </main>

        <footer className="relative z-10 flex shrink-0 flex-wrap items-center justify-between gap-2 border-t border-zinc-200/80 px-5 pt-3.5 pb-[max(0.875rem,env(safe-area-inset-bottom))] text-[11px] text-muted-foreground dark:border-white/[0.08] sm:px-10 sm:py-5 sm:text-[11.5px]">
          <span>© {new Date().getFullYear()} Imaginary Liquidity</span>
          <nav aria-label="Tautan legal" className="flex items-center gap-2">
            <a href="/legal/privacy" className="transition-colors hover:text-foreground">
              Privasi
            </a>
            <span aria-hidden="true" className="text-muted-foreground/50">
              ·
            </span>
            <a href="/legal/risk" className="transition-colors hover:text-foreground">
              Disclaimer Risiko
            </a>
          </nav>
        </footer>
      </div>
    </div>
  )
}
