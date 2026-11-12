import { Link, useSearchParams } from 'react-router-dom'
import { AuthLayout } from '@/features/auth/components/AuthLayout'
import { RegisterForm } from '@/features/auth/components/RegisterForm'
import { useInviteOnly } from '@/lib/use-invite-only'
import { useInviteCheck } from '@/lib/invite-api'

/** Kerangka halaman dengan tautan ke halaman masuk, dipakai semua keadaan. */
function Bungkus({ children }: { children: React.ReactNode }) {
  return (
    <AuthLayout
      footer={
        <>
          Sudah punya akun?{' '}
          <Link to="/login" className="font-medium text-foreground underline underline-offset-4">
            Masuk
          </Link>
        </>
      }
    >
      {children}
    </AuthLayout>
  )
}

/** Kartu penjelasan ketika pendaftaran tidak bisa dilanjutkan. */
function TidakBisaMasuk({ judul, isi }: { judul: string; isi: string }) {
  return (
    <div className="flex flex-col gap-3">
      <h1 className="text-lg font-bold tracking-tight text-foreground">{judul}</h1>
      <p className="text-sm leading-relaxed text-muted-foreground">{isi}</p>
      <Link
        to="/login"
        className="mt-1 inline-flex h-10 items-center justify-center rounded-lg bg-foreground px-4 text-sm font-semibold text-background"
      >
        Masuk ke akun
      </Link>
    </div>
  )
}

export default function RegisterPage() {
  const [params] = useSearchParams()
  const inviteCode = params.get('invite')?.trim() ?? ''
  const inviteOnly = useInviteOnly()
  const cek = useInviteCheck(inviteCode || null)

  // Tanpa kode undangan dan platform tertutup: tidak ada jalan mendaftar.
  if (!inviteCode && inviteOnly) {
    return (
      <Bungkus>
        <TidakBisaMasuk
          judul="Akses lewat undangan"
          isi="Pendaftaran mandiri sedang ditutup. Akun dibuat lewat tautan undangan dari admin. Sudah punya akun? Silakan masuk seperti biasa."
        />
      </Bungkus>
    )
  }

  // Ada kode undangan: tampilkan formulir hanya setelah kodenya terbukti berlaku.
  // Keputusan tetap diambil backend saat formulir dikirim; pemeriksaan ini
  // hanya supaya orang tidak mengisi formulir yang pasti ditolak.
  if (inviteCode) {
    if (cek.isPending) {
      return (
        <Bungkus>
          <p className="text-sm text-muted-foreground">Memeriksa tautan undangan…</p>
        </Bungkus>
      )
    }

    if (cek.isError || !cek.data?.valid) {
      return (
        <Bungkus>
          <TidakBisaMasuk
            judul="Undangan tidak berlaku"
            isi="Tautan ini sudah habis, kedaluwarsa, atau dicabut. Minta tautan baru ke admin."
          />
        </Bungkus>
      )
    }

    return (
      <Bungkus>
        <RegisterForm inviteCode={inviteCode} />
      </Bungkus>
    )
  }

  // Pendaftaran mandiri terbuka seperti sebelumnya.
  return (
    <Bungkus>
      <RegisterForm />
    </Bungkus>
  )
}
