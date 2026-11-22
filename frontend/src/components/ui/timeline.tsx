import { useRef } from 'react'
import { ArrowDown } from 'lucide-react'
import {
  motion,
  useReducedMotion,
  useScroll,
  useTransform,
  type MotionValue,
} from 'framer-motion'

export type TimelineMilestone = {
  id: string
  version: string
  dateLabel: string
  keywords: string
  content: string
  side: 'top' | 'bottom'
}

export type TimelineProps = {
  eyebrow?: string
  headingPrimary?: string
  headingSecondary?: string
  subtitle?: string
  periodLabel?: string
  items?: TimelineMilestone[]
}

const EASE_OUT_STRONG = [0.23, 1, 0.32, 1] as const

export const DEFAULT_ILA_MILESTONES: TimelineMilestone[] = [
  {
    id: 'ila-1-0',
    version: 'ILA 1.0',
    dateLabel: 'Oktober 2024',
    keywords: 'ERL, IRL & SMT',
    content:
      'Fondasi baca likuiditas: sweep swing high/low (ERL), area FVG (IRL), dan divergensi SMT antar pair berkorelasi.',
    side: 'top',
  },
  {
    id: 'ila-2-0',
    version: 'ILA 2.0',
    dateLabel: 'Mei 2025',
    keywords: 'Time Cycle · C2 Sweep · CISD',
    content:
      'Setup berbasis Time Cycle: tunggu C2 sweep di HTF, lalu turun ke LTF cari konfirmasi CISD — wajib closed candle dengan RR terukur.',
    side: 'bottom',
  },
  {
    id: 'ila-2-1',
    version: 'ILA 2.1',
    dateLabel: 'November 2025',
    keywords: 'ILA 2.0 + Filter SMT',
    content:
      'Penyempurnaan ILA 2.0 dengan filter SMT buat nyaring fake sweep sebelum ambil trigger CISD di LTF.',
    side: 'top',
  },
  {
    id: 'ila-3-0',
    version: 'ILA 3.0',
    dateLabel: 'September 2026',
    keywords: 'Nested MTF & Quarterly Theory',
    content:
      'Framework makro multi-timeframe: gabungin Quarterly Theory, nested entry LTF–HTF, dan target ekspansi Standard Deviation.',
    side: 'bottom',
  },
]

/**
 * Identitas Logo ILA bersih tanpa bingkai card di atas kanvas terang.
 */
function IlaLogoMark({ periodLabel }: { periodLabel: string }) {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center px-4 text-center">
      <img
        src="/brand/logo-light-128.png"
        alt="Logo Imaginary Liquidity Algorithm"
        draggable={false}
        className="h-28 w-auto object-contain lg:h-32"
        decoding="async"
      />
      <p className="mt-6 text-[17px] font-semibold tracking-tight text-[#09090b]">
        Imaginary Liquidity
      </p>
      <p className="mt-1 font-mono text-[12px] tracking-[0.14em] text-[#52525b] tabular-nums">
        {periodLabel}
      </p>
    </div>
  )
}

function DesktopMilestoneNode({
  item,
  index,
  total,
  progress,
  reducedMotion,
}: {
  item: TimelineMilestone
  index: number
  total: number
  progress: MotionValue<number>
  reducedMotion: boolean
}) {
  // Jendela aktivasi berurutan per milestone: item berikutnya tersembunyi penuh (opacity 0)
  // sampai rel progres mendekat, lalu batang -> judul -> caption muncul bertahap.
  const span = 0.84 / Math.max(1, total)
  const start = 0.04 + index * span
  const stemEnd = start + span * 0.36
  const titleStart = start + span * 0.22
  const titleEnd = start + span * 0.62
  const captionStart = start + span * 0.48
  const captionEnd = Math.min(0.96, start + span * 0.94)

  const stemScaleY = useTransform(progress, [start, stemEnd], [0, 1])
  const dotOpacity = useTransform(progress, [start, start + span * 0.22], [0, 1])
  const dotScale = useTransform(progress, [start, start + span * 0.28], [0.85, 1])

  const endDotOpacity = useTransform(progress, [start + span * 0.16, stemEnd], [0, 1])
  const endDotScale = useTransform(progress, [start + span * 0.16, stemEnd], [0.85, 1])

  const titleOpacity = useTransform(progress, [titleStart, titleEnd], [0, 1])
  const titleY = useTransform(
    progress,
    [titleStart, titleEnd],
    [item.side === 'top' ? 16 : -16, 0],
  )

  const captionOpacity = useTransform(progress, [captionStart, captionEnd], [0, 1])
  const captionY = useTransform(
    progress,
    [captionStart, captionEnd],
    [item.side === 'top' ? 14 : -14, 0],
  )

  const isTop = item.side === 'top'

  return (
    <div className="relative flex h-full w-[310px] shrink-0 flex-col lg:w-[345px]">
      {/* Paruh Atas */}
      <div className="relative flex h-1/2 flex-col justify-end pb-7 pl-6 pr-4">
        {isTop && (
          <>
            {/* Wrapper statis menahan sumbu X tepat di left-0 agar transform Framer Motion tidak menimpa translate */}
            <div
              aria-hidden="true"
              className="pointer-events-none absolute bottom-0 left-0 h-[84%] w-px -translate-x-1/2"
            >
              <motion.div
                className="h-full w-full origin-bottom bg-[#09090b]"
                style={reducedMotion ? undefined : { scaleY: stemScaleY }}
              />
            </div>

            <div
              aria-hidden="true"
              className="pointer-events-none absolute bottom-[84%] left-0 -translate-x-1/2 translate-y-1/2"
            >
              <motion.span
                className="block h-2.5 w-2.5 rounded-full border-[1.5px] border-[#09090b] bg-[#f5f5f7]"
                style={
                  reducedMotion ? undefined : { scale: endDotScale, opacity: endDotOpacity }
                }
              />
            </div>

            <div className="space-y-2">
              <motion.div
                style={reducedMotion ? undefined : { opacity: titleOpacity, y: titleY }}
                className="space-y-1.5"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="inline-flex items-center rounded-md bg-[#09090b] px-2 py-0.5 text-[11px] font-semibold tracking-tight text-white tabular-nums">
                    {item.version}
                  </span>
                  <span className="text-[12px] font-medium text-[#52525b] tabular-nums">
                    {item.dateLabel}
                  </span>
                </div>
                <h3 className="text-[17px] font-semibold leading-snug tracking-tight text-[#09090b]">
                  {item.keywords}
                </h3>
              </motion.div>

              <motion.p
                style={reducedMotion ? undefined : { opacity: captionOpacity, y: captionY }}
                className="text-[13px] leading-relaxed text-[#52525b]"
              >
                {item.content}
              </motion.p>
            </div>
          </>
        )}
      </div>

      {/* Titik persimpangan tepat di sumbu rel horizontal (wrapper statis menjaga posisi center) */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute left-0 top-1/2 z-10 -translate-x-1/2 -translate-y-1/2"
      >
        <motion.span
          className="block h-2.5 w-2.5 rounded-full bg-[#09090b]"
          style={reducedMotion ? undefined : { scale: dotScale, opacity: dotOpacity }}
        />
      </div>

      {/* Paruh Bawah */}
      <div className="relative flex h-1/2 flex-col justify-start pl-6 pr-4 pt-7">
        {!isTop && (
          <>
            <div
              aria-hidden="true"
              className="pointer-events-none absolute left-0 top-0 h-[84%] w-px -translate-x-1/2"
            >
              <motion.div
                className="h-full w-full origin-top bg-[#09090b]"
                style={reducedMotion ? undefined : { scaleY: stemScaleY }}
              />
            </div>

            <div
              aria-hidden="true"
              className="pointer-events-none absolute left-0 top-[84%] -translate-x-1/2 -translate-y-1/2"
            >
              <motion.span
                className="block h-2.5 w-2.5 rounded-full border-[1.5px] border-[#09090b] bg-[#f5f5f7]"
                style={
                  reducedMotion ? undefined : { scale: endDotScale, opacity: endDotOpacity }
                }
              />
            </div>

            <div className="space-y-2">
              <motion.div
                style={reducedMotion ? undefined : { opacity: titleOpacity, y: titleY }}
                className="space-y-1.5"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="inline-flex items-center rounded-md bg-[#09090b] px-2 py-0.5 text-[11px] font-semibold tracking-tight text-white tabular-nums">
                    {item.version}
                  </span>
                  <span className="text-[12px] font-medium text-[#52525b] tabular-nums">
                    {item.dateLabel}
                  </span>
                </div>
                <h3 className="text-[17px] font-semibold leading-snug tracking-tight text-[#09090b]">
                  {item.keywords}
                </h3>
              </motion.div>

              <motion.p
                style={reducedMotion ? undefined : { opacity: captionOpacity, y: captionY }}
                className="text-[13px] leading-relaxed text-[#52525b]"
              >
                {item.content}
              </motion.p>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

/**
 * DesktopStoryline: track horizontal sticky yang bergerak mulus mengikuti gulir
 * halaman (GPU-accelerated translateX + scaleX, tanpa layout reflow).
 */
function DesktopStoryline({
  periodLabel,
  items,
  reducedMotion,
}: {
  periodLabel: string
  items: TimelineMilestone[]
  reducedMotion: boolean
}) {
  const sectionRef = useRef<HTMLDivElement>(null)
  const { scrollYProgress } = useScroll({
    target: sectionRef,
    offset: ['start start', 'end end'],
  })

  // Track bergeser horizontal hanya sejauh yang dibutuhkan 4 milestone + logo.
  const trackX = useTransform(scrollYProgress, [0.04, 0.94], ['0%', '-40%'])
  const railScaleX = useTransform(scrollYProgress, [0.02, 0.92], [0.02, 1])

  return (
    <div ref={sectionRef} className="relative hidden h-[240vh] w-full md:block">
      <div className="sticky top-0 flex h-screen w-full flex-col justify-center overflow-hidden pt-8">
        <div className="mx-auto w-full max-w-6xl px-6">
          <motion.div
            style={reducedMotion ? undefined : { x: trackX }}
            className="flex h-[430px] w-max items-center gap-12 will-change-transform lg:h-[460px]"
          >
            {/* Identitas Logo ILA bersih tanpa card di kiri */}
            <div className="h-[320px] w-[220px] shrink-0 lg:w-[250px]">
              <IlaLogoMark periodLabel={periodLabel} />
            </div>

            {/* Area Rel + 4 Milestone Selang-seling */}
            <div className="relative flex h-full items-center pr-16">
              {/* Rel dasar tipis */}
              <div
                aria-hidden="true"
                className="absolute inset-x-0 top-1/2 h-px -translate-y-1/2 bg-zinc-300"
              />
              {/* Rel progres hitam — animasi scaleX murni GPU */}
              <motion.div
                aria-hidden="true"
                className="absolute inset-x-0 top-1/2 h-[1.5px] origin-left -translate-y-1/2 bg-[#09090b]"
                style={reducedMotion ? undefined : { scaleX: railScaleX }}
              />
              {/* Titik awal & akhir rel */}
              <div
                aria-hidden="true"
                className="absolute left-0 top-1/2 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-[#09090b]"
              />
              <div
                aria-hidden="true"
                className="absolute right-0 top-1/2 h-2 w-2 translate-x-1/2 -translate-y-1/2 rounded-full bg-zinc-400"
              />

              <div className="flex h-full items-center gap-6 pl-12">
                {items.map((item, idx) => (
                  <DesktopMilestoneNode
                    key={item.id}
                    item={item}
                    index={idx}
                    total={items.length}
                    progress={scrollYProgress}
                    reducedMotion={reducedMotion}
                  />
                ))}
              </div>
            </div>
          </motion.div>
        </div>
      </div>
    </div>
  )
}

/**
 * MobileStoryline: rel vertikal dengan garis progres hidup mengikuti gulir
 * dan reveal berurutan (titik -> judul -> caption) agar dinamis & proporsional di HP.
 */
function MobileStoryline({
  periodLabel,
  items,
  reducedMotion,
}: {
  periodLabel: string
  items: TimelineMilestone[]
  reducedMotion: boolean
}) {
  const listRef = useRef<HTMLDivElement>(null)
  const { scrollYProgress } = useScroll({
    target: listRef,
    offset: ['start 75%', 'end 60%'],
  })
  const animateInView = !reducedMotion && typeof IntersectionObserver !== 'undefined'

  return (
    <div className="mx-auto w-full max-w-6xl px-4 pb-16 sm:px-6 md:hidden">
      {/* Identitas Logo ILA bersih tanpa card */}
      <div className="flex items-center justify-center gap-3.5 border-y border-zinc-200/80 py-5">
        <img
          src="/brand/logo-light-128.png"
          alt="Logo Imaginary Liquidity Algorithm"
          draggable={false}
          className="h-11 w-auto object-contain"
          decoding="async"
        />
        <div className="text-left">
          <p className="text-[15px] font-semibold tracking-tight text-[#09090b]">
            Imaginary Liquidity
          </p>
          <p className="font-mono text-[11px] tracking-[0.12em] text-[#52525b] tabular-nums">
            {periodLabel} · ILA 1.0 → 3.0
          </p>
        </div>
      </div>

      {/* Rel vertikal + 4 milestone beranimasi bertahap */}
      <div ref={listRef} className="relative mt-8 pl-7">
        <div
          aria-hidden="true"
          className="absolute bottom-3 left-[9px] top-3 w-px bg-zinc-300"
        />
        <motion.div
          aria-hidden="true"
          className="absolute bottom-3 left-[8.5px] top-3 w-[1.5px] origin-top bg-[#09090b]"
          style={reducedMotion ? undefined : { scaleY: scrollYProgress }}
        />

        <ol className="space-y-5">
          {items.map((item) => (
            <li key={item.id} className="relative">
              <div
                aria-hidden="true"
                className="pointer-events-none absolute -left-[19px] top-5 -translate-x-1/2"
              >
                <motion.span
                  initial={animateInView ? { scale: 0.85, opacity: 0 } : false}
                  whileInView={animateInView ? { scale: 1, opacity: 1 } : undefined}
                  viewport={{ once: true, amount: 0.4 }}
                  transition={{ duration: 0.22, ease: EASE_OUT_STRONG }}
                  className="block h-2.5 w-2.5 rounded-full bg-[#09090b]"
                />
              </div>

              <div className="rounded-2xl border border-zinc-200/80 bg-white p-4 shadow-2xs">
                <motion.div
                  initial={animateInView ? { opacity: 0, y: 12 } : false}
                  whileInView={animateInView ? { opacity: 1, y: 0 } : undefined}
                  viewport={{ once: true, amount: 0.35 }}
                  transition={{ duration: 0.26, ease: EASE_OUT_STRONG }}
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="inline-flex items-center rounded-md bg-[#09090b] px-2 py-0.5 text-[11px] font-semibold tracking-tight text-white tabular-nums">
                      {item.version}
                    </span>
                    <span className="text-[12px] font-medium text-[#52525b] tabular-nums">
                      {item.dateLabel}
                    </span>
                  </div>
                  <h3 className="mt-2.5 text-[15px] font-semibold leading-snug tracking-tight text-[#09090b]">
                    {item.keywords}
                  </h3>
                </motion.div>

                <motion.p
                  initial={animateInView ? { opacity: 0, y: 10 } : false}
                  whileInView={animateInView ? { opacity: 1, y: 0 } : undefined}
                  viewport={{ once: true, amount: 0.35 }}
                  transition={{ duration: 0.26, delay: 0.08, ease: EASE_OUT_STRONG }}
                  className="mt-1.5 text-[13px] leading-relaxed text-[#52525b]"
                >
                  {item.content}
                </motion.p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    </div>
  )
}

/**
 * Timeline menyajikan storyline evolusi metode ILA (1.0 -> 3.0).
 * Desktop memakai sticky horizontal scroll; mobile memakai vertical rail
 * beranimasi agar tetap proporsional dan nyaman digulir.
 */
export default function Timeline({
  eyebrow = 'Evolusi Metodologi',
  headingPrimary = 'Dua tahun riset,',
  headingSecondary = 'satu kerangka likuiditas.',
  subtitle = 'Terus gulir — dari fondasi ERL–IRL & SMT sampai Nested Multi-Timeframe, setiap fase ILA berevolusi dari disiplin membaca likuiditas.',
  periodLabel = '2024 — 2026',
  items = DEFAULT_ILA_MILESTONES,
}: TimelineProps) {
  const prefersReducedMotion = useReducedMotion() ?? false

  return (
    <section
      id="evolusi-ila"
      aria-label="Storyline evolusi metode ILA"
      className="relative w-full border-t border-zinc-200/80 bg-[#f5f5f7] text-[#09090b]"
    >
      {/* Hero pembuka rata tengah sebelum track storyline */}
      <div className="mx-auto flex min-h-[54vh] max-w-4xl flex-col items-center justify-center gap-4 px-4 py-20 text-center sm:min-h-[64vh] sm:px-6 sm:py-28">
        <p className="font-mono text-[11px] font-medium uppercase tracking-[0.28em] text-[#52525b]">
          {eyebrow}
        </p>
        <h2 className="max-w-[18ch] text-4xl font-semibold leading-[1.08] tracking-tight sm:text-6xl">
          <span className="text-[#09090b]">{headingPrimary}</span>{' '}
          <span className="text-[#52525b]">{headingSecondary}</span>
        </h2>
        <p className="max-w-md text-[14px] leading-relaxed text-[#52525b]">{subtitle}</p>
        <ArrowDown
          aria-hidden="true"
          className={
            prefersReducedMotion
              ? 'mt-3 h-4 w-4 text-[#52525b]'
              : 'mt-3 h-4 w-4 animate-bounce text-[#52525b]'
          }
        />
      </div>

      <DesktopStoryline
        periodLabel={periodLabel}
        items={items}
        reducedMotion={prefersReducedMotion}
      />
      <MobileStoryline
        periodLabel={periodLabel}
        items={items}
        reducedMotion={prefersReducedMotion}
      />
    </section>
  )
}
