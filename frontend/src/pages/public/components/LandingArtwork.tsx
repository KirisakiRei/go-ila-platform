import { useId } from 'react'

type ArtworkProps = {
  className?: string
}

/**
 * BrandMark adalah lambang geometris monokrom Imaginary Liquidity pada header.
 */
export function BrandMark({ className = 'h-7 w-7' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg viewBox="0 0 32 32" fill="none" className={className} aria-hidden="true" focusable="false">
      <defs>
        <linearGradient
          id={`${uid}-bg`}
          x1="2"
          y1="2"
          x2="30"
          y2="30"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0%" stopColor="#27272a" />
          <stop offset="55%" stopColor="#09090b" />
          <stop offset="100%" stopColor="#050507" />
        </linearGradient>
        <linearGradient
          id={`${uid}-stroke`}
          x1="4"
          y1="4"
          x2="28"
          y2="28"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0.65" />
          <stop offset="100%" stopColor="#71717a" stopOpacity="0.3" />
        </linearGradient>
      </defs>
      <rect x="2" y="2" width="28" height="28" rx="8" fill={`url(#${uid}-bg)`} />
      <rect
        x="2.5"
        y="2.5"
        width="27"
        height="27"
        rx="7.5"
        stroke={`url(#${uid}-stroke)`}
        strokeWidth="1"
      />
      <path
        d="M9.5 11.5L16 8L22.5 11.5M9.5 20.5L16 24L22.5 20.5"
        stroke="#ffffff"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M11 16H21" stroke="#a1a1aa" strokeWidth="2" strokeLinecap="round" />
      <circle cx="16" cy="16" r="2.2" fill="#ffffff" />
    </svg>
  )
}

/**
 * HeroTradingGeometry merender latar geometris monokrom bertema trading yang
 * bersih: kisi koordinat halus, level likuiditas horizontal, serta struktur
 * swing harga tipis di sisi kanan tanpa mengganggu keterbacaan judul.
 */
export function HeroTradingGeometry({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  const cols = [120, 240, 360, 480, 600, 720, 840, 960, 1080]
  const rows = [110, 220, 330, 440, 550]
  const bars = [
    { x: 640, y: 310, h: 42, wickTop: 294, wickBottom: 368 },
    { x: 688, y: 282, h: 52, wickTop: 268, wickBottom: 346 },
    { x: 736, y: 318, h: 36, wickTop: 304, wickBottom: 374 },
    { x: 784, y: 254, h: 64, wickTop: 236, wickBottom: 332 },
    { x: 832, y: 222, h: 48, wickTop: 206, wickBottom: 284 },
    { x: 880, y: 244, h: 38, wickTop: 230, wickBottom: 298 },
    { x: 928, y: 184, h: 58, wickTop: 166, wickBottom: 256 },
    { x: 976, y: 152, h: 44, wickTop: 136, wickBottom: 214 },
    { x: 1024, y: 170, h: 34, wickTop: 154, wickBottom: 220 },
    { x: 1072, y: 118, h: 54, wickTop: 100, wickBottom: 186 },
  ]

  return (
    <svg
      viewBox="0 0 1200 660"
      fill="none"
      preserveAspectRatio="xMidYMid slice"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id={`${uid}-mask-grad`} cx="68%" cy="42%" r="58%">
          <stop offset="0%" stopColor="#ffffff" stopOpacity="1" />
          <stop offset="65%" stopColor="#ffffff" stopOpacity="0.45" />
          <stop offset="100%" stopColor="#ffffff" stopOpacity="0" />
        </radialGradient>
        <mask id={`${uid}-fade`}>
          <rect width="1200" height="660" fill={`url(#${uid}-mask-grad)`} />
        </mask>
        <linearGradient
          id={`${uid}-line`}
          x1="480"
          y1="450"
          x2="1140"
          y2="100"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0.06" />
          <stop offset="45%" stopColor="#ffffff" stopOpacity="0.32" />
          <stop offset="100%" stopColor="#ffffff" stopOpacity="0.14" />
        </linearGradient>
      </defs>

      <g mask={`url(#${uid}-fade)`}>
        {/* Kisi koordinat arsitektural */}
        {cols.map((x) => (
          <line
            key={`c-${x}`}
            x1={x}
            y1="0"
            x2={x}
            y2="660"
            stroke="#ffffff"
            strokeOpacity="0.05"
            strokeWidth="1"
          />
        ))}
        {rows.map((y) => (
          <line
            key={`r-${y}`}
            x1="0"
            y1={y}
            x2="1200"
            y2={y}
            stroke="#ffffff"
            strokeOpacity="0.05"
            strokeWidth="1"
          />
        ))}

        {/* Garis level likuiditas horizontal */}
        <line
          x1="440"
          y1="220"
          x2="1160"
          y2="220"
          stroke="#ffffff"
          strokeOpacity="0.16"
          strokeDasharray="4 6"
          strokeWidth="1"
        />
        <line
          x1="520"
          y1="330"
          x2="1140"
          y2="330"
          stroke="#ffffff"
          strokeOpacity="0.12"
          strokeDasharray="4 6"
          strokeWidth="1"
        />

        {/* Batang candlestick geometris minimalis di latar kanan */}
        {bars.map((bar) => (
          <g key={bar.x}>
            <line
              x1={bar.x + 8}
              y1={bar.wickTop}
              x2={bar.x + 8}
              y2={bar.wickBottom}
              stroke="#ffffff"
              strokeOpacity="0.14"
              strokeWidth="1"
            />
            <rect
              x={bar.x}
              y={bar.y}
              width="16"
              height={bar.h}
              rx="2"
              fill="#ffffff"
              fillOpacity="0.045"
              stroke="#ffffff"
              strokeOpacity="0.16"
              strokeWidth="1"
            />
          </g>
        ))}

        {/* Lintasan struktur swing high/low geometris */}
        <path
          d="M 520 420 L 648 334 L 744 366 L 840 242 L 888 278 L 984 176 L 1032 204 L 1128 118"
          stroke={`url(#${uid}-line)`}
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />

        {/* Titik simpul struktur */}
        {[
          { cx: 648, cy: 334 },
          { cx: 840, cy: 242 },
          { cx: 984, cy: 176 },
          { cx: 1128, cy: 118 },
        ].map((pt) => (
          <g key={`${pt.cx}-${pt.cy}`}>
            <circle cx={pt.cx} cy={pt.cy} r="7" fill="#ffffff" fillOpacity="0.06" />
            <circle
              cx={pt.cx}
              cy={pt.cy}
              r="3"
              fill="#050507"
              stroke="#ffffff"
              strokeOpacity="0.55"
              strokeWidth="1.5"
            />
          </g>
        ))}
      </g>
    </svg>
  )
}

/**
 * MetricsCurveSvg menggambar kurva kuadratik halus menanjak ke kanan-atas
 * (M 0 212 Q 600 212 1200 18) beserta 4 titik node dan garis vertikal `|`
 * yang turun sejajar ke masing-masing metrik.
 */
export function MetricsCurveSvg({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  const nodes = [
    { x: 96, y: 211, stemBottom: 324 },
    { x: 396, y: 191, stemBottom: 304 },
    { x: 696, y: 147, stemBottom: 260 },
    { x: 996, y: 78, stemBottom: 192 },
  ]

  return (
    <svg
      viewBox="0 0 1200 350"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id={`${uid}-curve`} x1="0%" y1="100%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#d4d4d8" stopOpacity="0.55" />
          <stop offset="45%" stopColor="#71717a" stopOpacity="0.85" />
          <stop offset="100%" stopColor="#09090b" stopOpacity="0.95" />
        </linearGradient>
      </defs>

      <path
        d="M 0 212 Q 600 212 1200 18"
        stroke={`url(#${uid}-curve)`}
        strokeWidth="1.6"
        strokeLinecap="round"
      />

      {nodes.map((node) => (
        <g key={node.x}>
          <line
            x1={node.x}
            y1={node.y + 6}
            x2={node.x}
            y2={node.stemBottom}
            stroke="#d4d4d8"
            strokeWidth="1.2"
          />
          <circle cx={node.x} cy={node.y} r="5.5" fill="#f5f5f7" stroke="#09090b" strokeWidth="2" />
        </g>
      ))}
    </svg>
  )
}

/**
 * ExecutionPreviewArtwork adalah ilustrasi antarmuka monokrom abstrak untuk
 * memperlihatkan alur pemantauan dan persetujuan sinyal tanpa jargon teknis.
 */
export function ExecutionPreviewArtwork({ className = '' }: ArtworkProps) {
  return (
    <svg
      viewBox="0 0 560 360"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <rect width="560" height="360" rx="20" fill="#f5f5f7" />
      <rect x="0.5" y="0.5" width="559" height="359" rx="19.5" stroke="#e4e4e7" />

      <rect x="64" y="42" width="432" height="252" rx="16" fill="#ffffff" stroke="#e4e4e7" />

      <circle cx="92" cy="70" r="5" fill="#09090b" />
      <rect x="108" y="66" width="112" height="8" rx="4" fill="#18181b" />
      <rect x="412" y="62" width="60" height="16" rx="8" fill="#f4f4f5" stroke="#d4d4d8" />

      <line x1="84" y1="94" x2="476" y2="94" stroke="#f4f4f5" strokeWidth="1.5" />

      <path
        d="M 88 214 L 146 184 L 194 202 L 266 138 L 324 166 L 398 118 L 468 134"
        stroke="#09090b"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <line
        x1="88"
        y1="158"
        x2="468"
        y2="158"
        stroke="#a1a1aa"
        strokeDasharray="4 4"
        strokeWidth="1.2"
      />
      <circle cx="324" cy="166" r="5" fill="#ffffff" stroke="#09090b" strokeWidth="2" />

      <rect x="116" y="224" width="344" height="96" rx="14" fill="#09090b" stroke="#27272a" />
      <rect x="140" y="248" width="96" height="9" rx="4.5" fill="#ffffff" />
      <rect x="140" y="266" width="154" height="7" rx="3.5" fill="#71717a" />
      <rect x="140" y="284" width="112" height="7" rx="3.5" fill="#52525b" />

      <rect x="336" y="256" width="100" height="32" rx="8" fill="#ffffff" />
      <rect x="358" y="269" width="56" height="6" rx="3" fill="#09090b" />
    </svg>
  )
}

/**
 * MonochromeLiquidityWaves adalah karya abstrak gelombang perak & obsidian
 * untuk bagian perbandingan ILA 2.0 & ILA 3.0.
 */
export function MonochromeLiquidityWaves({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 420 520"
      fill="none"
      preserveAspectRatio="xMidYMid slice"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id={`${uid}-base`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#18181b" />
          <stop offset="50%" stopColor="#09090b" />
          <stop offset="100%" stopColor="#050507" />
        </linearGradient>
        <linearGradient id={`${uid}-ribbon-a`} x1="0%" y1="10%" x2="100%" y2="90%">
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0.22" />
          <stop offset="48%" stopColor="#52525b" stopOpacity="0.45" />
          <stop offset="100%" stopColor="#09090b" stopOpacity="0.85" />
        </linearGradient>
        <linearGradient id={`${uid}-ribbon-b`} x1="100%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor="#f4f4f5" stopOpacity="0.28" />
          <stop offset="45%" stopColor="#3f3f46" stopOpacity="0.55" />
          <stop offset="100%" stopColor="#18181b" stopOpacity="0.9" />
        </linearGradient>
      </defs>

      <rect width="420" height="520" fill={`url(#${uid}-base)`} />

      <path
        d="M -40 150 C 80 50, 220 245, 460 100 L 460 275 C 240 390, 100 205, -40 325 Z"
        fill={`url(#${uid}-ribbon-a)`}
      />
      <path
        d="M -40 305 C 120 170, 260 380, 460 235 L 460 425 C 260 530, 100 350, -40 465 Z"
        fill={`url(#${uid}-ribbon-b)`}
      />
      <path
        d="M -20 218 C 110 112, 240 315, 440 175"
        stroke="#ffffff"
        strokeOpacity="0.45"
        strokeWidth="1.4"
      />
      <path
        d="M -20 372 C 130 255, 260 438, 440 308"
        stroke="#d4d4d8"
        strokeOpacity="0.32"
        strokeWidth="1.2"
      />
    </svg>
  )
}

/**
 * MonochromeHorizonTile adalah ubin lanskap mini bergelombang monokrom.
 */
export function MonochromeHorizonTile({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 320 150"
      fill="none"
      preserveAspectRatio="xMidYMid slice"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id={`${uid}-bg`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#18181b" />
          <stop offset="100%" stopColor="#050507" />
        </linearGradient>
        <linearGradient id={`${uid}-wave`} x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#27272a" />
          <stop offset="55%" stopColor="#52525b" />
          <stop offset="100%" stopColor="#d4d4d8" />
        </linearGradient>
      </defs>
      <rect width="320" height="150" fill={`url(#${uid}-bg)`} />
      <path
        d="M -10 112 C 75 36, 185 140, 330 42 L 330 160 L -10 160 Z"
        fill={`url(#${uid}-wave)`}
        fillOpacity="0.58"
      />
      <path
        d="M -10 82 C 80 26, 195 118, 330 30"
        stroke="#ffffff"
        strokeOpacity="0.5"
        strokeWidth="1.4"
      />
    </svg>
  )
}
