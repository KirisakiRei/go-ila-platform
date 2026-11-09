import { useQuery } from '@tanstack/react-query'
import { Check, Minus } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { formatIDR } from '@/features/admin/lib/format'
import { publicPlanApi } from '@/lib/admin-api'
import type { PublicPlan } from '@/types/admin'
import type { PageSection } from '@/types/system'
import { PageEmpty } from './components/PageRenderer'
import { PublicHero } from './components/PublicSections'
import { formatPlanBenefits, getPlanAudience } from './lib/plan-benefits'
import { pageTitle } from './lib/page-title'
import { usePageContent } from './lib/use-page-content'

/** Baris perbandingan disusun dari fitur paket, bukan dari teks bebas. */
type ComparisonRow = {
  label: string
  value: (plan: PublicPlan) => string
}

const COMPARISON: ComparisonRow[] = [
  {
    label: 'Akun broker',
    value: (plan) => {
      const limit = plan.features?.account_limit
      return limit ? (limit === '1' ? '1 akun' : `${limit} akun`) : '—'
    },
  },
  {
    label: 'Instrumen dipantau',
    value: (plan) => {
      const symbols = plan.features?.max_symbols
      if (!symbols) return '—'
      return symbols === '1' ? '1 pair' : `${symbols} pair`
    },
  },
  {
    label: 'Jeda sinyal',
    value: (plan) => {
      const delay = plan.features?.signal_delay_minutes
      if (delay === undefined) return '—'
      return delay === '0' ? 'Tanpa jeda' : `${delay} menit`
    },
  },
  {
    label: 'Eksekusi',
    value: (plan) => {
      if (plan.code === 'free') return 'Demo'
      return plan.features?.auto_execute === 'true' ? 'Auto & Manual' : 'Manual'
    },
  },
  {
    label: 'Strategi',
    value: (plan) =>
      plan.code === 'plus'
        ? plan.ila3_coming_soon
          ? 'ILA 2.0 + 3.0 bertahap'
          : 'ILA 2.0 + 3.0'
        : 'ILA 2.0',
  },
  {
    label: 'Ringkasan konteks setup',
    value: (plan) =>
      plan.features?.reason_ai === 'true' || plan.code === 'plus' ? 'Termasuk' : 'Tidak termasuk',
  },
]

/**
 * PricingPage menampilkan katalog paket dari basis data.
 *
 * Harga tidak ditulis di kode: perubahan dari panel admin langsung terlihat di
 * sini, sehingga tidak ada dua sumber harga yang bisa berbeda (plans/11 J4).
 * Judul, pengantar, FAQ, dan disclaimer berasal dari halaman "harga" yang
 * dikelola admin, sehingga kalimat pembuka dapat disunting tanpa deploy.
 */
export default function PricingPage() {
  const plansQuery = useQuery({
    queryKey: ['public', 'plans'],
    queryFn: publicPlanApi.list,
  })
  const page = usePageContent('harga')

  const plans = plansQuery.data ?? []
  const sections = page.data?.sections ?? []
  const faqSection = sections.find((section) => section.kind === 'faq')
  const noteSection = sections.find((section) => section.kind === 'callout')
  const comparisonSection = sections.find((section) => section.kind === 'table')

  if (page.isLoading) {
    return (
      <div className="bg-[#f5f5f7]">
        <div className="border-b border-zinc-200/80 bg-white">
          <div className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 sm:py-24">
            <div className="max-w-2xl space-y-4">
              <Skeleton className="h-4 w-24 bg-zinc-200" />
              <Skeleton className="h-10 w-3/4 bg-zinc-200" />
              <Skeleton className="h-5 w-full bg-zinc-200" />
              <Skeleton className="h-5 w-4/5 bg-zinc-200" />
              <div className="flex gap-3 pt-4">
                <Skeleton className="h-10 w-44 rounded-lg bg-zinc-200" />
                <Skeleton className="h-10 w-48 rounded-lg bg-zinc-200" />
              </div>
            </div>
          </div>
        </div>

        <div className="mx-auto w-full max-w-6xl px-4 py-12 sm:px-6">
          <div className="grid gap-4 lg:grid-cols-3">
            {[1, 2, 3].map((index) => (
              <div
                key={index}
                className="flex flex-col justify-between rounded-2xl border border-zinc-200 bg-white p-6 space-y-6"
              >
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Skeleton className="h-5 w-24 bg-zinc-200" />
                    <Skeleton className="h-4 w-3/4 bg-zinc-200" />
                  </div>
                  <Skeleton className="h-9 w-40 bg-zinc-200" />
                  <div className="space-y-2 pt-4 border-t border-zinc-100">
                    <Skeleton className="h-4 w-full bg-zinc-200" />
                    <Skeleton className="h-4 w-5/6 bg-zinc-200" />
                    <Skeleton className="h-4 w-4/5 bg-zinc-200" />
                    <Skeleton className="h-4 w-2/3 bg-zinc-200" />
                  </div>
                </div>
                <Skeleton className="h-10 w-full rounded-lg bg-zinc-200" />
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (!page.data) return <PageEmpty title={pageTitle('harga')} />

  return (
    <div className="bg-[#f5f5f7] text-[#09090b]">
      <PublicHero
        eyebrow={page.data.eyebrow}
        title={page.data.title}
        description={page.data.lead}
        actions={
          <>
            <a
              href="/app/register"
              className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-5 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-zinc-800"
            >
              Mulai Trial 7 Hari
            </a>
            <a
              href="/legal/refund"
              className="inline-flex items-center justify-center rounded-lg border border-zinc-300 bg-white px-5 py-2.5 text-[13px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100"
            >
              Kebijakan pengembalian dana
            </a>
          </>
        }
      />

      {/* Kartu paket: satu-satunya sumber harga adalah katalog aktif di server. */}
      <div className="mx-auto w-full max-w-6xl px-4 pb-14 sm:px-6">
        {plansQuery.isLoading ? (
          <div className="grid gap-4 lg:grid-cols-3">
            {[1, 2, 3].map((index) => (
              <div
                key={index}
                className="flex flex-col justify-between rounded-2xl border border-zinc-200 bg-white p-6 space-y-6"
              >
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Skeleton className="h-5 w-24 bg-zinc-200" />
                    <Skeleton className="h-4 w-3/4 bg-zinc-200" />
                  </div>
                  <Skeleton className="h-9 w-40 bg-zinc-200" />
                  <div className="space-y-2 pt-4 border-t border-zinc-100">
                    <Skeleton className="h-4 w-full bg-zinc-200" />
                    <Skeleton className="h-4 w-5/6 bg-zinc-200" />
                    <Skeleton className="h-4 w-4/5 bg-zinc-200" />
                    <Skeleton className="h-4 w-2/3 bg-zinc-200" />
                  </div>
                </div>
                <Skeleton className="h-10 w-full rounded-lg bg-zinc-200" />
              </div>
            ))}
          </div>
        ) : plans.length === 0 ? (
          <p className="rounded-2xl border border-zinc-200 bg-white p-6 text-[13px] text-[#52525b]">
            Katalog paket belum tersedia saat ini. Silakan periksa kembali sebentar lagi.
          </p>
        ) : (
          <div className="grid gap-4 lg:grid-cols-3">
            {plans.map((plan) => {
              const highlight = plan.code === 'plus'
              const benefits = formatPlanBenefits(plan)

              return (
                <article
                  key={plan.code}
                  className={
                    highlight
                      ? 'relative flex flex-col rounded-2xl border border-[#09090b] bg-[#09090b] p-6 text-white sm:p-7'
                      : 'relative flex flex-col rounded-2xl border border-zinc-200 bg-white p-6 sm:p-7'
                  }
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h2 className="text-[17px] font-semibold tracking-tight">{plan.name}</h2>
                      <p
                        className={
                          highlight
                            ? 'mt-1.5 text-[12.5px] leading-relaxed text-zinc-400'
                            : 'mt-1.5 text-[12.5px] leading-relaxed text-[#52525b]'
                        }
                      >
                        {getPlanAudience(plan)}
                      </p>
                    </div>
                    {highlight && (
                      <span className="shrink-0 rounded-full bg-white px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-[#09090b]">
                        Paling lengkap
                      </span>
                    )}
                  </div>

                  <div
                    className={
                      highlight
                        ? 'mt-6 border-b border-white/15 pb-6'
                        : 'mt-6 border-b border-zinc-200 pb-6'
                    }
                  >
                    <p className="tabular text-[30px] font-semibold leading-none tracking-tight">
                      {plan.price_idr === 0 ? 'Gratis' : formatIDR(plan.price_idr)}
                    </p>
                    <p
                      className={
                        highlight
                          ? 'mt-1.5 text-[11px] uppercase tracking-[0.1em] text-zinc-500'
                          : 'mt-1.5 text-[11px] uppercase tracking-[0.1em] text-[#a1a1aa]'
                      }
                    >
                      {plan.price_idr === 0 ? 'Selamanya' : 'Per bulan'}
                    </p>
                  </div>

                  <ul className="mt-6 flex-1 space-y-2.5">
                    {benefits.map((benefit) => (
                      <li key={benefit.label} className="flex items-start gap-2 text-[13px]">
                        <Check
                          className={
                            highlight
                              ? 'mt-0.5 h-3.5 w-3.5 shrink-0 text-white'
                              : 'mt-0.5 h-3.5 w-3.5 shrink-0 text-[#3f3f46]'
                          }
                          aria-hidden="true"
                        />
                        <span className={highlight ? 'text-zinc-200' : 'text-[#3f3f46]'}>
                          {benefit.label}
                        </span>
                      </li>
                    ))}
                    {plan.ila3_coming_soon && (
                      <li
                        className={
                          highlight
                            ? 'flex items-start gap-2 text-[12px] text-zinc-500'
                            : 'flex items-start gap-2 text-[12px] text-[#a1a1aa]'
                        }
                      >
                        <Minus className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                        <span>ILA 3.0 dibuka bertahap</span>
                      </li>
                    )}
                  </ul>

                  <a
                    href="/app/register"
                    className={
                      highlight
                        ? 'mt-7 inline-flex items-center justify-center rounded-lg bg-white px-4 py-2.5 text-[13px] font-semibold text-[#09090b] transition-colors hover:bg-zinc-200'
                        : 'mt-7 inline-flex items-center justify-center rounded-lg border border-zinc-300 px-4 py-2.5 text-[13px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100'
                    }
                  >
                    {plan.price_idr === 0 ? 'Mulai gratis' : `Pilih ${plan.name}`}
                  </a>
                </article>
              )
            })}
          </div>
        )}
      </div>

      {/* Tabel perbandingan: nilainya dihitung dari katalog aktif. */}
      {plans.length > 0 && (
        <section className="border-t border-zinc-200/80 bg-white">
          <div className="mx-auto w-full max-w-6xl px-4 py-14 sm:px-6 sm:py-16">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
              <h2 className="text-[24px] font-semibold leading-tight tracking-tight sm:text-[28px]">
                {comparisonSection?.title ?? 'Perbandingan paket'}
              </h2>
              <p className="max-w-sm text-[13px] leading-relaxed text-[#52525b]">
                {comparisonSection?.lead ||
                  'Nilai pada tabel dibaca langsung dari katalog aktif dan selalu berlaku sesuai tanggal penagihan.'}
              </p>
            </div>

            <div className="mt-8 overflow-x-auto rounded-2xl border border-zinc-200">
              <table className="w-full min-w-[680px] border-collapse text-left text-[13px]">
                <thead>
                  <tr className="bg-[#f4f4f5]">
                    <th
                      scope="col"
                      className="border-b border-zinc-200 px-5 py-3.5 text-[11px] font-semibold uppercase tracking-[0.12em] text-[#3f3f46]"
                    >
                      Kapabilitas
                    </th>
                    {plans.map((plan) => (
                      <th
                        key={plan.code}
                        scope="col"
                        className="border-b border-zinc-200 px-5 py-3.5 text-[13px] font-semibold text-[#09090b]"
                      >
                        {plan.name}
                        <span className="tabular mt-0.5 block text-[11px] font-normal text-[#71717a]">
                          {plan.price_idr === 0 ? 'Gratis' : formatIDR(plan.price_idr)}
                        </span>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {COMPARISON.map((row) => (
                    <tr key={row.label} className="even:bg-[#fafafa]">
                      <th
                        scope="row"
                        className="border-b border-zinc-100 px-5 py-3.5 text-[13px] font-medium text-[#3f3f46]"
                      >
                        {row.label}
                      </th>
                      {plans.map((plan) => (
                        <td
                          key={plan.code}
                          className="border-b border-zinc-100 px-5 py-3.5 text-[13px] text-[#52525b]"
                        >
                          {row.value(plan)}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="mt-3 text-[12px] text-[#71717a]">
              Trial 7 hari menyertakan fitur setara paket tertinggi dalam mode demo.
            </p>
          </div>
        </section>
      )}

      {/* FAQ pembayaran dari CMS. */}
      {faqSection && <PricingFaq section={faqSection} />}

      {/* Disclaimer risiko: wajib tampil di halaman harga. */}
      <section className="border-t border-zinc-200/80 bg-white">
        <div className="mx-auto w-full max-w-6xl px-4 py-10 sm:px-6">
          <div className="rounded-2xl border border-zinc-300 bg-[#fafafa] p-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[#09090b]">
              {noteSection?.title || 'Disclaimer risiko'}
            </p>
            <p className="mt-2 max-w-3xl whitespace-pre-line text-[13px] leading-relaxed text-[#52525b]">
              {noteSection?.body ||
                'Langganan memberi Anda akses ke alat bantu analisis dan eksekusi, bukan jaminan hasil. Hasil trading bergantung pada kondisi pasar dan keputusan Anda sendiri; modal dapat berkurang sebagian atau seluruhnya.'}{' '}
              <a
                href="/legal/risk"
                className="font-medium text-[#09090b] underline underline-offset-2"
              >
                Baca Disclaimer Risiko Trading
              </a>
              .
            </p>
          </div>
        </div>
      </section>
    </div>
  )
}

/** PricingFaq merender tanya-jawab pembayaran dari isi halaman. */
function PricingFaq({ section }: { section: PageSection }) {
  return (
    <section className="border-t border-zinc-200/80">
      <div className="mx-auto w-full max-w-6xl px-4 py-14 sm:px-6 sm:py-16">
        <div className="grid gap-10 lg:grid-cols-12">
          <div className="lg:col-span-4">
            <h2 className="text-[24px] font-semibold leading-tight tracking-tight sm:text-[28px]">
              {section.title}
            </h2>
            {section.lead && (
              <p className="mt-3 text-[13px] leading-relaxed text-[#52525b]">{section.lead}</p>
            )}
            <a
              href="/bantuan"
              className="mt-5 inline-flex items-center justify-center rounded-lg border border-zinc-300 bg-white px-4 py-2.5 text-[13px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100"
            >
              Buka pusat bantuan
            </a>
          </div>

          <div className="lg:col-span-8">
            <dl className="divide-y divide-zinc-200 border-t border-zinc-200">
              {section.items.map((item) => (
                <div key={item.label} className="py-5">
                  <dt className="text-[14px] font-semibold text-[#09090b]">{item.label}</dt>
                  <dd className="mt-1.5 text-[13px] leading-relaxed text-[#52525b]">
                    {item.value}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </div>
      </div>
    </section>
  )
}
