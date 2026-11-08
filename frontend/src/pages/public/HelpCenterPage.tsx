import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useLocation } from 'react-router-dom'
import {
  ArrowUpRight,
  BookOpen,
  Clock,
  CreditCard,
  Layers,
  Search,
  ShieldCheck,
} from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { publicContentApi } from '@/lib/system-api'
import { cn } from '@/lib/utils'
import HelpTimezonePage from './HelpTimezonePage'
import { MarkdownView } from './legal/MarkdownView'
import { outlineFromBlocks, parseMarkdown } from './legal/markdown-parse'
import { useScrollSpy } from './legal/use-scroll-spy'

/**
 * Topik panduan tetap ditulis di kode karena menyentuh alur yang dijaga ketat
 * dan jarang berubah, sedangkan FAQ diambil dari server agar dapat diubah tanpa
 * deploy (plans/25 §1).
 */
const TOPICS = [
  {
    slug: 'memulai',
    title: 'Memulai',
    icon: BookOpen,
    summary: 'Alur dari pendaftaran sampai sinyal pertama, termasuk masa simulasi.',
    keywords: ['daftar', 'registrasi', 'mulai', 'simulasi', 'trial', 'uji coba'],
    steps: [
      {
        title: 'Buat akun',
        body: 'Daftar dengan alamat surel aktif. Persetujuan dokumen dasar diminta pada langkah ini dan dicatat beserta nomor versinya.',
      },
      {
        title: 'Kenali mode simulasi',
        body: 'Selama tujuh hari, fitur setara paket tertinggi aktif dalam mode simulasi. Tidak ada eksekusi dana nyata dan tidak ada data pembayaran yang diminta.',
      },
      {
        title: 'Tentukan aturan risiko',
        body: 'Tetapkan persentase risiko per transaksi, batas jumlah transaksi harian, dan apakah sistem berhenti setelah mencapai target harian.',
      },
      {
        title: 'Hubungkan akun broker',
        body: 'Gunakan kredensial akses dari aplikasi broker Anda. Sebelum akun nyata dihubungkan, Anda akan diminta menyetujui Disclaimer Risiko Trading.',
      },
      {
        title: 'Terima notifikasi',
        body: 'Aktifkan Telegram agar permintaan persetujuan sampai tanpa perlu membuka panel. Sinyal juga selalu tersedia di panel pelanggan.',
      },
    ],
  },
  {
    slug: 'strategi',
    title: 'Strategi & sinyal',
    icon: Layers,
    summary: 'Apa yang diukur, kapan sinyal dikirim, dan mengapa sebagian hari kosong.',
    keywords: ['ila', 'sinyal', 'strategi', 'setup', 'entry', 'no signal'],
    steps: [
      {
        title: 'Dua generasi strategi',
        body: 'ILA 2.0 menjalankan pembacaan struktur pasar dan tersedia pada seluruh paket. ILA 3.0 disiapkan sebagai proses terpisah dan dibuka bertahap untuk paket tertinggi, dengan aturan risiko yang sama.',
      },
      {
        title: 'Kapan sinyal muncul',
        body: 'Sinyal dikirim ketika kondisi yang ditetapkan terpenuhi pada instrumen yang Anda pantau. Tidak semua hari menghasilkan setup, dan itu wajar.',
      },
      {
        title: 'Mengapa sebuah sinyal tidak diteruskan',
        body: 'Kuota harian habis, batas kerugian tercapai, sesi pasar tidak aktif, atau dokumen wajib belum disetujui. Alasan penolakan tercatat dan dapat Anda periksa.',
      },
      {
        title: 'Dua jam yang berbeda peran',
        body: 'Penilaian setup mengikuti jam pasar New York, sedangkan perhitungan kuota harian mengikuti tanggal WIB. Penjelasan lengkapnya ada pada topik sesi pasar & zona waktu.',
      },
    ],
  },
  {
    slug: 'akun-broker',
    title: 'Akun broker',
    icon: ShieldCheck,
    summary: 'Cara menghubungkan akun, bagaimana kredensial disimpan, dan batasannya.',
    keywords: ['broker', 'connect', 'token', 'kredensial', 'mt5', 'aman'],
    steps: [
      {
        title: 'Dapatkan kredensial dari broker',
        body: 'Buat kredensial akses melalui aplikasi atau portal broker Anda. Gunakan kredensial khusus untuk layanan ini, bukan kata sandi akun utama Anda.',
      },
      {
        title: 'Hubungkan melalui panel pelanggan',
        body: 'Masukkan kredensial pada halaman akun. Sistem menguji koneksi sebelum akun diaktifkan untuk pemantauan.',
      },
      {
        title: 'Kredensial disimpan terenkripsi',
        body: 'Setelah tersimpan, kredensial tidak dapat ditampilkan kembali kepada siapa pun, termasuk administrator. Kredensial hanya dipakai untuk mengirim permintaan ke broker atas nama Anda.',
      },
      {
        title: 'Cabut kapan saja',
        body: 'Lepas akun dari panel bila Anda tidak lagi menggunakannya. Kredensial dihapus dan pemantauan untuk akun tersebut dihentikan.',
      },
    ],
  },
  {
    slug: 'tagihan',
    title: 'Tagihan & langganan',
    icon: CreditCard,
    summary: 'Periode tagihan, perpanjangan, kupon, dan penghentian langganan.',
    keywords: ['harga', 'bayar', 'invoice', 'refund', 'kupon', 'langganan'],
    steps: [
      {
        title: 'Periode bulanan',
        body: 'Langganan ditagihkan per periode bulanan sesuai paket yang aktif. Rincian tagihan dan status pembayaran tersedia di panel pelanggan.',
      },
      {
        title: 'Masa tenggang',
        body: 'Setelah periode berakhir, akses terbatas masih diberikan dalam waktu singkat, termasuk kemampuan memantau posisi yang sedang berjalan.',
      },
      {
        title: 'Kupon dan kredit',
        body: 'Kupon dan kredit, bila Anda memilikinya, dipotongkan pada tagihan berikutnya. Nilai akhir yang dibayarkan tercantum pada tagihan.',
      },
      {
        title: 'Pengembalian dana',
        body: 'Ketentuan pengembalian dana bergantung pada tingkat penggunaan dalam periode berjalan. Rinciannya tersedia di Kebijakan Pengembalian Dana.',
      },
    ],
  },
  {
    slug: 'keamanan',
    title: 'Keamanan akun',
    icon: ShieldCheck,
    summary: 'Autentikasi dua faktor, sesi, dan langkah bila ada akses mencurigakan.',
    keywords: ['2fa', 'keamanan', 'sesi', 'password', 'logout', 'hack'],
    steps: [
      {
        title: 'Aktifkan autentikasi dua faktor',
        body: 'Gunakan aplikasi authenticator dan simpan kode cadangan di tempat yang aman. Kode cadangan hanya ditampilkan sekali.',
      },
      {
        title: 'Kelola sesi aktif',
        body: 'Panel pelanggan menampilkan sesi yang sedang aktif. Akhiri sesi yang tidak Anda kenali, lalu ubah kata sandi.',
      },
      {
        title: 'Keluar otomatis',
        body: 'Sesi berakhir otomatis setelah tidak ada aktivitas dalam jangka waktu tertentu. Anda perlu masuk kembali untuk melanjutkan.',
      },
      {
        title: 'Bila mencurigai akses tidak sah',
        body: 'Ubah kata sandi, akhiri seluruh sesi, lalu hubungi dukungan. Sebutkan waktu kejadian agar penelusuran dapat dipersempit.',
      },
    ],
  },
]

const TOPIC_TITLES: Record<string, string> = {
  memulai: 'Memulai',
  strategi: 'Strategi & sinyal',
  'akun-broker': 'Akun broker',
  tagihan: 'Tagihan & langganan',
  keamanan: 'Keamanan akun',
  'zona-waktu': 'Sesi pasar & zona waktu',
}

/**
 * HelpPage adalah pusat bantuan publik.
 *
 * Rute: `/bantuan` (hub), `/bantuan/:topik` (panduan), dan `/bantuan/zona-waktu`
 * yang punya halaman sendiri karena paling sering ditanyakan (plans/25 §2).
 */
export default function HelpCenterPage() {
  const { pathname } = useLocation()
  const topic = pathname.split('/')[2]

  if (topic === 'zona-waktu') return <HelpTimezonePage />

  const guide = TOPICS.find((item) => item.slug === topic)
  if (guide) return <HelpTopicPage slug={guide.slug} />
  if (topic) return <HelpTopicPage slug={topic} />

  return <HelpHubPage />
}

/** HelpHubPage menampilkan topik panduan, pencarian, dan FAQ dari server. */
function HelpHubPage() {
  const [query, setQuery] = useState('')

  return (
    <div className="bg-[#f5f5f7] text-[#09090b]">
      <div className="mx-auto w-full max-w-6xl px-4 pt-10 pb-16 sm:px-6 sm:pt-14 sm:pb-20">
        <div className="grid gap-10 lg:grid-cols-12 lg:items-end">
          <div className="lg:col-span-7">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[#71717a]">
              Pusat Bantuan
            </p>
            <h1 className="mt-4 text-[2rem] font-semibold leading-[1.08] tracking-tight sm:text-[2.9rem]">
              Jawaban yang bisa
              <span className="block text-[#a1a1aa]">Anda temukan sendiri.</span>
            </h1>
            <p className="mt-5 max-w-xl text-[15px] leading-relaxed text-[#52525b]">
              Kami menyusun penjelasan untuk pertanyaan yang paling sering muncul, dari
              menghubungkan akun broker sampai membaca tagihan. Bila belum terjawab, dukungan siap
              membantu.
            </p>
          </div>

          <div className="lg:col-span-5">
            <label htmlFor="cari-bantuan" className="sr-only">
              Cari topik bantuan
            </label>
            <div className="relative">
              <Search
                className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-[#a1a1aa]"
                aria-hidden="true"
              />
              <input
                id="cari-bantuan"
                type="search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Cari topik, misalnya jam atau broker"
                className="w-full rounded-xl border border-zinc-300 bg-white py-3 pl-10 pr-4 text-[14px] text-[#09090b] outline-none transition-colors placeholder:text-[#a1a1aa] focus:border-[#09090b]"
              />
            </div>
          </div>
        </div>

        <TopicGrid query={query} />

        <FaqSection query={query} />

        {/* Kontak: tanpa formulir, sesuai keputusan produk. */}
        <div className="mt-14 rounded-2xl border border-zinc-200 bg-white p-6 sm:p-8">
          <div className="flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
            <div className="max-w-xl">
              <h2 className="text-[17px] font-semibold tracking-tight">Belum menemukan jawaban?</h2>
              <p className="mt-2 text-[13px] leading-relaxed text-[#52525b]">
                Hubungi dukungan melalui surel. Sertakan alamat surel akun Anda dan waktu kejadian
                agar penelusuran lebih cepat.
              </p>
            </div>
            <a
              href="mailto:dukungan@rekavia.com"
              className="inline-flex shrink-0 items-center justify-center rounded-lg bg-[#09090b] px-4 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-zinc-800"
            >
              dukungan@rekavia.com
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}

/** TopicGrid menyaring topik berdasarkan kata kunci pencarian. */
function TopicGrid({ query }: { query: string }) {
  const needle = query.trim().toLowerCase()

  const filtered = TOPICS.filter((topic) => {
    if (needle === '') return true
    return (
      topic.title.toLowerCase().includes(needle) ||
      topic.summary.toLowerCase().includes(needle) ||
      topic.keywords.some((word) => word.includes(needle))
    )
  })

  // Zona waktu punya halaman tersendiri; ikut dicari sebagai topik biasa.
  const showTimezone =
    needle === '' ||
    ['jam', 'waktu', 'ny', 'wib', 'zona', 'sesi', 'market'].some((word) => word.includes(needle))

  return (
    <section aria-labelledby="topik-bantuan" className="mt-14">
      <div className="flex items-end justify-between border-t border-zinc-300/70 pt-6">
        <h2 id="topik-bantuan" className="text-[13px] font-semibold uppercase tracking-[0.16em]">
          Topik panduan
        </h2>
        <p className="text-[12px] text-[#71717a] tabular">
          {filtered.length + (showTimezone ? 1 : 0)} topik
        </p>
      </div>

      {filtered.length === 0 && !showTimezone ? (
        <p className="mt-6 rounded-xl border border-zinc-200 bg-white p-5 text-[13px] text-[#52525b]">
          Tidak ada topik yang cocok dengan kata kunci itu. Coba kata lain, atau tanyakan langsung
          ke dukungan.
        </p>
      ) : (
        <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {showTimezone && (
            <Link
              to="/bantuan/zona-waktu"
              className="group flex flex-col justify-between rounded-xl border border-zinc-200 bg-white p-5 transition-colors hover:border-zinc-400"
            >
              <div>
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#f4f4f5] text-[#3f3f46]">
                  <Clock className="h-4 w-4" aria-hidden="true" />
                </span>
                <h3 className="mt-3 text-[14px] font-semibold tracking-tight">
                  Sesi pasar &amp; zona waktu
                </h3>
                <p className="mt-1.5 text-[12.5px] leading-relaxed text-[#52525b]">
                  Mengapa panel menampilkan jam New York dan WIB sekaligus.
                </p>
              </div>
              <span className="mt-4 inline-flex items-center gap-1 text-[12px] font-medium text-[#3f3f46]">
                Baca
                <ArrowUpRight
                  className="h-3.5 w-3.5 transition-transform group-hover:-translate-y-0.5 group-hover:translate-x-0.5"
                  aria-hidden="true"
                />
              </span>
            </Link>
          )}

          {filtered.map((topic) => {
            const Icon = topic.icon
            return (
              <Link
                key={topic.slug}
                to={`/bantuan/${topic.slug}`}
                className="group flex flex-col justify-between rounded-xl border border-zinc-200 bg-white p-5 transition-colors hover:border-zinc-400"
              >
                <div>
                  <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#f4f4f5] text-[#3f3f46]">
                    <Icon className="h-4 w-4" aria-hidden="true" />
                  </span>
                  <h3 className="mt-3 text-[14px] font-semibold tracking-tight">{topic.title}</h3>
                  <p className="mt-1.5 text-[12.5px] leading-relaxed text-[#52525b]">
                    {topic.summary}
                  </p>
                </div>
                <span className="mt-4 inline-flex items-center gap-1 text-[12px] font-medium text-[#3f3f46]">
                  Baca
                  <ArrowUpRight
                    className="h-3.5 w-3.5 transition-transform group-hover:-translate-y-0.5 group-hover:translate-x-0.5"
                    aria-hidden="true"
                  />
                </span>
              </Link>
            )
          })}
        </div>
      )}
    </section>
  )
}

/** FaqSection menampilkan tanya-jawab dari server, dikelompokkan per kategori. */
function FaqSection({ query }: { query: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ['public', 'faq'],
    queryFn: publicContentApi.faq,
  })

  const needle = query.trim().toLowerCase()

  const filtered = (data ?? []).filter((item) => {
    if (needle === '') return true
    return (
      item.question.toLowerCase().includes(needle) ||
      item.answer_md.toLowerCase().includes(needle) ||
      item.category.toLowerCase().includes(needle)
    )
  })

  const grouped = useMemo(() => {
    const map = new Map<string, typeof filtered>()
    for (const item of filtered) {
      const list = map.get(item.category) ?? []
      list.push(item)
      map.set(item.category, list)
    }
    return [...map.entries()]
  }, [filtered])

  return (
    <section aria-labelledby="faq-bantuan" className="mt-14">
      <div className="flex items-end justify-between border-t border-zinc-300/70 pt-6">
        <h2 id="faq-bantuan" className="text-[13px] font-semibold uppercase tracking-[0.16em]">
          Pertanyaan umum
        </h2>
      </div>

      {isLoading && (
        <div className="mt-5 space-y-3">
          {[0, 1, 2, 3].map((index) => (
            <div
              key={index}
              className="flex items-center justify-between rounded-xl border border-zinc-200 bg-white px-5 py-4"
            >
              <div className="w-3/4 space-y-2">
                <Skeleton className="h-4 w-2/3 bg-zinc-200" />
                <Skeleton className="h-3 w-1/3 bg-zinc-200" />
              </div>
              <Skeleton className="h-4 w-4 rounded-full bg-zinc-200" />
            </div>
          ))}
        </div>
      )}

      {!isLoading && grouped.length === 0 && (
        <p className="mt-5 rounded-xl border border-zinc-200 bg-white p-5 text-[13px] text-[#52525b]">
          {needle === ''
            ? 'Belum ada pertanyaan umum yang terbit. Silakan hubungi dukungan untuk pertanyaan Anda.'
            : 'Tidak ada pertanyaan umum yang cocok dengan kata kunci itu.'}
        </p>
      )}

      <div className="mt-5 space-y-8">
        {grouped.map(([category, items]) => (
          <div key={category}>
            <h3 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[#71717a]">
              {category}
            </h3>
            <div className="mt-3 divide-y divide-zinc-200 rounded-xl border border-zinc-200 bg-white">
              {items.map((item) => (
                <details key={item.id} className="group px-5 py-4">
                  <summary className="flex cursor-pointer list-none items-start justify-between gap-4">
                    <span className="text-[14px] font-medium leading-snug text-[#09090b]">
                      {item.question}
                    </span>
                    <span
                      aria-hidden="true"
                      className="mt-0.5 shrink-0 text-[16px] leading-none text-[#a1a1aa] transition-transform group-open:rotate-45"
                    >
                      +
                    </span>
                  </summary>
                  <div className="mt-3 pr-6">
                    <MarkdownView source={item.answer_md} className="text-[13.5px]" />
                  </div>
                </details>
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

/**
 * HelpTopicPage menampilkan satu panduan dengan daftar isi yang menempel.
 *
 * Panduan menyimpan isinya di kode sebagai langkah bernomor, lalu dirender ke
 * markdown agar memakai perender yang sama dengan dokumen legal.
 */
function HelpTopicPage({ slug }: { slug: string }) {
  const topic = TOPICS.find((item) => item.slug === slug)
  const title = TOPIC_TITLES[slug]

  const source = useMemo(() => {
    if (!topic) return ''
    return topic.steps
      .map((step, index) => `## ${index + 1}. ${step.title}\n\n${step.body}`)
      .join('\n\n')
  }, [topic])

  const blocks = useMemo(() => parseMarkdown(source), [source])
  const outline = useMemo(() => outlineFromBlocks(blocks), [blocks])
  const activeId = useScrollSpy(
    outline.map((item) => item.id),
    outline.length > 0,
  )

  if (!topic) return <HelpUnknownTopic title={title} />

  return (
    <div className="bg-[#f5f5f7] text-[#09090b]">
      <div className="mx-auto w-full max-w-6xl px-4 pt-10 pb-20 sm:px-6 sm:pt-14 sm:pb-24">
        <nav aria-label="Jejak navigasi" className="text-[12px] text-[#71717a]">
          <ol className="flex flex-wrap items-center gap-1.5">
            <li>
              <Link to="/" className="transition-colors hover:text-[#09090b]">
                Beranda
              </Link>
            </li>
            <li aria-hidden="true">/</li>
            <li>
              <Link to="/bantuan" className="transition-colors hover:text-[#09090b]">
                Pusat Bantuan
              </Link>
            </li>
            <li aria-hidden="true">/</li>
            <li className="text-[#3f3f46]">{topic.title}</li>
          </ol>
        </nav>

        <div className="mt-8 grid gap-10 lg:grid-cols-12 lg:gap-12">
          <aside className="hidden lg:col-span-3 lg:block">
            <div className="sticky top-24 space-y-8">
              {outline.length > 0 && (
                <nav aria-label="Daftar isi panduan">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[#71717a]">
                    Daftar isi
                  </p>
                  <ul className="mt-4 space-y-0.5 border-l border-zinc-300/80">
                    {outline.map((item) => (
                      <li key={item.id}>
                        <a
                          href={`#${item.id}`}
                          aria-current={item.id === activeId ? 'true' : undefined}
                          className={cn(
                            '-ml-px block border-l-2 py-1.5 pl-3.5 text-[13px] leading-snug transition-colors',
                            item.id === activeId
                              ? 'border-[#09090b] font-medium text-[#09090b]'
                              : 'border-transparent text-[#71717a] hover:border-zinc-400 hover:text-[#3f3f46]',
                          )}
                        >
                          {item.label}
                        </a>
                      </li>
                    ))}
                  </ul>
                </nav>
              )}

              <nav aria-label="Topik bantuan lain">
                <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[#71717a]">
                  Panduan lain
                </p>
                <ul className="mt-4 space-y-1">
                  <li>
                    <Link
                      to="/bantuan/zona-waktu"
                      className="block rounded-md px-2 py-1.5 text-[13px] text-[#71717a] transition-colors hover:bg-white hover:text-[#09090b]"
                    >
                      Sesi pasar &amp; zona waktu
                    </Link>
                  </li>
                  {TOPICS.map((item) => (
                    <li key={item.slug}>
                      <Link
                        to={`/bantuan/${item.slug}`}
                        aria-current={item.slug === slug ? 'page' : undefined}
                        className={cn(
                          'block rounded-md px-2 py-1.5 text-[13px] transition-colors',
                          item.slug === slug
                            ? 'bg-white font-medium text-[#09090b]'
                            : 'text-[#71717a] hover:bg-white hover:text-[#09090b]',
                        )}
                      >
                        {item.title}
                      </Link>
                    </li>
                  ))}
                </ul>
              </nav>
            </div>
          </aside>

          <div className="lg:col-span-9">
            <div className="rounded-2xl border border-zinc-200 bg-white px-5 py-8 sm:px-10 sm:py-10">
              <header className="border-b border-zinc-200 pb-6">
                <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[#71717a]">
                  Pusat Bantuan
                </p>
                <h1 className="mt-2 text-[26px] font-semibold leading-tight tracking-tight sm:text-[32px]">
                  {topic.title}
                </h1>
                <p className="mt-3 max-w-2xl text-[13.5px] leading-relaxed text-[#52525b]">
                  {topic.summary}
                </p>
              </header>

              {/* Daftar isi ringkas untuk layar kecil. */}
              {outline.length > 0 && (
                <details className="mt-6 rounded-xl border border-zinc-200 bg-[#fafafa] px-4 py-3 lg:hidden">
                  <summary className="cursor-pointer text-[13px] font-semibold text-[#09090b]">
                    Langkah dalam panduan ini
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

              <MarkdownView source={source} className="mt-8" />

              <div className="mt-10 flex flex-col gap-3 border-t border-zinc-200 pt-6 sm:flex-row sm:items-center sm:justify-between">
                <p className="text-[13px] text-[#52525b]">
                  Masih ada yang belum jelas dari panduan ini?
                </p>
                <div className="flex flex-wrap gap-2">
                  <Link
                    to="/bantuan"
                    className="inline-flex items-center justify-center rounded-lg border border-zinc-300 px-3.5 py-2 text-[12px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100"
                  >
                    Semua topik
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
    </div>
  )
}

/** HelpUnknownTopic adalah halaman ramah untuk topik yang tidak dikenal. */
function HelpUnknownTopic({ title }: { title?: string }) {
  return (
    <div className="bg-[#f5f5f7] text-[#09090b]">
      <div className="mx-auto w-full max-w-3xl px-4 py-20 text-center sm:px-6">
        <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[#71717a]">
          Pusat Bantuan
        </p>
        <h1 className="mt-3 text-[26px] font-semibold tracking-tight">
          {title ? `${title} belum tersedia` : 'Topik tidak ditemukan'}
        </h1>
        <p className="mx-auto mt-3 max-w-md text-[14px] leading-relaxed text-[#52525b]">
          Halaman ini mungkin sudah dipindahkan. Kembali ke pusat bantuan untuk melihat topik yang
          tersedia, atau hubungi dukungan bila Anda memerlukan bantuan langsung.
        </p>
        <div className="mt-6 flex flex-wrap justify-center gap-3">
          <Link
            to="/bantuan"
            className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-4 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-zinc-800"
          >
            Pusat bantuan
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
  )
}
