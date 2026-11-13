import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import {
  ArrowLeft,
  Check,
  Copy,
  KeyRound,
  Laptop,
  LogOut,
  QrCode,
  ShieldAlert,
  ShieldCheck,
  Smartphone,
  Trash2,
} from 'lucide-react'
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
  Modal,
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
      <div className="mb-2">
        <Link
          to="/pengaturan"
          className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Kembali ke Pengaturan
        </Link>
      </div>

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
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [tab, setTab] = useState<'qr' | 'manual'>('qr')
  const [copiedKey, setCopiedKey] = useState(false)
  const [code, setCode] = useState('')
  const [codes, setCodes] = useState<string[] | null>(null)
  const [copiedAllCodes, setCopiedAllCodes] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)

  const status = useQuery({ queryKey: ['2fa', 'status'], queryFn: twoFactorApi.status })

  const begin = useMutation({
    mutationFn: () => twoFactorApi.setup(),
    onSuccess: (data) => {
      storeSetup(data)
      setSetup(data)
      setIsModalOpen(true)
      setNotice(null)
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal memulai penyiapan 2FA.'),
  })

  const activate = useMutation({
    mutationFn: () => twoFactorApi.activate(code),
    onSuccess: (data) => {
      setCodes(data.recovery_codes)
      storeSetup(null)
      setSetup(null)
      setIsModalOpen(false)
      setCode('')
      void queryClient.invalidateQueries({ queryKey: ['2fa'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Kode verifikasi tidak sah. Pastikan waktu perangkat Anda sinkron.'),
  })

  const disable = useMutation({
    mutationFn: () => twoFactorApi.disable(code),
    onSuccess: () => {
      setCode('')
      setNotice('Autentikasi dua faktor berhasil dimatikan.')
      void queryClient.invalidateQueries({ queryKey: ['2fa'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Kode verifikasi tidak sah.'),
  })

  const cancelSetup = () => {
    storeSetup(null)
    setSetup(null)
    setIsModalOpen(false)
    setCode('')
  }

  const copySecret = () => {
    if (!setup?.secret) return
    void navigator.clipboard.writeText(setup.secret)
    setCopiedKey(true)
    setTimeout(() => setCopiedKey(false), 2000)
  }

  const copyAllCodes = () => {
    if (!codes) return
    void navigator.clipboard.writeText(codes.join('\n'))
    setCopiedAllCodes(true)
    setTimeout(() => setCopiedAllCodes(false), 2000)
  }

  const enabled = status.data?.enabled ?? false

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between gap-2">
            <div>
              <CardTitle className="text-sm font-semibold">Autentikasi Dua Faktor (2FA)</CardTitle>
              <CardDescription className="text-xs">
                Mengamankan login menggunakan kode 6 digit berbasis TOTP (Google Authenticator).
              </CardDescription>
            </div>
            <Badge variant={enabled ? 'solid' : 'outline'}>{enabled ? 'Aktif' : 'Nonaktif'}</Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {notice && <Alert>{notice}</Alert>}
          {status.isLoading && <Skeleton className="h-12 w-full" />}

          {/* Dialog Sukses & Kode Pemulihan */}
          {codes && (
            <Alert variant="warning" className="border-amber-500/30 bg-amber-500/10">
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2 font-semibold text-amber-800 dark:text-amber-300">
                  <ShieldAlert className="h-4 w-4" />
                  <span>2FA Berhasil Diaktifkan! Simpan Kode Pemulihan Anda.</span>
                </div>
                <p className="text-xs text-amber-900/80 dark:text-amber-200/80">
                  Jika Anda kehilangan akses ke aplikasi authenticator, setiap kode berikut dapat digunakan 1 kali untuk masuk ke akun Anda. Kode ini tidak akan pernah ditampilkan kembali.
                </p>
                <div className="my-1 grid grid-cols-2 gap-2 rounded-lg border border-amber-500/20 bg-background/50 p-3 font-mono text-xs sm:grid-cols-4">
                  {codes.map((item) => (
                    <span key={item} className="select-all font-semibold text-foreground">
                      {item}
                    </span>
                  ))}
                </div>
                <div className="flex flex-wrap items-center gap-2 pt-1">
                  <Button size="sm" variant="outline" onClick={copyAllCodes}>
                    {copiedAllCodes ? (
                      <>
                        <Check className="mr-1 h-3.5 w-3.5 text-emerald-600" />
                        Tersalin ke Clipboard
                      </>
                    ) : (
                      <>
                        <Copy className="mr-1 h-3.5 w-3.5" />
                        Salin Semua Kode
                      </>
                    )}
                  </Button>
                  <Button size="sm" onClick={() => setCodes(null)}>
                    Saya Sudah Menyimpan Kode
                  </Button>
                </div>
              </div>
            </Alert>
          )}

          {!enabled && (
            <div className="flex flex-col gap-2">
              <p className="text-xs text-muted-foreground">
                Tingkatkan keamanan akun dengan mewajibkan verifikasi kode Google Authenticator saat masuk.
              </p>
              <Button
                size="sm"
                className="self-start"
                disabled={begin.isPending}
                isLoading={begin.isPending}
                onClick={() => begin.mutate()}
              >
                Aktifkan 2FA
              </Button>
            </div>
          )}

          {enabled && (
            <div className="space-y-3">
              {status.data?.activated_at && (
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <ShieldCheck className="h-4 w-4 text-emerald-600" />
                  <span>
                    Aktif sejak {formatDateTime(status.data.activated_at)} · tersisa{' '}
                    <strong>{status.data.recovery_remaining}</strong> kode pemulihan
                  </span>
                </div>
              )}
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <Input
                  value={code}
                  onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
                  placeholder="Kode 6 digit untuk verifikasi"
                  inputMode="numeric"
                  className="max-w-48 font-mono text-center tracking-widest"
                />
                <Button
                  size="sm"
                  variant="outline"
                  disabled={code.length !== 6 || disable.isPending}
                  isLoading={disable.isPending}
                  onClick={() => disable.mutate()}
                >
                  Matikan 2FA
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Modal Setup Google Authenticator */}
      <Modal
        open={isModalOpen && Boolean(setup)}
        onClose={cancelSetup}
        title="Penyiapan Google Authenticator"
        description="Amankan akun Anda dengan menghubungkan aplikasi Google Authenticator."
        size="lg"
      >
        {setup && (
          <div className="space-y-4 pt-1">
            {/* Tab Navigasi: QR Code vs Manual Key */}
            <div className="flex rounded-lg border border-border p-1">
              <button
                type="button"
                onClick={() => setTab('qr')}
                className={`flex flex-1 items-center justify-center gap-2 rounded-md py-1.5 text-xs font-semibold transition-all ${
                  tab === 'qr' ? 'bg-foreground text-background shadow-sm' : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                <QrCode className="h-3.5 w-3.5" />
                Pindai Kode QR
              </button>
              <button
                type="button"
                onClick={() => setTab('manual')}
                className={`flex flex-1 items-center justify-center gap-2 rounded-md py-1.5 text-xs font-semibold transition-all ${
                  tab === 'manual' ? 'bg-foreground text-background shadow-sm' : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                <KeyRound className="h-3.5 w-3.5" />
                Input Kunci Manual
              </button>
            </div>

            {/* Tab 1: Pindai QR Code */}
            {tab === 'qr' && (
              <div className="flex flex-col items-center gap-3 py-2 text-center">
                <div className="flex h-52 w-52 items-center justify-center rounded-xl border border-zinc-200 bg-white p-3 shadow-inner">
                  <QRCodeSVG value={setup.otpauth_url} size={184} level="M" />
                </div>
                <div className="max-w-sm space-y-1 text-xs text-muted-foreground">
                  <p className="font-semibold text-foreground">1. Buka aplikasi Google Authenticator di ponsel Anda.</p>
                  <p>2. Ketuk tanda "+" lalu pilih <strong>"Pindai kode QR"</strong>.</p>
                  <p>3. Arahkan kamera ponsel Anda ke barcode di atas.</p>
                </div>
              </div>
            )}

            {/* Tab 2: Input Manual */}
            {tab === 'manual' && (
              <div className="space-y-3 py-2 text-xs">
                <p className="text-muted-foreground">
                  Jika kamera Anda bermasalah, masukkan kunci rahasia berikut secara manual pada aplikasi Google Authenticator (pilih opsi <strong>"Masukkan kunci penyiapan"</strong>):
                </p>
                <div>
                  <Label className="text-[11px] text-muted-foreground">Kunci Rahasia (Secret Key)</Label>
                  <div className="mt-1 flex items-center justify-between gap-2 rounded-lg border border-border bg-secondary/50 p-2.5">
                    <span className="select-all font-mono text-sm font-semibold tracking-wider text-foreground">
                      {setup.secret}
                    </span>
                    <Button size="sm" variant="ghost" onClick={copySecret} className="h-8 gap-1 px-2.5 text-xs">
                      {copiedKey ? (
                        <>
                          <Check className="h-3.5 w-3.5 text-emerald-600" />
                          <span>Tersalin</span>
                        </>
                      ) : (
                        <>
                          <Copy className="h-3.5 w-3.5" />
                          <span>Salin</span>
                        </>
                      )}
                    </Button>
                  </div>
                </div>
                <p className="text-[11px] text-muted-foreground">
                  Jenis kunci: <strong>Berbasis waktu (TOTP)</strong> · Interval: <strong>30 detik</strong>
                </p>
              </div>
            )}

            {/* Verifikasi 6 Digit Input */}
            <div className="rounded-xl border border-border bg-secondary/20 p-4">
              <Label className="text-xs font-semibold text-foreground">
                Langkah Terakhir: Masukkan Kode 6-Digit dari Authenticator
              </Label>
              <div className="mt-2 flex flex-col gap-3 sm:flex-row">
                <Input
                  value={code}
                  onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
                  placeholder="000000"
                  inputMode="numeric"
                  className="max-w-40 text-center font-mono text-base font-semibold tracking-widest"
                />
                <div className="flex flex-1 justify-end gap-2">
                  <Button variant="outline" size="sm" onClick={cancelSetup}>
                    Batal
                  </Button>
                  <Button
                    size="sm"
                    disabled={code.length !== 6 || activate.isPending}
                    isLoading={activate.isPending}
                    onClick={() => activate.mutate()}
                  >
                    Verifikasi & Aktifkan
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}
      </Modal>
    </>
  )
}

function SessionsCard() {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)
  const [revokeTarget, setRevokeTarget] = useState<Session | null>(null)
  const [confirmLogoutAll, setConfirmLogoutAll] = useState(false)

  const sessions = useQuery({ queryKey: ['me', 'sessions'], queryFn: sessionApi.list })

  const revoke = useMutation({
    mutationFn: (id: string) => sessionApi.revoke(id),
    onSuccess: () => {
      setNotice('Sesi perangkat berhasil dicabut tanpa mengganggu sesi Anda saat ini.')
      setRevokeTarget(null)
      void queryClient.invalidateQueries({ queryKey: ['me', 'sessions'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal mencabut sesi perangkat.'),
  })

  const logoutAll = useMutation({
    mutationFn: () => securityApi.logoutAll(),
    onSuccess: () => {
      window.location.href = '/app/login'
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal keluar dari semua perangkat.'),
  })

  const rows: Session[] = sessions.data ?? []

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-semibold">Perangkat yang Masuk</CardTitle>
          <CardDescription className="text-xs">
            Daftar sesi aktif yang terhubung ke akun Anda. Cabut perangkat yang tidak Anda kenali.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {notice && <Alert>{notice}</Alert>}
          {sessions.isLoading && <Skeleton className="h-20 w-full" />}
          
          <div className="divide-y divide-border rounded-lg border border-border">
            {rows.map((row) => {
              const isMobile = /mobile|android|iphone|ipad/i.test(row.device_label || '')
              const DeviceIcon = isMobile ? Smartphone : Laptop

              return (
                <div key={row.id} className="flex flex-wrap items-center justify-between gap-3 p-3 text-xs">
                  <div className="flex items-center gap-3">
                    <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-secondary text-foreground">
                      <DeviceIcon className="h-4 w-4" />
                    </span>
                    <div>
                      <div className="flex items-center gap-2">
                        <p className="font-semibold text-foreground">
                          {row.device_label || 'Perangkat tidak dikenal'}
                        </p>
                        {row.current && (
                          <Badge variant="outline" className="border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                            Perangkat Ini
                          </Badge>
                        )}
                      </div>
                      <p className="text-[11px] text-muted-foreground">
                        Terakhir aktif {formatDateTime(row.last_seen_at)}
                      </p>
                    </div>
                  </div>

                  {!row.current && (
                    <Button
                      size="sm"
                      variant="outline"
                      className="gap-1 text-xs hover:border-red-500/30 hover:bg-red-500/10 hover:text-red-600"
                      onClick={() => setRevokeTarget(row)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      Cabut
                    </Button>
                  )}
                </div>
              )
            })}
          </div>

          {!sessions.isLoading && rows.length > 0 && (
            <div className="pt-1">
              <Button
                size="sm"
                variant="ghost"
                className="gap-1.5 text-xs text-muted-foreground hover:text-red-600"
                onClick={() => setConfirmLogoutAll(true)}
              >
                <LogOut className="h-3.5 w-3.5" />
                Keluar dari semua perangkat
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Modal Konfirmasi Cabut Sesi Spesifik */}
      <Modal
        open={Boolean(revokeTarget)}
        onClose={() => setRevokeTarget(null)}
        title="Cabut Sesi Perangkat?"
        description="Perangkat ini akan langsung dikeluarkan dari akun Anda."
        size="sm"
      >
        <div className="space-y-4 pt-1">
          <p className="text-xs text-muted-foreground">
            Apakah Anda yakin ingin mencabut sesi untuk <strong>{revokeTarget?.device_label || 'Perangkat ini'}</strong>? Sesi Anda pada browser saat ini tidak akan terpengaruh.
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setRevokeTarget(null)}>
              Batal
            </Button>
            <Button
              variant="destructive"
              size="sm"
              disabled={revoke.isPending}
              isLoading={revoke.isPending}
              onClick={() => revokeTarget && revoke.mutate(revokeTarget.id)}
            >
              Ya, Cabut Sesi
            </Button>
          </div>
        </div>
      </Modal>

      {/* Modal Konfirmasi Logout All */}
      <Modal
        open={confirmLogoutAll}
        onClose={() => setConfirmLogoutAll(false)}
        title="Keluar dari Semua Perangkat?"
        description="Tindakan ini akan mengakhiri semua sesi termasuk sesi saat ini."
        size="sm"
      >
        <div className="space-y-4 pt-1">
          <Alert variant="warning" className="text-xs">
            Setelah keluar dari semua perangkat, Anda akan diarahkan ke halaman login dan harus masuk kembali.
          </Alert>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setConfirmLogoutAll(false)}>
              Batal
            </Button>
            <Button
              variant="destructive"
              size="sm"
              disabled={logoutAll.isPending}
              isLoading={logoutAll.isPending}
              onClick={() => logoutAll.mutate()}
            >
              Keluar Semua
            </Button>
          </div>
        </div>
      </Modal>
    </>
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
      setNotice('Preferensi keluar otomatis disimpan.')
      void queryClient.invalidateQueries({ queryKey: ['auth', 'me'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan preferensi.'),
  })

  const value = me.data?.idle_warning ?? true

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">Keluar Otomatis (Auto Logout)</CardTitle>
        <CardDescription className="text-xs">
          Demi keamanan aset dan akun Anda, sistem secara otomatis mengakhiri sesi yang tidak aktif.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {notice && <Alert>{notice}</Alert>}
        <label className="flex items-start gap-2.5 text-xs text-foreground">
          <input
            type="checkbox"
            className="mt-0.5 h-4 w-4 rounded border-border text-foreground accent-foreground"
            checked={value}
            disabled={toggle.isPending}
            onChange={(event) => toggle.mutate(event.target.checked)}
          />
          <div className="space-y-0.5">
            <span className="font-medium">Tampilkan peringatan konfirmasi sebelum keluar otomatis</span>
            <p className="text-[11px] text-muted-foreground">
              Dialog interaktif "Anda masih di sini?" dengan hitung mundur 60 detik akan ditampilkan sebelum sesi benar-benar ditutup.
            </p>
          </div>
        </label>
      </CardContent>
    </Card>
  )
}
