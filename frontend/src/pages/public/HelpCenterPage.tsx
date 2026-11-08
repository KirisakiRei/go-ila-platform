import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Alert, Card, CardContent, CardDescription, CardHeader, CardTitle, EmptyState, Skeleton } from '@/components/ui'
import { publicContentApi } from '@/lib/system-api'
import HelpTimezonePage from './HelpTimezonePage'

/**
 * HelpCenterPage adalah pusat bantuan: daftar topik dan FAQ dari database.
 *
 * FAQ diambil dari server sehingga bisa diubah tanpa deploy, sedangkan topik
 * panduan tetap di kode karena menyentuh alur yang dijaga ketat (plans/11 J8).
 */
export default function HelpCenterPage() {
  const { topik } = useParams()

  if (topik === 'zona-waktu') {
    return <HelpTimezonePage />
  }

  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-12">
      <h1 className="text-2xl font-bold tracking-tight">Pusat Bantuan</h1>
      <p className="mt-2 text-[13px] text-muted-foreground">
        Jawaban singkat untuk pertanyaan yang paling sering muncul.
      </p>

      <div className="mt-6 space-y-2">
        <Link
          to="/bantuan/zona-waktu"
          className="block rounded-md border border-border p-3 transition-colors hover:bg-secondary/40"
        >
          <p className="text-[13px] font-medium">Sesi pasar & zona waktu</p>
          <p className="text-[12px] text-muted-foreground">
            Mengapa dashboard menampilkan WIB dan New York sekaligus.
          </p>
        </Link>
      </div>

      <FaqList />
    </div>
  )
}

function FaqList() {
  const { data, isLoading, error } = useQuery({ queryKey: ['public', 'faq'], queryFn: publicContentApi.faq })

  const grouped = new Map<string, typeof data>()
  for (const item of data ?? []) {
    const list = grouped.get(item.category) ?? []
    list.push(item)
    grouped.set(item.category, list)
  }

  return (
    <div className="mt-6 space-y-5">
      {isLoading && <Skeleton className="h-40 w-full" />}
      {error && <Alert variant="destructive">Gagal memuat FAQ.</Alert>}
      {!isLoading && (data ?? []).length === 0 && (
        <EmptyState title="Belum ada FAQ" description="Hubungi dukungan lewat Telegram untuk pertanyaan Anda." />
      )}

      {[...grouped.entries()].map(([category, items]) => (
        <div key={category}>
          <h2 className="text-[13px] font-semibold uppercase tracking-wider text-muted-foreground">{category}</h2>
          <div className="mt-2 space-y-2">
            {(items ?? []).map((item) => (
              <Card key={item.id}>
                <CardHeader>
                  <CardTitle className="text-sm">{item.question}</CardTitle>
                  <CardDescription className="whitespace-pre-line text-xs leading-relaxed">
                    {item.answer_md}
                  </CardDescription>
                </CardHeader>
                <CardContent />
              </Card>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
