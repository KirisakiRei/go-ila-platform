import { Skeleton } from '@/components/ui/skeleton'
import { PageEmpty, PageRenderer } from './components/PageRenderer'
import { usePageContent } from './lib/use-page-content'

/**
 * PublicContentPage merender halaman publik yang isinya dikelola admin.
 *
 * Dipakai halaman Fitur, Harga, Tentang, dan Legal Center agar semuanya punya
 * kerangka yang sama. Bila isi belum tersedia, halaman menampilkan keadaan
 * kosong yang jujur alih-alih teks lama yang tidak lagi diakui admin.
 */
export function PublicContentPage({
  slug,
  fallbackTitle,
}: {
  slug: string
  fallbackTitle: string
}) {
  const { data, isLoading } = usePageContent(slug)

  if (isLoading) {
    return (
      <div className="bg-[#f5f5f7]">
        {/* Skeleton Hero matching PublicHero */}
        <div className="border-b border-zinc-200/80 bg-white">
          <div className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 sm:py-24">
            <div className="grid gap-10 lg:grid-cols-12 lg:items-start">
              <div className="space-y-4 lg:col-span-7">
                <Skeleton className="h-4 w-28 bg-zinc-200" />
                <Skeleton className="h-10 w-4/5 bg-zinc-200" />
                <Skeleton className="h-5 w-full bg-zinc-200" />
                <Skeleton className="h-5 w-3/4 bg-zinc-200" />
                <div className="flex gap-3 pt-4">
                  <Skeleton className="h-10 w-36 rounded-lg bg-zinc-200" />
                  <Skeleton className="h-10 w-28 rounded-lg bg-zinc-200" />
                </div>
              </div>
              <div className="lg:col-span-5">
                <div className="space-y-3 rounded-2xl border border-zinc-200 bg-white/80 p-6">
                  <Skeleton className="h-4 w-24 bg-zinc-200" />
                  <Skeleton className="h-7 w-full bg-zinc-200" />
                  <Skeleton className="h-7 w-full bg-zinc-200" />
                  <Skeleton className="h-7 w-4/5 bg-zinc-200" />
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Skeleton Section Grid */}
        <div className="mx-auto w-full max-w-6xl px-4 py-12 sm:px-6">
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="space-y-3 rounded-2xl border border-zinc-200 bg-white p-6">
                <Skeleton className="h-8 w-8 rounded-lg bg-zinc-200" />
                <Skeleton className="h-5 w-1/2 bg-zinc-200" />
                <Skeleton className="h-4 w-full bg-zinc-200" />
                <Skeleton className="h-4 w-4/5 bg-zinc-200" />
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (!data) return <PageEmpty title={fallbackTitle} />

  return <PageRenderer page={data} />
}
