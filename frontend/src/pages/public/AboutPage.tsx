import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'

const SECTIONS = [
  {
    title: 'Apa yang kami bangun',
    body: 'Imaginary Liquidity adalah perangkat lunak yang menerjemahkan aturan trading tertulis menjadi proses yang dapat diaudit: pengenalan setup, pembatasan risiko, eksekusi, dan pencatatan.',
  },
  {
    title: 'Apa yang tidak kami lakukan',
    body: 'Kami tidak mengelola dana, tidak menjanjikan hasil, dan tidak menutup posisi Anda. Setiap order berjalan di akun broker Anda sendiri dengan kredensial yang Anda berikan.',
  },
  {
    title: 'Bagaimana keputusan diambil',
    body: 'Engine mencari setup memakai aturan yang sama untuk semua pelanggan. Runner menilai kelayakan per akun: kuota harian, tier, jam pasar, dan persetujuan dokumen.',
  },
  {
    title: 'Akses yang kami pegang',
    body: 'Kredensial broker disimpan terenkripsi dengan kunci master yang tidak berada di dalam database. Admin tidak memiliki jalan untuk menampilkan kredensial tersebut.',
  },
]

/** AboutPage menjelaskan produk dan batasannya secara terbuka. */
export default function AboutPage() {
  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-12">
      <h1 className="text-2xl font-bold tracking-tight">Tentang Imaginary Liquidity</h1>
      <p className="mt-2 text-[13px] text-muted-foreground">
        Dibangun untuk trader yang sudah punya aturan dan ingin menjalankannya secara konsisten.
      </p>

      <div className="mt-6 space-y-3">
        {SECTIONS.map((section) => (
          <Card key={section.title}>
            <CardHeader>
              <CardTitle className="text-sm">{section.title}</CardTitle>
              <CardDescription className="text-xs leading-relaxed">{section.body}</CardDescription>
            </CardHeader>
            <CardContent />
          </Card>
        ))}
      </div>
    </div>
  )
}
