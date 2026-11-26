import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { PageHeader } from '@/components/ui'
import { SessionsCard } from './SecuritySettingsPage'

/**
 * PerangkatPage memisahkan sesi perangkat dari Keamanan.
 *
 * Halaman ini dipakai ketika seseorang perlu mencabut akses perangkat lain,
 * bukan saat mengganti kata sandi. Menyatukan keduanya membuat halaman
 * keamanan panjang dan menenggelamkan tindakan yang mendesak.
 */
export default function PerangkatPage() {
  return (
    <>
      <div className="mb-2">
        <Link
          to="/pengaturan"
          className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors duration-150 hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Kembali ke Pengaturan
        </Link>
      </div>
      <PageHeader
        title="Perangkat & Sesi"
        description="Daftar perangkat yang sedang masuk ke akun Anda."
      />
      <div className="flex flex-col gap-4">
        <SessionsCard />
      </div>
    </>
  )
}
