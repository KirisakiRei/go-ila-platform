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
 * HeroObsidianRings merender cakram kaca obsidian mengambang dengan kilau perak
 * di belakang judul utama hero.
 */
export function HeroObsidianRings({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 960 620"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id={`${uid}-aura`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0.12" />
          <stop offset="48%" stopColor="#a1a1aa" stopOpacity="0.06" />
          <stop offset="100%" stopColor="#050507" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={`${uid}-disc-dark`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#27272a" stopOpacity="0.88" />
          <stop offset="50%" stopColor="#09090b" stopOpacity="0.96" />
          <stop offset="100%" stopColor="#18181b" stopOpacity="0.82" />
        </linearGradient>
        <linearGradient id={`${uid}-rim-bright`} x1="12%" y1="4%" x2="88%" y2="96%">
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0.72" />
          <stop offset="35%" stopColor="#a1a1aa" stopOpacity="0.32" />
          <stop offset="70%" stopColor="#3f3f46" stopOpacity="0.25" />
          <stop offset="100%" stopColor="#e4e4e7" stopOpacity="0.58" />
        </linearGradient>
        <linearGradient id={`${uid}-rim-soft`} x1="0%" y1="100%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#71717a" stopOpacity="0.48" />
          <stop offset="50%" stopColor="#ffffff" stopOpacity="0.18" />
          <stop offset="100%" stopColor="#d4d4d8" stopOpacity="0.45" />
        </linearGradient>
      </defs>

      <ellipse cx="480" cy="315" rx="430" ry="255" fill={`url(#${uid}-aura)`} />

      {/* Cakram kiri */}
      <g transform="translate(260 300) rotate(-18)">
        <ellipse
          cx="0"
          cy="0"
          rx="168"
          ry="112"
          fill={`url(#${uid}-disc-dark)`}
          stroke={`url(#${uid}-rim-soft)`}
          strokeWidth="2"
        />
        <ellipse
          cx="0"
          cy="0"
          rx="105"
          ry="66"
          fill="#050507"
          fillOpacity="0.9"
          stroke="#ffffff"
          strokeOpacity="0.22"
          strokeWidth="1.4"
        />
      </g>

      {/* Cakram kanan */}
      <g transform="translate(690 258) rotate(22)">
        <ellipse
          cx="0"
          cy="0"
          rx="156"
          ry="102"
          fill={`url(#${uid}-disc-dark)`}
          stroke={`url(#${uid}-rim-soft)`}
          strokeWidth="2"
        />
        <ellipse
          cx="0"
          cy="0"
          rx="96"
          ry="60"
          fill="#050507"
          fillOpacity="0.92"
          stroke="#e4e4e7"
          strokeOpacity="0.24"
          strokeWidth="1.4"
        />
      </g>

      {/* Cakram utama tengah */}
      <g transform="translate(478 308) rotate(-8)">
        <ellipse
          cx="0"
          cy="0"
          rx="236"
          ry="148"
          fill={`url(#${uid}-disc-dark)`}
          stroke={`url(#${uid}-rim-bright)`}
          strokeWidth="2.2"
        />
        <ellipse
          cx="0"
          cy="0"
          rx="212"
          ry="132"
          stroke="#ffffff"
          strokeOpacity="0.09"
          strokeWidth="1"
        />
        <ellipse
          cx="0"
          cy="0"
          rx="132"
          ry="82"
          fill="#050507"
          fillOpacity="0.94"
          stroke={`url(#${uid}-rim-bright)`}
          strokeWidth="1.6"
        />
      </g>

      <circle cx="215" cy="180" r="2" fill="#ffffff" fillOpacity="0.55" />
      <circle cx="755" cy="154" r="1.8" fill="#d4d4d8" fillOpacity="0.5" />
      <circle cx="640" cy="452" r="2" fill="#a1a1aa" fillOpacity="0.45" />
    </svg>
  )
}

/**
 * HeroFinsPreview adalah kartu pratinjau kaca monokrom di sudut kanan atas hero.
 */
export function HeroFinsPreview({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 220 132"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id={`${uid}-core`} cx="50%" cy="60%" r="65%">
          <stop offset="0%" stopColor="#52525b" stopOpacity="0.55" />
          <stop offset="55%" stopColor="#18181b" stopOpacity="0.7" />
          <stop offset="100%" stopColor="#050507" stopOpacity="0.98" />
        </radialGradient>
        <linearGradient id={`${uid}-fin`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0.65" />
          <stop offset="50%" stopColor="#a1a1aa" stopOpacity="0.3" />
          <stop offset="100%" stopColor="#27272a" stopOpacity="0.2" />
        </linearGradient>
      </defs>
      <rect width="220" height="132" rx="10" fill={`url(#${uid}-core)`} />
      {[-52, -38, -24, -10, 4, 18, 32, 46].map((angle) => (
        <ellipse
          key={angle}
          cx="110"
          cy="78"
          rx="74"
          ry="22"
          transform={`rotate(${angle} 110 78)`}
          fill="#09090b"
          fillOpacity="0.55"
          stroke={`url(#${uid}-fin)`}
          strokeWidth="1"
        />
      ))}
      <circle
        cx="110"
        cy="78"
        r="14"
        fill="#050507"
        stroke="#e4e4e7"
        strokeOpacity="0.65"
        strokeWidth="1.2"
      />
      <circle cx="110" cy="78" r="3.8" fill="#ffffff" />
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
  // Untuk Bezier kuadratik dengan P0.x=0, P1.x=600, P2.x=1200, posisi x linear terhadap t:
  // t = x / 1200 dan y(t) = 212*(1 - t^2) + 18*(t^2).
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

      {/* Kartu belakang */}
      <rect x="64" y="42" width="432" height="252" rx="16" fill="#ffffff" stroke="#e4e4e7" />

      {/* Header mini bar */}
      <circle cx="92" cy="70" r="5" fill="#09090b" />
      <rect x="108" y="66" width="112" height="8" rx="4" fill="#18181b" />
      <rect x="412" y="62" width="60" height="16" rx="8" fill="#f4f4f5" stroke="#d4d4d8" />

      <line x1="84" y1="94" x2="476" y2="94" stroke="#f4f4f5" strokeWidth="1.5" />

      {/* Mini garis struktur harga */}
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

      {/* Kartu aksi depan melayang */}
      <rect x="116" y="224" width="344" height="96" rx="14" fill="#09090b" stroke="#27272a" />
      <rect x="140" y="248" width="96" height="9" rx="4.5" fill="#ffffff" />
      <rect x="140" y="266" width="154" height="7" rx="3.5" fill="#71717a" />
      <rect x="140" y="284" width="112" height="7" rx="3.5" fill="#52525b" />

      {/* Tombol aksi monokrom */}
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
