import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'

/**
 * EmptyState menandai area yang fiturnya belum tersedia.
 * Disengaja menampilkan penjelasan, bukan spinner tanpa akhir.
 */
export function EmptyState() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Sinyal</CardTitle>
        <CardDescription>Belum ada sinyal untuk ditampilkan.</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-[13px] text-muted-foreground">
          Setelah akun broker terhubung dan strategi dipilih, sinyal dari engine akan tampil di
          sini beserta alasannya.
        </p>
      </CardContent>
    </Card>
  )
}
