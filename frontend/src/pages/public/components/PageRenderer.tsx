import {
  ArrowUpRight,
  Bell,
  BookOpen,
  Clock,
  CreditCard,
  Gauge,
  HelpCircle,
  History,
  Layers,
  Plug,
  ShieldCheck,
  Sparkles,
  Zap,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { PageContent, PageItem, PageSection, PageSectionKind } from '@/types/system'
import { MarkdownView } from '../legal/MarkdownView'
import { ActionLink, PublicHero, SectionHeading } from './PublicSections'

/**
 * Ikon bagian halaman dipilih dari daftar tertutup.
 *
 * Admin hanya mengetik nama ikon, bukan markup, sehingga isi halaman tidak
 * pernah bisa menyuntikkan elemen tak terduga ke dalam tata letak.
 */
const ICONS: Record<string, LucideIcon> = {
  layers: Layers,
  shield: ShieldCheck,
  bell: Bell,
  zap: Zap,
  gauge: Gauge,
  history: History,
  book: BookOpen,
  plug: Plug,
  card: CreditCard,
  clock: Clock,
  sparkles: Sparkles,
}

function IconFor({ name, className }: { name: string; className?: string }) {
  const Icon = ICONS[name] ?? Layers
  return <Icon className={className} aria-hidden="true" />
}

/**
 * PageRenderer merender satu halaman publik dari isi yang dikirim server.
 *
 * Seluruh kalimat berasal dari basis data, sedangkan bentuk setiap bagian
 * ditentukan di sini. Dengan pembagian itu, admin dapat menulis ulang isi
 * tanpa bisa merusak tata letak, dan pengunjung selalu melihat versi terakhir
 * yang tersimpan.
 */
export function PageRenderer({ page }: { page: PageContent }) {
  const [heroSection, ...rest] = page.sections

  return (
    <div className="bg-[#f5f5f7]">
      <PublicHero
        eyebrow={page.eyebrow}
        title={page.title}
        description={page.lead}
        actions={
          <>
            <ActionLink href="/app/register">Mulai Trial 7 Hari</ActionLink>
            <ActionLink href="/harga" variant="secondary">
              Lihat harga
            </ActionLink>
          </>
        }
        aside={<HeroAside page={page} />}
      />

      <div className="mx-auto w-full max-w-6xl px-4 pb-20 sm:px-6">
        {heroSection && (
          <div className="-mt-6 sm:-mt-8">
            <SectionBlock section={heroSection} index={1} first />
          </div>
        )}
        {rest.map((section, index) => (
          <SectionBlock key={section.key} section={section} index={index + 2} />
        ))}
      </div>

      <PageFooterNote />
    </div>
  )
}

/** HeroAside meringkas isi halaman: jumlah bagian dan waktu pembaruan. */
function HeroAside({ page }: { page: PageContent }) {
  const headings = page.sections.filter((section) => section.title).slice(0, 4)

  if (headings.length === 0) return null

  return (
    <nav
      aria-label="Isi halaman"
      className="rounded-2xl border border-zinc-200 bg-white/80 p-5 backdrop-blur-sm sm:p-6"
    >
      <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[#a1a1aa]">
        Di halaman ini
      </p>
      <ol className="mt-3 space-y-2">
        {headings.map((section, index) => (
          <li key={section.key}>
            <a
              href={`#${section.key}`}
              className="group flex items-baseline gap-3 rounded-md py-1 text-[13.5px] text-[#3f3f46] transition-colors hover:text-[#09090b]"
            >
              <span className="tabular text-[11px] text-[#a1a1aa]">
                {String(index + 1).padStart(2, '0')}
              </span>
              <span className="underline-offset-4 group-hover:underline">{section.title}</span>
            </a>
          </li>
        ))}
      </ol>
    </nav>
  )
}

const SECTION_TONE: Record<PageSectionKind, 'light' | 'quiet' | 'dark'> = {
  prose: 'light',
  cards: 'quiet',
  steps: 'light',
  faq: 'quiet',
  table: 'light',
  callout: 'dark',
  features: 'quiet',
  stats: 'light',
}

/** SectionBlock memilih bentuk render sesuai jenis bagian. */
function SectionBlock({
  section,
  index,
  first,
}: {
  section: PageSection
  index: number
  first?: boolean
}) {
  const tone = SECTION_TONE[section.kind] ?? 'light'
  const label = String(index).padStart(2, '0')

  return (
    <section
      id={section.key}
      className={cn(
        'scroll-mt-24 rounded-2xl border p-5 sm:p-8',
        !first && 'mt-10 sm:mt-12',
        tone === 'dark' && 'border-[#09090b] bg-[#09090b] text-zinc-300',
        tone === 'quiet' && 'border-zinc-200 bg-white',
        tone === 'light' && 'border-zinc-200 bg-white',
      )}
    >
      {section.title && (
        <SectionHeading
          index={label}
          title={section.title}
          description={section.lead || undefined}
          tone={tone === 'dark' ? 'dark' : 'light'}
        />
      )}
      {!section.title && section.lead && (
        <p className="max-w-2xl text-[14px] leading-relaxed text-[#52525b]">{section.lead}</p>
      )}

      <div className={cn(section.title || section.lead ? 'mt-6' : '')}>
        <SectionBody section={section} tone={tone} />
      </div>
    </section>
  )
}

function SectionBody({
  section,
  tone,
}: {
  section: PageSection
  tone: 'light' | 'quiet' | 'dark'
}) {
  const isDark = tone === 'dark'

  switch (section.kind) {
    case 'prose':
      return (
        <div className={cn(isDark && 'prose-invert')}>
          <MarkdownView source={section.body} className={isDark ? 'text-zinc-300' : undefined} />
        </div>
      )

    case 'callout':
      return <CalloutBlock section={section} />

    case 'stats':
      return (
        <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {section.items.map((item) => (
            <div key={item.label} className="rounded-xl border border-zinc-200 bg-[#fafafa] p-4">
              <dt className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[#a1a1aa]">
                {item.label}
              </dt>
              <dd className="mt-2 text-[13.5px] leading-relaxed text-[#3f3f46]">{item.value}</dd>
            </div>
          ))}
        </dl>
      )

    case 'steps':
      return (
        <ol className="space-y-3">
          {section.items.map((item, index) => (
            <li
              key={`${section.key}-${item.label}`}
              className="flex gap-4 rounded-xl border border-zinc-200 bg-[#fafafa] p-4 sm:p-5"
            >
              <span className="tabular shrink-0 text-[12px] font-semibold text-[#a1a1aa]">
                {String(index + 1).padStart(2, '0')}
              </span>
              <div className="min-w-0">
                <p className="text-[14px] font-semibold text-[#09090b]">{item.label}</p>
                <p className="mt-1.5 text-[13.5px] leading-relaxed text-[#52525b]">{item.value}</p>
              </div>
            </li>
          ))}
        </ol>
      )

    case 'cards':
      return (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {section.items.map((item) => (
            <ItemCard key={`${section.key}-${item.label}`} item={item} />
          ))}
        </div>
      )

    case 'features':
      return (
        <div className="grid gap-3 sm:grid-cols-2">
          {section.items.map((item) => (
            <ItemCard key={`${section.key}-${item.label}`} item={item} showNote />
          ))}
        </div>
      )

    case 'table':
      return <ItemTable items={section.items} />

    case 'faq':
      return (
        <div className="divide-y divide-zinc-200 border-y border-zinc-200">
          {section.items.map((item) => (
            <details key={`${section.key}-${item.label}`} className="group py-4">
              <summary className="flex cursor-pointer list-none items-start justify-between gap-4 text-[14px] font-medium text-[#09090b] marker:content-none">
                <span>{item.label}</span>
                <span
                  aria-hidden="true"
                  className="mt-0.5 shrink-0 text-[18px] leading-none text-[#a1a1aa] transition-transform group-open:rotate-45"
                >
                  +
                </span>
              </summary>
              <p className="mt-2 max-w-2xl text-[13.5px] leading-relaxed text-[#52525b]">
                {item.value}
              </p>
            </details>
          ))}
        </div>
      )

    default:
      return null
  }
}

function CalloutBlock({ section }: { section: PageSection }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/[0.03] p-5 sm:p-6">
      {/* Judul bagian sudah tampil di atas; label di sini hanya ada bila bagian
          memang tidak punya judul, agar kalimatnya tidak terbaca dua kali. */}
      {!section.title && section.eyebrow && (
        <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">
          {section.eyebrow}
        </p>
      )}
      <p className="whitespace-pre-line text-[13.5px] leading-relaxed text-zinc-300">
        {section.body}
      </p>
    </div>
  )
}

function ItemCard({ item, showNote }: { item: PageItem; showNote?: boolean }) {
  const content = (
    <>
      <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-zinc-200 bg-white">
        <IconFor name={item.icon} className="h-4 w-4 text-[#3f3f46]" />
      </span>
      <p className="mt-3 flex items-center gap-1 text-[14px] font-semibold text-[#09090b]">
        {item.label}
        {item.href && <ArrowUpRight className="h-3.5 w-3.5 text-[#a1a1aa]" aria-hidden="true" />}
      </p>
      {item.value && (
        <p className="mt-1.5 text-[13.5px] leading-relaxed text-[#52525b]">{item.value}</p>
      )}
      {showNote && item.note && (
        <p className="mt-2.5 border-t border-zinc-200 pt-2.5 text-[12.5px] leading-relaxed text-[#71717a]">
          {item.note}
        </p>
      )}
    </>
  )

  const className =
    'block rounded-xl border border-zinc-200 bg-white p-4 transition-colors sm:p-5' +
    (item.href ? ' hover:border-zinc-300 hover:bg-[#fafafa]' : '')

  if (item.href) {
    return (
      <a href={item.href} className={className}>
        {content}
      </a>
    )
  }

  return <div className={className}>{content}</div>
}

/**
 * ItemTable merender butir sebagai tabel dua kolom.
 *
 * Butir dengan `note` tampil sebagai baris kedua di kolom kanan, sehingga isi
 * tetap terbaca di layar sempit tanpa tabel yang meluber.
 */
function ItemTable({ items }: { items: PageItem[] }) {
  if (items.length === 0) return null

  return (
    <div className="overflow-hidden rounded-xl border border-zinc-200">
      <table className="w-full border-collapse text-left text-[13.5px]">
        <tbody>
          {items.map((item) => (
            <tr key={item.label} className="align-top even:bg-[#fafafa]">
              <th
                scope="row"
                className="w-2/5 border-b border-zinc-100 px-4 py-3 text-[12.5px] font-semibold text-[#09090b]"
              >
                {item.label}
              </th>
              <td className="border-b border-zinc-100 px-4 py-3 text-[#52525b]">
                <span className="block">{item.value}</span>
                {item.note && (
                  <span className="mt-1 block text-[12px] text-[#71717a]">{item.note}</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/** PageFooterNote menutup halaman dengan jalur bantuan, bukan ajakan membeli. */
function PageFooterNote() {
  return (
    <div className="border-t border-zinc-200 bg-white">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-3 px-4 py-8 sm:flex-row sm:items-center sm:justify-between sm:px-6">
        <p className="flex items-center gap-2 text-[13px] text-[#52525b]">
          <HelpCircle className="h-4 w-4 text-[#a1a1aa]" aria-hidden="true" />
          Ada yang belum jelas? Tim dukungan membalas pada hari kerja.
        </p>
        <a
          href="/bantuan"
          className="text-[13px] font-medium text-[#09090b] underline underline-offset-4"
        >
          Pusat Bantuan
        </a>
      </div>
    </div>
  )
}

/** PageEmpty dipakai saat isi halaman belum disiapkan admin. */
export function PageEmpty({ title }: { title: string }) {
  return (
    <div className="bg-[#f5f5f7]">
      <div className="mx-auto w-full max-w-3xl px-4 py-24 text-center sm:px-6">
        <h1 className="text-[26px] font-semibold tracking-tight text-[#09090b] sm:text-[32px]">
          {title}
        </h1>
        <p className="mx-auto mt-3 max-w-md text-[14px] leading-relaxed text-[#52525b]">
          Isi halaman ini sedang disiapkan. Sementara itu, Anda dapat memulai dari pusat bantuan
          atau menghubungi tim dukungan.
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <ActionLink href="/bantuan">Pusat Bantuan</ActionLink>
          <ActionLink href="/" variant="secondary">
            Kembali ke beranda
          </ActionLink>
        </div>
      </div>
    </div>
  )
}
