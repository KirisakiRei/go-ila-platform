import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useLocation } from 'react-router-dom'
import { ArrowUpRight, FileText, ShieldAlert } from 'lucide-react'
import { request } from '@/lib/http'
import { cn } from '@/lib/utils'
import { MarkdownView } from './MarkdownView'
import { outlineFromBlocks, parseMarkdown } from './markdown-parse'
import { useScrollSpy } from './use-scroll-spy'
import { usePageContent } from '../lib/use-page-content'
import {
  LEGAL_DOCUMENTS,
  LEGAL_GROUPS,
  LEGAL_ORDER,
  isLegalSlug,
  type LegalSlug,
} from './documents'

type LegalDocResponse = {
  slug: string
  version: number
  title: string
  body_md?: string
  require_reaccept: boolean
  published: boolean
  effective_at?: string
}

/** LegalListing adalah daftar dokumen yang sedang berlaku di server. */
type LegalListing = Pick<LegalDocResponse, 'slug' | 'version' | 'title' | 'effective_at'>[]

/** formatEffective mengubah tanggal berlaku menjadi bentuk yang dibaca manusia. */
function formatEffective(value?: string): string | null {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return new Intl.DateTimeFormat('id-ID', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(date)
}

/**
 * LegalCenterPage adalah portal dokumen legal: hub, tab dokumen, daftar isi
 * yang mengikuti posisi gulir, dan area baca dengan lebar yang nyaman.
 *
 * Isi diambil dari server karena dokumen legal wajib berversi dan persetujuannya
 * dicatat per versi (plans/09). Ketika server belum menerbitkan dokumen, halaman
 * memakai teks bawaan agar pengunjung tetap mendapat penjelasan yang utuh dan
 * tidak melihat halaman kosong.
 */
export default function LegalCenterPage() {
  const { pathname } = useLocation()
  const slug = pathname.split('/')[2]
  const active: LegalSlug | null = isLegalSlug(slug) ? slug : null

  if (!active) return <LegalHubPage />
  return <LegalDocumentPage slug={active} />
}

/** Breadcrumb dipakai hub dan dokumen agar posisi pengguna selalu jelas. */
function Breadcrumb({ current }: { current?: string }) {
  return (
    <nav aria-label="Jejak navigasi" className="text-[12px] text-[#71717a]">
      <ol className="flex flex-wrap items-center gap-1.5">
        <li>
          <Link to="/" className="transition-colors hover:text-[#09090b]">
            Beranda
          </Link>
        </li>
        <li aria-hidden="true">/</li>
        <li>
          {current ? (
            <Link to="/legal" className="transition-colors hover:text-[#09090b]">
              Legal Center
            </Link>
          ) : (
            <span className="text-[#3f3f46]">Legal Center</span>
          )}
        </li>
        {current && (
          <>
            <li aria-hidden="true">/</li>
            <li className="text-[#3f3f46]">{current}</li>
          </>
        )}
      </ol>
    </nav>
  )
}

/** LegalHubPage menampilkan seluruh dokumen sebagai satu portal terpadu. */
function LegalHubPage() {
  const page = usePageContent('legal')
  const heading = page.data?.sections.find((section) => section.kind === 'table')

  // Ringkasan kartu diambil dari CMS; bila admin belum mengisi, kartu memakai
  // judul dokumen saja agar tidak menampilkan kalimat lama yang tidak berlaku.
  const listing = useQuery({
    queryKey: ['legal', 'list'],
    queryFn: () => request<LegalListing>('/legal'),
    retry: 0,
    staleTime: 15 * 60_000,
  })

  const summaryBySlug = useMemo(() => {
    const map = new Map<string, string>()
    for (const item of heading?.items ?? []) {
      if (item.href) map.set(item.href, item.value)
    }
    return map
  }, [heading])

  const titleBySlug = useMemo(() => {
    const map = new Map<string, string>()
    for (const item of listing.data ?? []) {
      if (item.title) map.set(`/legal/${item.slug}`, item.title)
    }
    return map
  }, [listing.data])

  return (
    <div className="bg-[#f5f5f7] text-[#09090b]">
      <div className="mx-auto w-full max-w-6xl px-4 pt-10 pb-20 sm:px-6 sm:pt-14 sm:pb-28">
        <Breadcrumb />

        <header className="mt-8 max-w-2xl">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[#71717a]">
            {page.data?.eyebrow || 'Legal Center'}
          </p>
          <h1 className="mt-3 text-3xl font-semibold leading-[1.1] tracking-tight sm:text-[2.6rem]">
            {page.data?.title || 'Ketentuan yang berlaku, ditulis untuk dibaca.'}
          </h1>
          <p className="mt-4 text-[15px] leading-relaxed text-[#52525b]">
            {page.data?.lead ||
              'Dokumen di bawah ini mengatur hubungan Anda dengan Imaginary Liquidity. Semuanya terbuka tanpa perlu masuk, dan setiap versi baru diterbitkan dengan tanggal berlaku yang jelas.'}
          </p>
        </header>

        <div className="mt-14 space-y-12">
          {LEGAL_GROUPS.map((group) => {
            const items = LEGAL_ORDER.map((key) => LEGAL_DOCUMENTS[key]).filter(
              (doc) => doc.group === group.key,
            )

            return (
              <section key={group.key} aria-labelledby={`grup-${group.key}`}>
                <div className="flex flex-col gap-1 border-t border-zinc-300/70 pt-6 sm:flex-row sm:items-end sm:justify-between">
                  <h2
                    id={`grup-${group.key}`}
                    className="text-[13px] font-semibold uppercase tracking-[0.16em] text-[#09090b]"
                  >
                    {group.label}
                  </h2>
                  <p className="max-w-md text-[13px] leading-relaxed text-[#52525b]">
                    {group.note}
                  </p>
                </div>

                <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {items.map((doc) => (
                    <Link
                      key={doc.slug}
                      to={`/legal/${doc.slug}`}
                      className="group flex h-full flex-col justify-between rounded-2xl border border-zinc-200 bg-white p-6 transition-colors hover:border-zinc-400"
                    >
                      <div>
                        <div className="flex items-start justify-between gap-3">
                          <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-200 bg-[#f4f4f5] text-[#3f3f46]">
                            {doc.slug === 'risk' ? (
                              <ShieldAlert className="h-4 w-4" aria-hidden="true" />
                            ) : (
                              <FileText className="h-4 w-4" aria-hidden="true" />
                            )}
                          </span>
                          {doc.featured && (
                            <span className="rounded-full border border-zinc-300 px-2 py-0.5 text-[10px] font-medium uppercase tracking-[0.1em] text-[#52525b]">
                              Utama
                            </span>
                          )}
                        </div>
                        <h3 className="mt-4 text-[16px] font-semibold tracking-tight">
                          {titleBySlug.get(`/legal/${doc.slug}`) ?? doc.title}
                        </h3>
                        <p className="mt-2 text-[13px] leading-relaxed text-[#52525b]">
                          {summaryBySlug.get(`/legal/${doc.slug}`) ?? doc.summary}
                        </p>
                      </div>
                      <span className="mt-6 inline-flex items-center gap-1.5 text-[12px] font-medium text-[#3f3f46]">
                        Baca dokumen
                        <ArrowUpRight
                          className="h-3.5 w-3.5 transition-transform group-hover:-translate-y-0.5 group-hover:translate-x-0.5"
                          aria-hidden="true"
                        />
                      </span>
                    </Link>
                  ))}
                </div>
              </section>
            )
          })}
        </div>

        <div className="mt-16 rounded-2xl border border-zinc-200 bg-white p-6 sm:p-8">
          <h2 className="text-[15px] font-semibold tracking-tight">Ada yang perlu ditanyakan?</h2>
          <p className="mt-2 max-w-2xl text-[13px] leading-relaxed text-[#52525b]">
            Jika ada bagian yang belum jelas, sampaikan melalui dukungan. Untuk pertanyaan yang
            sering muncul, pusat bantuan menjawab banyak hal lebih cepat.
          </p>
          <div className="mt-5 flex flex-wrap gap-3">
            <Link
              to="/bantuan"
              className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-4 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-zinc-800"
            >
              Pusat Bantuan
            </Link>
            <a
              href="mailto:dukungan@rekavia.com"
              className="inline-flex items-center justify-center rounded-lg border border-zinc-300 px-4 py-2.5 text-[13px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100"
            >
              dukungan@rekavia.com
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}

/**
 * LegalDocumentPage menyusun dokumen dalam dua kolom: daftar isi yang menempel
 * di kiri dan area baca di kanan. Di layar kecil daftar isi berubah menjadi
 * deretan tautan di atas dokumen.
 */
function LegalDocumentPage({ slug }: { slug: LegalSlug }) {
  const fallback = LEGAL_DOCUMENTS[slug]

  // Dokumen publik boleh dibaca tanpa sesi (plans/09 §10). Kegagalan diambil
  // sebagai sinyal untuk memakai teks bawaan, bukan sebagai galat halaman.
  const remote = useQuery({
    queryKey: ['legal', 'doc', slug],
    queryFn: () => request<LegalDocResponse>(`/legal/${slug}`),
    retry: 0,
    staleTime: 15 * 60_000,
  })

  const remoteBody = remote.data?.published ? remote.data.body_md : undefined
  const body = remoteBody && remoteBody.trim() !== '' ? remoteBody : fallback.body
  const title = remote.data?.title || fallback.title
  const version = remote.data?.version

  const blocks = useMemo(() => parseMarkdown(body), [body])
  const outline = useMemo(() => outlineFromBlocks(blocks), [blocks])
  const activeId = useScrollSpy(
    outline.map((item) => item.id),
    outline.length > 0,
  )

  return (
    <div className="bg-[#f5f5f7] text-[#09090b]">
      <div className="mx-auto w-full max-w-6xl px-4 pt-10 pb-20 sm:px-6 sm:pt-14 sm:pb-28">
        <Breadcrumb current={title} />

        {/* Tab sub-navigasi: seluruh dokumen legal dalam satu portal terpadu. */}
        <nav aria-label="Dokumen legal" className="mt-6 -mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
          <div className="flex w-max min-w-full gap-1 border-b border-zinc-300/70">
            {LEGAL_ORDER.map((key) => {
              const doc = LEGAL_DOCUMENTS[key]
              const isActive = key === slug
              return (
                <Link
                  key={key}
                  to={`/legal/${key}`}
                  aria-current={isActive ? 'page' : undefined}
                  className={cn(
                    'relative whitespace-nowrap px-3 py-2.5 text-[13px] transition-colors',
                    isActive
                      ? 'font-semibold text-[#09090b]'
                      : 'text-[#71717a] hover:text-[#09090b]',
                  )}
                >
                  {doc.title}
                  <span
                    aria-hidden="true"
                    className={cn(
                      'absolute inset-x-2 -bottom-px h-0.5 rounded-full transition-opacity',
                      isActive ? 'bg-[#09090b] opacity-100' : 'opacity-0',
                    )}
                  />
                </Link>
              )
            })}
          </div>
        </nav>

        <div className="mt-10 lg:grid lg:grid-cols-12 lg:gap-12">
          {/* Daftar isi: menempel di kiri pada layar lebar. */}
          <aside className="hidden lg:col-span-3 lg:block">
            <div className="sticky top-24">
              <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[#71717a]">
                Daftar isi
              </p>
              <nav aria-label="Daftar isi dokumen" className="mt-4">
                <ul className="space-y-0.5 border-l border-zinc-300/80">
                  {outline.map((item) => {
                    const isActive = item.id === activeId
                    return (
                      <li key={item.id}>
                        <a
                          href={`#${item.id}`}
                          aria-current={isActive ? 'true' : undefined}
                          className={cn(
                            '-ml-px block border-l-2 py-1.5 pl-3.5 text-[13px] leading-snug transition-colors',
                            isActive
                              ? 'border-[#09090b] font-medium text-[#09090b]'
                              : 'border-transparent text-[#71717a] hover:border-zinc-400 hover:text-[#3f3f46]',
                          )}
                        >
                          {item.label}
                        </a>
                      </li>
                    )
                  })}
                </ul>
              </nav>
            </div>
          </aside>

          {/* Area baca. */}
          <div className="lg:col-span-9">
            <div className="rounded-2xl border border-zinc-200 bg-white px-5 py-8 sm:px-10 sm:py-10">
              <header className="border-b border-zinc-200 pb-6">
                <h1 className="text-[26px] font-semibold leading-tight tracking-tight sm:text-[32px]">
                  {title}
                </h1>
                <p className="mt-3 text-[13px] leading-relaxed text-[#52525b]">
                  {fallback.summary}
                </p>
                <dl className="mt-5 flex flex-wrap items-center gap-x-6 gap-y-2 text-[12px] text-[#71717a]">
                  <div className="flex items-center gap-1.5">
                    <dt className="font-medium text-[#3f3f46]">Versi</dt>
                    <dd className="tabular">{version ?? 1}</dd>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <dt className="font-medium text-[#3f3f46]">Berlaku sejak</dt>
                    <dd>
                      {formatEffective(remote.data?.effective_at) ?? 'Mengikuti versi terbit'}
                    </dd>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <dt className="font-medium text-[#3f3f46]">Grup</dt>
                    <dd>{fallback.group}</dd>
                  </div>
                </dl>
              </header>

              {/* Di layar kecil, daftar isi tampil sebagai boks ringkas di atas teks. */}
              {outline.length > 0 && (
                <details className="mt-6 rounded-xl border border-zinc-200 bg-[#fafafa] px-4 py-3 lg:hidden">
                  <summary className="cursor-pointer text-[13px] font-semibold text-[#09090b]">
                    Daftar isi dokumen
                  </summary>
                  <ol className="mt-3 space-y-1.5 text-[13px] text-[#52525b]">
                    {outline.map((item) => (
                      <li key={item.id}>
                        <a href={`#${item.id}`} className="hover:text-[#09090b]">
                          {item.label}
                        </a>
                      </li>
                    ))}
                  </ol>
                </details>
              )}

              <MarkdownView source={body} className="mt-8" />
            </div>

            <div className="mt-6 flex flex-col gap-3 rounded-2xl border border-zinc-200 bg-white p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6">
              <p className="text-[13px] leading-relaxed text-[#52525b]">
                Butuh bantuan memahami bagian tertentu? Hubungi dukungan dan sebutkan nomor pasal.
              </p>
              <div className="flex flex-wrap gap-2">
                <Link
                  to="/legal"
                  className="inline-flex items-center justify-center rounded-lg border border-zinc-300 px-3.5 py-2 text-[12px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100"
                >
                  Semua dokumen
                </Link>
                <a
                  href="mailto:dukungan@rekavia.com"
                  className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-3.5 py-2 text-[12px] font-semibold text-white transition-colors hover:bg-zinc-800"
                >
                  Hubungi dukungan
                </a>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
