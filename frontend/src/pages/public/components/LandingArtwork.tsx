import { useId } from 'react'

type ArtworkProps = {
  className?: string
}

/**
 * BrandMark adalah lambang geometris Imaginary Liquidity pada header dan footer.
 * Bentuk segi enam terbelah merepresentasikan sapuan likuiditas dan pemisahan
 * antara lapisan analisis dengan akun broker pengguna.
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
          <stop offset="0%" stopColor="#6d28d9" />
          <stop offset="55%" stopColor="#4c1d95" />
          <stop offset="100%" stopColor="#1e0948" />
        </linearGradient>
        <linearGradient
          id={`${uid}-stroke`}
          x1="6"
          y1="6"
          x2="26"
          y2="26"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0%" stopColor="#e9d5ff" stopOpacity="0.95" />
          <stop offset="100%" stopColor="#8b5cf6" stopOpacity="0.45" />
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
        stroke="#f5f3ff"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M11 16H21" stroke="#c4b5fd" strokeWidth="2" strokeLinecap="round" />
      <circle cx="16" cy="16" r="2.2" fill="#ffffff" />
    </svg>
  )
}

/**
 * HeroObsidianRings merender tiga cakram kaca obsidian mengambang dengan bias
 * cahaya dark purple sebagai latar tengah bagian hero.
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
        <radialGradient id={`${uid}-aura`} cx="50%" cy="52%" r="50%">
          <stop offset="0%" stopColor="#7c3aed" stopOpacity="0.38" />
          <stop offset="45%" stopColor="#4c1d95" stopOpacity="0.22" />
          <stop offset="100%" stopColor="#06040b" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={`${uid}-disc-dark`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#1b1130" stopOpacity="0.92" />
          <stop offset="50%" stopColor="#090612" stopOpacity="0.96" />
          <stop offset="100%" stopColor="#2e1065" stopOpacity="0.72" />
        </linearGradient>
        <linearGradient id={`${uid}-rim-bright`} x1="15%" y1="5%" x2="85%" y2="95%">
          <stop offset="0%" stopColor="#e9d5ff" stopOpacity="0.65" />
          <stop offset="35%" stopColor="#8b5cf6" stopOpacity="0.45" />
          <stop offset="70%" stopColor="#3b0764" stopOpacity="0.3" />
          <stop offset="100%" stopColor="#c4b5fd" stopOpacity="0.55" />
        </linearGradient>
        <linearGradient id={`${uid}-rim-soft`} x1="0%" y1="100%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#6d28d9" stopOpacity="0.55" />
          <stop offset="50%" stopColor="#ffffff" stopOpacity="0.16" />
          <stop offset="100%" stopColor="#a855f7" stopOpacity="0.5" />
        </linearGradient>
      </defs>

      <ellipse cx="480" cy="325" rx="430" ry="260" fill={`url(#${uid}-aura)`} />

      {/* Cakram kiri belakang */}
      <g transform="translate(275 305) rotate(-18)">
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
          fill="#06040b"
          fillOpacity="0.88"
          stroke="#7c3aed"
          strokeOpacity="0.35"
          strokeWidth="1.5"
        />
      </g>

      {/* Cakram kanan belakang */}
      <g transform="translate(685 265) rotate(22)">
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
          fill="#06040b"
          fillOpacity="0.9"
          stroke="#a855f7"
          strokeOpacity="0.35"
          strokeWidth="1.5"
        />
      </g>

      {/* Cakram utama tengah */}
      <g transform="translate(480 310) rotate(-8)">
        <ellipse
          cx="0"
          cy="0"
          rx="235"
          ry="148"
          fill={`url(#${uid}-disc-dark)`}
          stroke={`url(#${uid}-rim-bright)`}
          strokeWidth="2.4"
        />
        <ellipse
          cx="0"
          cy="0"
          rx="212"
          ry="132"
          stroke="#ffffff"
          strokeOpacity="0.08"
          strokeWidth="1"
        />
        <ellipse
          cx="0"
          cy="0"
          rx="132"
          ry="82"
          fill="#06040b"
          fillOpacity="0.92"
          stroke={`url(#${uid}-rim-bright)`}
          strokeWidth="1.8"
        />
      </g>

      {/* Partikel bias halus */}
      <circle cx="218" cy="182" r="2.2" fill="#d8b4fe" fillOpacity="0.6" />
      <circle cx="752" cy="154" r="2" fill="#c4b5fd" fillOpacity="0.55" />
      <circle cx="645" cy="458" r="2.5" fill="#a855f7" fillOpacity="0.5" />
    </svg>
  )
}

/**
 * HeroFinsPreview adalah kartu pratinjau kaca mini di sudut kanan atas hero
 * dengan motif bilah radial ungu gelap.
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
          <stop offset="0%" stopColor="#7c3aed" stopOpacity="0.65" />
          <stop offset="55%" stopColor="#3b0764" stopOpacity="0.45" />
          <stop offset="100%" stopColor="#090514" stopOpacity="0.95" />
        </radialGradient>
        <linearGradient id={`${uid}-fin`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#e9d5ff" stopOpacity="0.7" />
          <stop offset="50%" stopColor="#7c3aed" stopOpacity="0.4" />
          <stop offset="100%" stopColor="#1e0948" stopOpacity="0.2" />
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
          fill="#1b0b38"
          fillOpacity="0.42"
          stroke={`url(#${uid}-fin)`}
          strokeWidth="1"
        />
      ))}
      <circle
        cx="110"
        cy="78"
        r="14"
        fill="#090514"
        stroke="#c4b5fd"
        strokeOpacity="0.65"
        strokeWidth="1.2"
      />
      <circle cx="110" cy="78" r="4" fill="#e9d5ff" />
    </svg>
  )
}

/**
 * MetricsCurveSvg menggambar lintasan kurva menanjak dengan 4 titik node ungu
 * untuk section metrik faktual di atas kanvas #f3f3f6.
 */
export function MetricsCurveSvg({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  const nodes = [
    { x: 110, y: 204 },
    { x: 370, y: 158 },
    { x: 650, y: 110 },
    { x: 960, y: 50 },
  ]

  return (
    <svg
      viewBox="0 0 1120 260"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id={`${uid}-curve`} x1="0%" y1="100%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#c4b5fd" stopOpacity="0.35" />
          <stop offset="38%" stopColor="#7c3aed" stopOpacity="0.7" />
          <stop offset="100%" stopColor="#4c1d95" stopOpacity="0.95" />
        </linearGradient>
        <linearGradient id={`${uid}-area`} x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor="#7c3aed" stopOpacity="0.11" />
          <stop offset="100%" stopColor="#f3f3f6" stopOpacity="0" />
        </linearGradient>
      </defs>

      <path
        d="M 16 226 C 210 212, 340 164, 520 132 C 700 100, 860 72, 1104 26 L 1104 256 L 16 256 Z"
        fill={`url(#${uid}-area)`}
      />

      <path
        d="M 16 226 C 210 212, 340 164, 520 132 C 700 100, 860 72, 1104 26"
        stroke={`url(#${uid}-curve)`}
        strokeWidth="2.5"
        strokeLinecap="round"
      />

      {nodes.map((node) => (
        <g key={node.x}>
          <circle cx={node.x} cy={node.y} r="13" fill="#ede9fe" />
          <circle cx={node.x} cy={node.y} r="7" fill="#5b21b6" stroke="#ffffff" strokeWidth="2.5" />
        </g>
      ))}
    </svg>
  )
}

/**
 * SecurityChainArtwork adalah ilustrasi rantai metalik krom & violet gelap
 * yang diagonal mengilap untuk spotlight keamanan di kanvas putih.
 */
export function SecurityChainArtwork({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 640 420"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id={`${uid}-shadow`} cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="#4c1d95" stopOpacity="0.16" />
          <stop offset="65%" stopColor="#6d28d9" stopOpacity="0.06" />
          <stop offset="100%" stopColor="#ffffff" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={`${uid}-chrome`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#ffffff" />
          <stop offset="32%" stopColor="#cbd5e1" />
          <stop offset="68%" stopColor="#475569" />
          <stop offset="100%" stopColor="#0f172a" />
        </linearGradient>
        <linearGradient id={`${uid}-violet-metal`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#e9d5ff" />
          <stop offset="28%" stopColor="#8b5cf6" />
          <stop offset="65%" stopColor="#4c1d95" />
          <stop offset="100%" stopColor="#1e0948" />
        </linearGradient>
        <linearGradient id={`${uid}-obsidian-metal`} x1="15%" y1="0%" x2="85%" y2="100%">
          <stop offset="0%" stopColor="#64748b" />
          <stop offset="40%" stopColor="#1e1b4b" />
          <stop offset="100%" stopColor="#06040b" />
        </linearGradient>
      </defs>

      <ellipse cx="325" cy="230" rx="260" ry="135" fill={`url(#${uid}-shadow)`} />

      {/* Mata rantai 1 (kiri atas) */}
      <g transform="translate(150 115) rotate(-32)">
        <rect
          x="-95"
          y="-52"
          width="190"
          height="104"
          rx="52"
          stroke={`url(#${uid}-chrome)`}
          strokeWidth="30"
        />
        <rect
          x="-95"
          y="-52"
          width="190"
          height="104"
          rx="52"
          stroke="#ffffff"
          strokeOpacity="0.55"
          strokeWidth="2"
        />
      </g>

      {/* Mata rantai 2 (tengah ungu metalik) */}
      <g transform="translate(275 190) rotate(24)">
        <rect
          x="-105"
          y="-56"
          width="210"
          height="112"
          rx="56"
          stroke={`url(#${uid}-violet-metal)`}
          strokeWidth="32"
        />
        <rect
          x="-105"
          y="-56"
          width="210"
          height="112"
          rx="56"
          stroke="#f5f3ff"
          strokeOpacity="0.45"
          strokeWidth="1.5"
        />
      </g>

      {/* Mata rantai 3 (kanan tengah obsidian) */}
      <g transform="translate(405 260) rotate(-30)">
        <rect
          x="-100"
          y="-54"
          width="200"
          height="108"
          rx="54"
          stroke={`url(#${uid}-obsidian-metal)`}
          strokeWidth="30"
        />
        <rect
          x="-100"
          y="-54"
          width="200"
          height="108"
          rx="54"
          stroke="#a855f7"
          strokeOpacity="0.5"
          strokeWidth="1.8"
        />
      </g>

      {/* Simpul pengunci kanan bawah */}
      <g transform="translate(505 320) rotate(20)">
        <rect
          x="-72"
          y="-42"
          width="144"
          height="84"
          rx="42"
          stroke={`url(#${uid}-violet-metal)`}
          strokeWidth="24"
        />
      </g>
    </svg>
  )
}

/**
 * PurpleSilkWaves adalah karya abstrak gelombang sutra ungu gelap untuk kolom
 * tengah Asymmetric Showcase.
 */
export function PurpleSilkWaves({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 420 560"
      fill="none"
      preserveAspectRatio="xMidYMid slice"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id={`${uid}-base`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#120726" />
          <stop offset="50%" stopColor="#2e1065" />
          <stop offset="100%" stopColor="#090414" />
        </linearGradient>
        <linearGradient id={`${uid}-ribbon-a`} x1="0%" y1="10%" x2="100%" y2="90%">
          <stop offset="0%" stopColor="#a855f7" stopOpacity="0.82" />
          <stop offset="48%" stopColor="#5b21b6" stopOpacity="0.92" />
          <stop offset="100%" stopColor="#1e0948" stopOpacity="0.7" />
        </linearGradient>
        <linearGradient id={`${uid}-ribbon-b`} x1="100%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" stopColor="#e9d5ff" stopOpacity="0.65" />
          <stop offset="45%" stopColor="#7c3aed" stopOpacity="0.85" />
          <stop offset="100%" stopColor="#3b0764" stopOpacity="0.9" />
        </linearGradient>
      </defs>

      <rect width="420" height="560" fill={`url(#${uid}-base)`} />

      <path
        d="M -40 160 C 80 60, 220 260, 460 110 L 460 290 C 240 410, 100 220, -40 340 Z"
        fill={`url(#${uid}-ribbon-a)`}
      />
      <path
        d="M -40 320 C 120 180, 260 400, 460 250 L 460 450 C 260 560, 100 370, -40 490 Z"
        fill={`url(#${uid}-ribbon-b)`}
        fillOpacity="0.85"
      />
      <path
        d="M -20 230 C 110 120, 240 330, 440 185"
        stroke="#f3e8ff"
        strokeOpacity="0.48"
        strokeWidth="1.6"
      />
      <path
        d="M -20 390 C 130 270, 260 460, 440 325"
        stroke="#c4b5fd"
        strokeOpacity="0.38"
        strokeWidth="1.3"
      />
    </svg>
  )
}

/**
 * PurpleRibbonTile adalah ubin lanskap mini bergelombang ungu gelap.
 */
export function PurpleRibbonTile({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 320 160"
      fill="none"
      preserveAspectRatio="xMidYMid slice"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id={`${uid}-bg`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#1e0948" />
          <stop offset="100%" stopColor="#090414" />
        </linearGradient>
        <linearGradient id={`${uid}-wave`} x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#4c1d95" />
          <stop offset="55%" stopColor="#7c3aed" />
          <stop offset="100%" stopColor="#c4b5fd" />
        </linearGradient>
      </defs>
      <rect width="320" height="160" fill={`url(#${uid}-bg)`} />
      <path
        d="M -10 120 C 70 40, 180 150, 330 45 L 330 170 L -10 170 Z"
        fill={`url(#${uid}-wave)`}
        fillOpacity="0.72"
      />
      <path
        d="M -10 88 C 80 28, 195 126, 330 32"
        stroke="#e9d5ff"
        strokeOpacity="0.55"
        strokeWidth="1.5"
      />
    </svg>
  )
}

/**
 * PurpleOrbTile adalah grafik pusaran orbital ungu gelap untuk kartu kiri Bento Grid.
 */
export function PurpleOrbTile({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 320 180"
      fill="none"
      preserveAspectRatio="xMidYMid slice"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id={`${uid}-glow`} cx="50%" cy="50%" r="55%">
          <stop offset="0%" stopColor="#9333ea" stopOpacity="0.75" />
          <stop offset="55%" stopColor="#4c1d95" stopOpacity="0.5" />
          <stop offset="100%" stopColor="#0d051a" stopOpacity="1" />
        </radialGradient>
      </defs>
      <rect width="320" height="180" fill={`url(#${uid}-glow)`} />
      {[84, 64, 44, 26].map((r, idx) => (
        <ellipse
          key={r}
          cx="160"
          cy="90"
          rx={r * 1.45}
          ry={r * 0.72}
          transform={`rotate(${idx % 2 === 0 ? -14 : 12} 160 90)`}
          stroke="#e9d5ff"
          strokeOpacity={0.18 + idx * 0.14}
          strokeWidth="1.4"
        />
      ))}
      <circle cx="160" cy="90" r="10" fill="#f5f3ff" fillOpacity="0.92" />
    </svg>
  )
}

/**
 * CtaPrismArtwork adalah prisma/kubus kaca 3D dark-purple pada sisi kanan
 * banner CTA penutup yang gelap.
 */
export function CtaPrismArtwork({ className = '' }: ArtworkProps) {
  const uid = useId().replace(/:/g, '')
  return (
    <svg
      viewBox="0 0 420 340"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <radialGradient id={`${uid}-aura`} cx="52%" cy="50%" r="48%">
          <stop offset="0%" stopColor="#7c3aed" stopOpacity="0.55" />
          <stop offset="55%" stopColor="#4c1d95" stopOpacity="0.28" />
          <stop offset="100%" stopColor="#06040b" stopOpacity="0" />
        </radialGradient>
        <linearGradient id={`${uid}-top`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#e9d5ff" stopOpacity="0.38" />
          <stop offset="100%" stopColor="#6d28d9" stopOpacity="0.24" />
        </linearGradient>
        <linearGradient id={`${uid}-left`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#5b21b6" stopOpacity="0.5" />
          <stop offset="100%" stopColor="#1e0948" stopOpacity="0.85" />
        </linearGradient>
        <linearGradient id={`${uid}-right`} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#7c3aed" stopOpacity="0.45" />
          <stop offset="100%" stopColor="#0f071f" stopOpacity="0.92" />
        </linearGradient>
      </defs>

      <circle cx="215" cy="165" r="155" fill={`url(#${uid}-aura)`} />

      {/* Kubus dalam */}
      <polygon points="210,104 276,142 210,180 144,142" fill="#8b5cf6" fillOpacity="0.28" />
      <polygon points="144,142 210,180 210,250 144,212" fill="#3b0764" fillOpacity="0.65" />
      <polygon points="276,142 210,180 210,250 276,212" fill="#4c1d95" fillOpacity="0.55" />

      {/* Kubus kaca luar */}
      <polygon
        points="210,58 320,122 210,186 100,122"
        fill={`url(#${uid}-top)`}
        stroke="#e9d5ff"
        strokeOpacity="0.55"
        strokeWidth="1.5"
      />
      <polygon
        points="100,122 210,186 210,302 100,238"
        fill={`url(#${uid}-left)`}
        stroke="#a855f7"
        strokeOpacity="0.45"
        strokeWidth="1.5"
      />
      <polygon
        points="320,122 210,186 210,302 320,238"
        fill={`url(#${uid}-right)`}
        stroke="#c4b5fd"
        strokeOpacity="0.5"
        strokeWidth="1.5"
      />

      {/* Kilau tepi */}
      <line
        x1="210"
        y1="186"
        x2="210"
        y2="302"
        stroke="#f5f3ff"
        strokeOpacity="0.7"
        strokeWidth="1.8"
      />
      <circle cx="210" cy="186" r="4" fill="#ffffff" />
    </svg>
  )
}

/**
 * CtaInsetPreview adalah mini grafik alur sinyal/pipeline untuk kartu kaca kecil
 * di sudut kanan bawah banner CTA gelap.
 */
export function CtaInsetPreview({ className = '' }: ArtworkProps) {
  return (
    <svg
      viewBox="0 0 180 64"
      fill="none"
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <path
        d="M 10 44 C 42 44, 54 18, 90 18 C 126 18, 136 40, 170 22"
        stroke="#a855f7"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <circle cx="10" cy="44" r="3.5" fill="#c4b5fd" />
      <circle cx="90" cy="18" r="4" fill="#ffffff" stroke="#7c3aed" strokeWidth="2" />
      <circle cx="170" cy="22" r="3.5" fill="#c4b5fd" />
    </svg>
  )
}
