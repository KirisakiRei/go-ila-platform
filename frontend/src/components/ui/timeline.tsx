import { useRef } from 'react'
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
  headingPrimary?: string
  headingSecondary?: string
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
 * Panel identitas Logo ILA yang menemani sisi kiri track storyline.
 * Menggunakan logo tema gelap bawaan proyek agar tajam tanpa aset eksternal.
 */
function IlaBrandPanel({ periodLabel }: { periodLabel: string }) {
  return (
    <div className="relative flex h-full w-full flex-col justify-between overflow-hidden rounded-2xl border border-zinc-800/80 bg-[#050507] p-6 text-white shadow-xl shadow-black/15">
      {/* Lapisan garis grid teknikal halus, senada dengan Hero 1 */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-20"
        style={{
          backgroundImage:
            'linear-gradient(to right, rgba(255,255,255,0.08) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.08) 1px, transparent 1px)',
          backgroundSize: '36px 36px',
        }}
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-12 -top-12 h-44 w-44 rounded-full bg-white/[0.06] blur-3xl"
      />

      <div className="relative z-10 flex items-center justify-between text-[11px] font-medium uppercase tracking-[0.16em] text-zinc-400">
        <span>Evolusi Metode</span>
        <span className="tabular-nums text-zinc-300">{periodLabel}</span>
      </div>

      <div className="relative z-10 flex flex-1 flex-col items-center justify-center py-6">
        <div className="flex h-28 w-28 items-center justify-center rounded-2xl border border-white/10 bg-white/[0.03] shadow-inner">
          <img
            src="/brand/logo-dark-128.png"
            alt="Logo Imaginary Liquidity Algorithm"
            draggable={false}
            className="h-20 w-auto object-contain"
            decoding="async"
          />
        </div>
        <p className="mt-5 text-[15px] font-semibold tracking-tight text-white">
          Imaginary Liquidity
        </p>
        <p className="mt-1 text-[12px] text-zinc-400">ILA 1.0 → ILA 3.0</p>
      </div>

      <div className="relative z-10 flex items-center justify-between border-t border-white/10 pt-4 text-[11px] text-zinc-400">
        <span>4 Fase Riset</span>
        <span className="tabular-nums text-zinc-300">Okt 2024 – Sep 2026</span>
      </div>
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
  // Setiap titik mendapat jendela aktivasi sendiri sepanjang progres gulir.
  const span = 0.72 / Math.max(1, total)
  const start = 0.08 + index * span
  const end = Math.min(0.95, start + span * 1.05)

  const stemScaleY = useTransform(progress, [start, end], [0, 1])
  // Tidak pernah mulai dari scale(0): muncul dari 0.85 + opacity 0 agar terasa natural.
  const dotScale = useTransform(progress, [start, start + span * 0.5], [0.85, 1])
  const dotOpacity = useTransform(progress, [start, start + span * 0.45], [0.25, 1])
  const copyOpacity = useTransform(progress, [start + span * 0.12, end], [0.22, 1])
  const copyY = useTransform(
    progress,
    [start + span * 0.12, end],
    [item.side === 'top' ? 12 : -12, 0],
  )

  const isTop = item.side === 'top'

  return (
    <div className="relative flex h-full w-[310px] shrink-0 flex-col lg:w-[340px]">
      {/* Bagian Atas */}
      <div className="relative flex h-1/2 flex-col justify-end pb-7 pl-6 pr-4">
        {isTop && (
          <>
            <motion.div
              aria-hidden="true"
              className="absolute bottom-0 left-0 h-[86%] w-px origin-bottom bg-[#09090b]"
              style={reducedMotion ? undefined : { scaleY: stemScaleY }}
            />
            <motion.div
              aria-hidden="true"
              className="absolute bottom-[86%] left-0 h-2.5 w-2.5 -translate-x-1/2 translate-y-1/2 rounded-full border-2 border-[#09090b] bg-[#f5f5f7]"
              style={
                reducedMotion
                  ? undefined
                  : { scale: dotScale, opacity: dotOpacity }
              }
            />
            <motion.div
              style={
                reducedMotion
                  ? undefined
                  : { opacity: copyOpacity, y: copyY }
              }
              className="space-y-2"
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
              <p className="text-[13px] leading-relaxed text-[#52525b]">{item.content}</p>
            </motion.div>
          </>
        )}
      </div>

      {/* Titik persimpangan pada rel horizontal */}
      <motion.div
        aria-hidden="true"
        className="absolute left-0 top-1/2 z-10 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-[#09090b] bg-[#f5f5f7]"
        style={
          reducedMotion
            ? undefined
            : { scale: dotScale, opacity: dotOpacity }
        }
      />

      {/* Bagian Bawah */}
      <div className="relative flex h-1/2 flex-col justify-start pl-6 pr-4 pt-7">
        {!isTop && (
          <>
            <motion.div
              aria-hidden="true"
              className="absolute left-0 top-0 h-[86%] w-px origin-top bg-[#09090b]"
              style={reducedMotion ? undefined : { scaleY: stemScaleY }}
            />
            <motion.div
              aria-hidden="true"
              className="absolute left-0 top-[86%] h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-[#09090b] bg-[#f5f5f7]"
              style={
                reducedMotion
                  ? undefined
                  : { scale: dotScale, opacity: dotOpacity }
              }
            />
            <motion.div
              style={
                reducedMotion
                  ? undefined
                  : { opacity: copyOpacity, y: copyY }
              }
              className="space-y-2"
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
              <p className="text-[13px] leading-relaxed text-[#52525b]">{item.content}</p>
            </motion.div>
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
  headingPrimary,
  headingSecondary,
  periodLabel,
  items,
  reducedMotion,
}: Required<TimelineProps> & { reducedMotion: boolean }) {
  const sectionRef = useRef<HTMLDivElement>(null)
  const { scrollYProgress } = useScroll({
    target: sectionRef,
    offset: ['start start', 'end end'],
  })

  // Track bergeser horizontal hanya sejauh yang dibutuhkan 4 milestone + panel logo.
  const trackX = useTransform(scrollYProgress, [0.04, 0.94], ['0%', '-42%'])
  const railScaleX = useTransform(scrollYProgress, [0.06, 0.92], [0.08, 1])

  return (
    <div ref={sectionRef} className="relative hidden h-[230vh] w-full md:block">
      <div className="sticky top-0 flex h-screen w-full flex-col justify-center overflow-hidden pt-12">
        <div className="mx-auto mb-8 flex w-full max-w-6xl items-end justify-between gap-6 px-6">
          <div className="max-w-2xl">
            <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#52525b]">
              Storyline Metodologi
            </p>
            <h2 className="mt-2 text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
              <span className="text-[#09090b]">{headingPrimary}</span>{' '}
              <span className="text-[#52525b]">{headingSecondary}</span>
            </h2>
          </div>
          <p className="shrink-0 text-[13px] font-medium text-[#52525b] tabular-nums">
            {periodLabel}
          </p>
        </div>

        {/* Viewport track horizontal */}
        <div className="mx-auto w-full max-w-6xl px-6">
          <motion.div
            style={reducedMotion ? undefined : { x: trackX }}
            className="flex h-[430px] w-max items-center gap-12 will-change-transform lg:h-[460px]"
          >
            {/* Panel Logo ILA di kiri */}
            <div className="h-[360px] w-[280px] shrink-0 lg:h-[390px] lg:w-[310px]">
              <IlaBrandPanel periodLabel={periodLabel} />
            </div>

            {/* Area Rel + 4 Milestone Selang-seling */}
            <div className="relative flex h-full items-center pr-16">
              {/* Rel dasar abu-abu */}
              <div
                aria-hidden="true"
                className="absolute inset-x-0 top-1/2 h-px -translate-y-1/2 bg-zinc-300"
              />
              {/* Rel progres hitam — animasi scaleX murni GPU */}
              <motion.div
                aria-hidden="true"
                className="absolute inset-x-0 top-1/2 h-[2px] origin-left -translate-y-1/2 bg-[#09090b]"
                style={reducedMotion ? undefined : { scaleX: railScaleX }}
              />
              {/* Titik awal & akhir rel */}
              <div
                aria-hidden="true"
                className="absolute left-0 top-1/2 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-[#09090b]"
              />
              <div
                aria-hidden="true"
                className="absolute right-0 top-1/2 h-2.5 w-2.5 translate-x-1/2 -translate-y-1/2 rounded-full bg-zinc-400"
              />

              <div className="flex h-full items-center gap-4 pl-8">
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
 * dan kartu milestone bertahap agar tetap dinamis namun proporsional di layar HP.
 */
function MobileStoryline({
  headingPrimary,
  headingSecondary,
  periodLabel,
  items,
  reducedMotion,
}: Required<TimelineProps> & { reducedMotion: boolean }) {
  const listRef = useRef<HTMLDivElement>(null)
  const { scrollYProgress } = useScroll({
    target: listRef,
    offset: ['start 75%', 'end 60%'],
  })
  const animateInView = !reducedMotion && typeof IntersectionObserver !== 'undefined'

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-14 sm:px-6 md:hidden">
      <div className="space-y-2">
        <div className="flex items-center justify-between gap-2">
          <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[#52525b]">
            Storyline Metodologi
          </p>
          <span className="text-[12px] font-medium text-[#52525b] tabular-nums">{periodLabel}</span>
        </div>
        <h2 className="text-2xl font-semibold leading-[1.15] tracking-tight sm:text-3xl">
          <span className="text-[#09090b]">{headingPrimary}</span>{' '}
          <span className="text-[#52525b]">{headingSecondary}</span>
        </h2>
      </div>

      {/* Kartu identitas ringkas dengan Logo ILA */}
      <div className="relative mt-6 flex items-center gap-4 overflow-hidden rounded-2xl border border-zinc-800/80 bg-[#050507] p-4 text-white">
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 opacity-20"
          style={{
            backgroundImage:
              'linear-gradient(to right, rgba(255,255,255,0.08) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.08) 1px, transparent 1px)',
            backgroundSize: '28px 28px',
          }}
        />
        <div className="relative z-10 flex h-14 w-14 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/[0.04]">
          <img
            src="/brand/logo-dark-128.png"
            alt="Logo Imaginary Liquidity Algorithm"
            draggable={false}
            className="h-10 w-auto object-contain"
            decoding="async"
          />
        </div>
        <div className="relative z-10 min-w-0">
          <p className="text-[14px] font-semibold tracking-tight text-white">
            Imaginary Liquidity
          </p>
          <p className="mt-0.5 text-[12px] text-zinc-400">
            4 fase riset likuiditas · ILA 1.0 → ILA 3.0
          </p>
        </div>
      </div>

      {/* Rel vertikal + 4 milestone beranimasi */}
      <div ref={listRef} className="relative mt-8 pl-7">
        {/* Rel dasar */}
        <div
          aria-hidden="true"
          className="absolute bottom-3 left-[9px] top-3 w-px bg-zinc-300"
        />
        {/* Rel progres hitam mengikuti gulir */}
        <motion.div
          aria-hidden="true"
          className="absolute bottom-3 left-[8.5px] top-3 w-[2px] origin-top bg-[#09090b]"
          style={reducedMotion ? undefined : { scaleY: scrollYProgress }}
        />

        <ol className="space-y-4">
          {items.map((item, idx) => (
            <motion.li
              key={item.id}
              initial={animateInView ? { opacity: 0, y: 16 } : false}
              whileInView={animateInView ? { opacity: 1, y: 0 } : undefined}
              viewport={{ once: true, amount: 0.3 }}
              transition={{
                duration: 0.28,
                delay: idx * 0.04,
                ease: EASE_OUT_STRONG,
              }}
              className="relative"
            >
              <motion.span
                aria-hidden="true"
                initial={animateInView ? { scale: 0.85, opacity: 0 } : false}
                whileInView={animateInView ? { scale: 1, opacity: 1 } : undefined}
                viewport={{ once: true, amount: 0.3 }}
                transition={{ duration: 0.24, ease: EASE_OUT_STRONG }}
                className="absolute -left-[24px] top-5 h-3 w-3 rounded-full border-2 border-[#09090b] bg-[#f5f5f7]"
              />
              <div className="rounded-2xl border border-zinc-200/80 bg-white p-4 shadow-2xs">
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
                <p className="mt-1.5 text-[13px] leading-relaxed text-[#52525b]">
                  {item.content}
                </p>
              </div>
            </motion.li>
          ))}
        </ol>
      </div>
    </div>
  )
}

/**
 * Timeline menyajikan storyline evolusi metode ILA (1.0 → 3.0).
 * Desktop memakai sticky horizontal scroll; mobile memakai vertical rail
 * beranimasi agar tetap proporsional dan nyaman digulir.
 */
export default function Timeline({
  headingPrimary = 'Dua tahun riset,',
  headingSecondary = 'satu kerangka likuiditas.',
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
      <DesktopStoryline
        headingPrimary={headingPrimary}
        headingSecondary={headingSecondary}
        periodLabel={periodLabel}
        items={items}
        reducedMotion={prefersReducedMotion}
      />
      <MobileStoryline
        headingPrimary={headingPrimary}
        headingSecondary={headingSecondary}
        periodLabel={periodLabel}
        items={items}
        reducedMotion={prefersReducedMotion}
      />
    </section>
  )
}
