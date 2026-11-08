import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'

/** ZONA menjelaskan dua waktu yang muncul di dashboard. */
const ZONA = [
  {
    title: 'Waktu New York — jam trading',
    body: 'Setup dinilai memakai jam pasar New York, karena perilaku harga mengikuti sesi itu. Jam pasar buka dan tutup juga mengikuti zona ini, termasuk perubahan musim panas dan dingin.',
  },
  {
    title: 'Waktu Indonesia Barat (WIB) — hari akuntansi',
    body: 'Kuota harian, jumlah trade, dan penentuan "hari ini menang" memakai tanggal WIB. Tujuannya agar batas harian sesuai kebiasaan Anda, bukan berganti tanggal tengah malam waktu New York.',
  },
]

/**
 * HelpTimezonePage menjelaskan mengapa dashboard menampilkan dua zona waktu.
 *
 * Kebingungan ini paling sering muncul di pertanyaan dukungan, jadi
 * penjelasannya dibuat halaman sendiri (plans/22).
 */
export default function HelpTimezonePage() {
  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-12">
      <h1 className="text-2xl font-bold tracking-tight">Sesi Pasar & Zona Waktu</h1>
      <p className="mt-2 text-[13px] text-muted-foreground">
        Dashboard menampilkan dua waktu sekaligus. Keduanya sengaja berbeda peran.
      </p>

      <div className="mt-6 space-y-3">
        {ZONA.map((item) => (
          <Card key={item.title}>
            <CardHeader>
              <CardTitle className="text-sm">{item.title}</CardTitle>
              <CardDescription className="text-xs leading-relaxed">{item.body}</CardDescription>
            </CardHeader>
            <CardContent />
          </Card>
        ))}
      </div>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-sm">Contoh yang sering membingungkan</CardTitle>
          <CardDescription className="text-xs leading-relaxed">
            Pukul 09.00 WIB masih tanggal yang sama di Jakarta, tetapi di New York masih hari sebelumnya.
            Karena itu kuota harian Anda dihitung dari tanggal WIB, sedangkan sinyal yang masuk tetap
            dinilai memakai jam New York.
          </CardDescription>
        </CardHeader>
        <CardContent />
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle className="text-sm">Akhir pekan</CardTitle>
          <CardDescription className="text-xs leading-relaxed">
            Pasar tertutup Sabtu dan Minggu. Tidak ada pemindaian setup pada hari itu; pemindaian
            dilanjutkan saat pasar buka kembali.
          </CardDescription>
        </CardHeader>
        <CardContent />
      </Card>
    </div>
  )
}
