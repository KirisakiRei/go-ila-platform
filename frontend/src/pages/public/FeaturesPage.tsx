import { Link } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'

const FEATURES = [
  {
    title: 'Dua engine, satu aturan',
    body: 'ILA 2.0 tersedia untuk semua paket; ILA 3.0 berjalan sebagai proses terpisah dan dibuka bertahap. Keduanya mengirim setup ke pipeline yang sama, sehingga aturan risiko tetap seragam.',
  },
  {
    title: 'Risiko diatur per akun',
    body: 'Persentase risiko per trade, batas jumlah trade harian, dan berhenti setelah menang. Semua dihitung dengan hari akuntansi WIB, sementara jam eksekusi mengikuti waktu New York.',
  },
  {
    title: 'Akun tetap milik Anda',
    body: 'Kami tidak memegang dana. Kredensial broker disimpan terenkripsi dan tidak pernah ditampilkan kembali, bahkan kepada admin.',
  },
  {
    title: 'Konfirmasi lewat tautan singkat',
    body: 'Mode manual mengirim tautan sekali pakai ke Telegram atau dashboard. Satu ketukan, tanpa mengetik ulang angka entry.',
  },
  {
    title: 'Posisi berjalan tidak diganggu',
    body: 'Mode pemeliharaan atau gangguan layanan hanya menahan entry baru. Stop loss dan take profit yang sudah terpasang tetap dijaga broker.',
  },
  {
    title: 'Jejak keputusan',
    body: 'Setiap perubahan pengaturan, tier, dan alasan penolakan sinyal tercatat. Saat ada pertanyaan "kenapa tidak masuk", jawabannya ada.',
  },
]

/** FeaturesPage menjelaskan kemampuan produk tanpa klaim hasil. */
export default function FeaturesPage() {
  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-12">
      <h1 className="text-2xl font-bold tracking-tight">Fitur</h1>
      <p className="mt-2 max-w-2xl text-[13px] text-muted-foreground">
        Semua fitur bekerja di atas akun broker Anda sendiri melalui token akses. Tidak ada dana yang
        disimpan di platform.
      </p>

      <div className="mt-6 grid gap-3 sm:grid-cols-2">
        {FEATURES.map((feature) => (
          <Card key={feature.title}>
            <CardHeader>
              <CardTitle className="text-sm">{feature.title}</CardTitle>
              <CardDescription className="text-xs leading-relaxed">{feature.body}</CardDescription>
            </CardHeader>
            <CardContent />
          </Card>
        ))}
      </div>

      <div className="mt-8 flex flex-wrap gap-2">
        {/* Anchor biasa: panel pelanggan bundel terpisah di /app/*. */}
        <a href="/app/register" className="rounded-md bg-foreground px-4 py-2 text-[13px] font-medium text-background">
          Mulai uji coba
        </a>
        <Link to="/bantuan/zona-waktu" className="rounded-md border border-border px-4 py-2 text-[13px] font-medium">
          Bagaimana zona waktu bekerja
        </Link>
      </div>
    </div>
  )
}
