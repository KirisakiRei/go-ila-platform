import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, ShieldCheck } from 'lucide-react'
import { PageHeader, Card, Skeleton } from '@/components/ui'
import { authApi } from '@/lib/auth-api'
import { formatDate } from '@/lib/format'

const TIER_LABEL: Record<string, string> = {
  free: 'Gratis',
  basic: 'Basic',
  plus: 'Plus',
}

/** Baris data akun. Nilai kosong ditandai garis, bukan dibiarkan hilang. */
function Baris({ label, nilai }: { label: string; nilai: string }) {
  return (
    <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-0.5 border-b border-border py-2.5 last:border-0">
      <span className="text-[11.5px] text-muted-foreground">{label}</span>
      <span className="min-w-0 break-words text-[12.5px] font-medium text-foreground">
        {nilai || '—'}
      </span>
    </div>
  )
}

/**
 * ProfilPage menampilkan data akun.
 *
 * Sengaja hanya membaca: belum ada endpoint perubahan data diri, dan formulir
 * yang tidak bisa menyimpan lebih buruk daripada tidak ada formulir.
 */
export default function ProfilPage() {
  const { data, isLoading } = useQuery({ queryKey: ['auth', 'me'], queryFn: authApi.me })

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
      <PageHeader title="Profil & Akun" description="Data diri dan informasi kontak akun Anda." />

      {isLoading ? (
        <Skeleton className="h-56 w-full rounded-2xl" />
      ) : (
        <Card className="p-4">
          <Baris label="Nama" nilai={data?.name ?? ''} />
          <Baris label="Nama pengguna" nilai={data?.username ? `@${data.username}` : ''} />
          <Baris label="Email" nilai={data?.email ?? ''} />
          <Baris label="Telepon" nilai={data?.phone ?? ''} />
          <Baris label="Paket" nilai={TIER_LABEL[data?.tier ?? ''] ?? data?.tier ?? ''} />
          <Baris label="Terdaftar" nilai={data?.created_at ? formatDate(data.created_at) : ''} />
          <div className="flex items-center justify-between pt-3">
            <span className="inline-flex items-center gap-1.5 text-[11.5px] text-muted-foreground">
              <ShieldCheck className="h-3.5 w-3.5" />
              {data?.email_verified_at ? 'Email terverifikasi' : 'Email belum terverifikasi'}
            </span>
            <Link
              to="/pengaturan/keamanan"
              className="text-[11.5px] font-semibold text-foreground underline underline-offset-2"
            >
              Kelola keamanan
            </Link>
          </div>
        </Card>
      )}
    </>
  )
}
