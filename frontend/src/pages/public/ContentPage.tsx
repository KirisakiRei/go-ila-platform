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
        <div className="mx-auto w-full max-w-6xl animate-pulse px-4 py-16 sm:px-6">
          <div className="h-3 w-24 rounded bg-zinc-200" />
          <div className="mt-4 h-10 w-3/4 rounded bg-zinc-200" />
          <div className="mt-3 h-4 w-1/2 rounded bg-zinc-200" />
        </div>
      </div>
    )
  }

  if (!data) return <PageEmpty title={fallbackTitle} />

  return <PageRenderer page={data} />
}
