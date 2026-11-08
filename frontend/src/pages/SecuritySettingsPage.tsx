import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { request } from '@/lib/http'
import { sessionApi, securityApi } from '@/lib/api'
import { authApi } from '@/lib/auth-api'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Label,
  PageHeader,
  Skeleton,
} from '@/components/ui'
import { formatDateTime } from '@/lib/format'
import type { Session } from '@/types/api'

type TwoFactorStatus = {
  enabled: boolean
  activated_at: string
  last_used_at: string
  recovery_remaining: number
}

type TwoFactorSetup = {
  secret: string
  otpauth_url: string
  expires_at: string
}

/**
 * Penyiapan 2FA disimpan di sessionStorage agar tahan refresh namun hilang saat
 * tab ditutup. Tanpa ini, pengguna yang menekan refresh harus memindai ulang
 * QR dan secret lama tetap menunggu di server sampai kedaluwarsa.
 */
const TWO_FACTOR_SETUP_KEY = 'ila_2fa_setup'

function readStoredSetup(): TwoFactorSetup | null {
  const raw = sessionStorage.getItem(TWO_FACTOR_SETUP_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as TwoFactorSetup
    if (!parsed.secret || !parsed.expires_at) return null
    if (new Date(parsed.expires_at).getTime() <= Date.now()) return null
    return parsed
  } catch {
    return null
  }
}

function storeSetup(value: TwoFactorSetup | null) {
  if (value) sessionStorage.setItem(TWO_FACTOR_SETUP_KEY, JSON.stringify(value))
  else sessionStorage.removeItem(TWO_FACTOR_SETUP_KEY)
}

const twoFactorApi = {
  status: () => request<TwoFactorStatus>('/2fa'),
  setup: () => request<TwoFactorSetup>('/2fa/setup', { method: 'POST' }),
  activate: (code: string) =>
    request<{ recovery_codes: string[] }>('/2fa/activate', { method: 'POST', body: { code } }),
  disable: (code: string) => request<void>('/2fa/disable', { method: 'POST', body: { code } }),
  regenerate: () => request<{ recovery_codes: string[] }>('/2fa/recovery-codes', { method: 'POST', body: {} }),
}

/**
 * SecuritySettingsPage mengumpulkan seluruh pengaturan keamanan akun.
 *
 * 2FA memakai TOTP standar (RFC 6238) tanpa layanan pihak ketiga: kode dapat
 * dibaca aplikasi authenticator mana pun. Kode pemulihan hanya ditampilkan
 * sekali, saat dibuat (plans/17 §6).
 */
export default function SecuritySettingsPage() {
  return (
    <>
      <PageHeader
        title="Keamanan"
        description="Kata sandi, verifikasi email, autentikasi dua faktor, dan perangkat yang sedang masuk."
      />
      <div className="flex flex-col gap-4">
        <EmailVerificationCard />
        <PasswordCard />
        <TwoFactorCard />
        <SessionsCard />
        <IdleWarningCard />
      </div>
    </>
  )
}

function EmailVerificationCard() {
  const me = useQuery({ queryKey: ['auth', 'me'], queryFn: authApi.me })
  const resend = useMutation({ mutationFn: () => authApi.resendVerification() })

  const verified = Boolean(me.data?.email_verified_at)

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div>
            <CardTitle className="text-sm">Email</CardTitle>
            <CardDescription className="text-xs">
              Verifikasi membuka trial, koneksi broker, dan pembayaran.
            </CardDescription>
          </div>
          <Badge variant={verified ? 'solid' : 'outline'}>{verified ? 'Terverifikasi' : 'Belum'}</Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="text-[13px]">{me.data?.email ?? '—'}</p>
        {!verified && (
          <>
            <p className="text-[12px] text-muted-foreground">
              Selama belum terverifikasi, dashboard tetap bisa dibuka tetapi trial, akun broker, dan
              checkout akan ditolak.
            </p>
            <Button size="sm" disabled={resend.isPending} onClick={() => resend.mutate()}>
              Kirim ulang tautan verifikasi
            </Button>
            {resend.isSuccess && <Alert variant="success">Tautan baru sudah dikirim.</Alert>}
            {resend.isError && (
              <Alert variant="destructive">
                {resend.error instanceof Error ? resend.error.message : 'Gagal mengirim tautan.'}
              </Alert>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}

function PasswordCard() {
  const [form, setForm] = useState({ current: '', next: '', confirm: '' })
  const [notice, setNotice] = useState<string | null>(null)

  const change = useMutation({
    mutationFn: () => securityApi.changePassword({ current_password: form.current, new_password: form.next }),
    onSuccess: () => {
      // Ganti sandi mencabut semua sesi; pengguna diarahkan masuk kembali.
      window.location.href = '/app/login'
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal mengganti kata sandi.'),
  })

  const mismatch = form.confirm.length > 0 && form.next !== form.confirm
  const canSubmit = form.current !== '' && form.next.length >= 8 && form.next === form.confirm

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Kata sandi</CardTitle>
        <CardDescription className="text-xs">
          Setelah diganti, semua perangkat diminta masuk kembali.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        {notice && <Alert variant="destructive">{notice}</Alert>}
        <div className="grid gap-2 sm:grid-cols-3">
          <div>
            <Label htmlFor="cur">Sandi saat ini</Label>
            <Input
              id="cur"
              type="password"
              autoComplete="current-password"
              value={form.current}
              onChange={(event) => setForm({ ...form, current: event.target.value })}
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="new">Sandi baru</Label>
            <Input
              id="new"
              type="password"
              autoComplete="new-password"
              value={form.next}
              onChange={(event) => setForm({ ...form, next: event.target.value })}
              className="mt-1"
            />
          </div>
          <div>
            <Label htmlFor="conf">Ulangi sandi baru</Label>
            <Input
              id="conf"
              type="password"
              autoComplete="new-password"
              value={form.confirm}
              onChange={(event) => setForm({ ...form, confirm: event.target.value })}
              className="mt-1"
            />
            {mismatch && <p className="mt-1 text-[11px] text-muted-foreground">Belum sama.</p>}
          </div>
        </div>
        <Button size="sm" disabled={!canSubmit || change.isPending} onClick={() => change.mutate()}>
          Ganti kata sandi
        </Button>
      </CardContent>
    </Card>
  )
}

function TwoFactorCard() {
  const queryClient = useQueryClient()
  const [setup, setSetup] = useState<TwoFactorSetup | null>(() => readStoredSetup())
  const [code, setCode] = useState('')
  const [codes, setCodes] = useState<string[] | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const status = useQuery({ queryKey: ['2fa', 'status'], queryFn: twoFactorApi.status })

  const begin = useMutation({
    mutationFn: () => twoFactorApi.setup(),
    onSuccess: (data) => {
      storeSetup(data)
      setSetup(data)
      setNotice(null)
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal memulai 2FA.'),
  })

  const activate = useMutation({
    mutationFn: () => twoFactorApi.activate(code),
    onSuccess: (data) => {
      setCodes(data.recovery_codes)
      storeSetup(null)
      setSetup(null)
      setCode('')
      void queryClient.invalidateQueries({ queryKey: ['2fa'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Kode tidak sah.'),
  })

  const disable = useMutation({
    mutationFn: () => twoFactorApi.disable(code),
    onSuccess: () => {
      setCode('')
      setNotice('2FA dimatikan.')
      void queryClient.invalidateQueries({ queryKey: ['2fa'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Kode tidak sah.'),
  })

  const cancelSetup = () => {
    storeSetup(null)
    setSetup(null)
    setCode('')
  }

  const enabled = status.data?.enabled ?? false

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div>
            <CardTitle className="text-sm">Autentikasi dua faktor</CardTitle>
            <CardDescription className="text-xs">
              Kode 6 digit dari aplikasi authenticator. Bisa dimatikan; tidak wajib.
            </CardDescription>
          </div>
          <Badge variant={enabled ? 'solid' : 'outline'}>{enabled ? 'Aktif' : 'Nonaktif'}</Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {notice && <Alert>{notice}</Alert>}
        {status.isLoading && <Skeleton className="h-12 w-full" />}

        {codes && (
          <Alert variant="warning">
            <p className="font-medium">Simpan kode pemulihan ini sekarang.</p>
            <p className="mt-1 text-[12px]">Kode tidak akan ditampilkan lagi.</p>
            <div className="mt-2 grid grid-cols-2 gap-1 font-mono text-[12px] sm:grid-cols-4">
              {codes.map((item) => (
                <span key={item}>{item}</span>
              ))}
            </div>
          </Alert>
        )}

        {!enabled && !setup && (
          <Button size="sm" disabled={begin.isPending} onClick={() => begin.mutate()}>
            Aktifkan 2FA
          </Button>
        )}

        {setup && (
          <div className="space-y-2">
            <p className="text-[12px] text-muted-foreground">
              Masukkan kunci ini ke aplikasi authenticator, lalu ketik kode 6 digit yang muncul.
            </p>
            <p className="rounded-md border border-border p-2 font-mono text-[12px] break-all">{setup.secret}</p>
            <div className="flex gap-2">
              <Input
                value={code}
                onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="123456"
                inputMode="numeric"
                className="max-w-40 font-mono"
              />
              <Button size="sm" disabled={code.length !== 6 || activate.isPending} onClick={() => activate.mutate()}>
                Aktifkan
              </Button>
              <Button size="sm" variant="ghost" onClick={cancelSetup}>
                Batal
              </Button>
            </div>
          </div>
        )}

        {enabled && (
          <div className="space-y-2">
            {status.data?.activated_at && (
              <p className="text-[12px] text-muted-foreground">
                Aktif sejak {formatDateTime(status.data.activated_at)} · kode pemulihan tersisa{' '}
                {status.data.recovery_remaining}
              </p>
            )}
            <div className="flex flex-wrap gap-2">
              <Input
                value={code}
                onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="Kode 6 digit"
                inputMode="numeric"
                className="max-w-40 font-mono"
              />
              <Button
                size="sm"
                variant="outline"
                disabled={code.length !== 6 || disable.isPending}
                onClick={() => disable.mutate()}
              >
                Matikan 2FA
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function SessionsCard() {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)

  const sessions = useQuery({ queryKey: ['me', 'sessions'], queryFn: sessionApi.list })

  const revoke = useMutation({
    mutationFn: (id: string) => sessionApi.revoke(id),
    onSuccess: () => {
      setNotice('Perangkat dicabut.')
      void queryClient.invalidateQueries({ queryKey: ['me', 'sessions'] })
    },
  })

  const logoutAll = useMutation({
    mutationFn: () => securityApi.logoutAll(),
    onSuccess: () => {
      window.location.href = '/app/login'
    },
  })

  const rows: Session[] = sessions.data ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Perangkat yang masuk</CardTitle>
        <CardDescription className="text-xs">
          Cabut perangkat yang tidak Anda kenali. Sesi yang dicabut langsung tidak berlaku.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        {notice && <Alert>{notice}</Alert>}
        {sessions.isLoading && <Skeleton className="h-20 w-full" />}
        {rows.map((row) => (
          <div key={row.id} className="flex flex-wrap items-center justify-between gap-2 border-b border-border pb-2 last:border-0">
            <div className="min-w-0">
              <p className="text-[13px] font-medium">
                {row.device_label || 'Perangkat tidak dikenal'}
                {row.current && <span className="ml-2 text-[11px] text-muted-foreground">(perangkat ini)</span>}
              </p>
              <p className="text-[11px] text-muted-foreground">
                Terakhir aktif {formatDateTime(row.last_seen_at)}
              </p>
            </div>
            {!row.current && (
              <Button size="sm" variant="outline" onClick={() => revoke.mutate(row.id)}>
                Cabut
              </Button>
            )}
          </div>
        ))}
        {!sessions.isLoading && rows.length > 0 && (
          <Button size="sm" variant="ghost" disabled={logoutAll.isPending} onClick={() => logoutAll.mutate()}>
            Keluar dari semua perangkat
          </Button>
        )}
      </CardContent>
    </Card>
  )
}

function IdleWarningCard() {
  const queryClient = useQueryClient()
  const me = useQuery({ queryKey: ['auth', 'me'], queryFn: authApi.me })
  const [notice, setNotice] = useState<string | null>(null)

  const toggle = useMutation({
    mutationFn: (enabled: boolean) =>
      request<{ idle_warning: boolean }>('/me/idle-warning', { method: 'PATCH', body: { enabled } }),
    onSuccess: () => {
      setNotice('Preferensi disimpan.')
      void queryClient.invalidateQueries({ queryKey: ['auth', 'me'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan preferensi.'),
  })

  const value = me.data?.idle_warning ?? true

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Peringatan tidak ada aktivitas</CardTitle>
        <CardDescription className="text-xs">
          Menampilkan dialog "masih di sini?" sebelum sesi berakhir karena diam. Mematikan dialog tidak
          memperpanjang sesi: batasnya tetap berlaku di server.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        {notice && <Alert>{notice}</Alert>}
        <label className="flex items-center gap-2 text-[13px]">
          <input
            type="checkbox"
            checked={value}
            disabled={toggle.isPending}
            onChange={(event) => toggle.mutate(event.target.checked)}
          />
          Tampilkan peringatan sebelum keluar otomatis
        </label>
      </CardContent>
    </Card>
  )
}
