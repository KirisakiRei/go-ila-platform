import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { request } from '@/lib/http'
import { Card, CardContent, EmptyState, PageHeader, Skeleton } from '@/components/ui'

type LegalDoc = {
  slug: string
  version: number
  title: string
  body_md?: string
  require_reaccept: boolean
  published: boolean
}

/**
 * LegalPage menampilkan dokumen legal publik.
 *
 * Halaman ini boleh dibaca tanpa masuk: calon pengguna perlu menilai isinya
 * sebelum memutuskan mendaftar. Persetujuan hanya diminta setelah login.
 */
export default function LegalPage() {
  const { slug } = useParams()

  const list = useQuery({
    queryKey: ['legal', 'list'],
    queryFn: () => request<LegalDoc[]>('/legal'),
    enabled: !slug,
  })
  const doc = useQuery({
    queryKey: ['legal', 'doc', slug],
    queryFn: () => request<LegalDoc>(`/legal/${slug}`),
    enabled: Boolean(slug),
  })

  if (slug) {
    if (doc.isLoading) return <Skeleton className="h-64 w-full" />
    if (doc.isError || !doc.data) {
      return <EmptyState title="Dokumen tidak ditemukan" description="Dokumen mungkin sudah diganti versi baru." />
    }
    return (
      <>
        <PageHeader
          title={doc.data.title}
          description={`Versi ${doc.data.version}`}
          actions={
            <Link className="text-[12px] underline" to="/legal">
              Semua dokumen
            </Link>
          }
        />
        <Card>
          <CardContent className="whitespace-pre-wrap p-5 text-[13px] leading-relaxed">
            {doc.data.body_md}
          </CardContent>
        </Card>
      </>
    )
  }

  if (list.isLoading) return <Skeleton className="h-40 w-full" />

  return (
    <>
      <PageHeader title="Dokumen" description="Ketentuan dan kebijakan yang berlaku di Imaginary Liquidity." />
      <div className="grid gap-3 sm:grid-cols-2">
        {(list.data ?? []).map((item) => (
          <Link key={item.slug} to={`/legal/${item.slug}`}>
            <Card className="h-full transition-colors hover:bg-muted/40">
              <CardContent className="flex flex-col gap-1 p-4">
                <span className="text-[13px] font-medium">{item.title}</span>
                <span className="text-[11px] text-muted-foreground">Versi {item.version}</span>
              </CardContent>
            </Card>
          </Link>
        ))}
        {list.data && list.data.length === 0 && (
          <EmptyState title="Belum ada dokumen" description="Dokumen akan tampil setelah diterbitkan." />
        )}
      </div>
    </>
  )
}
