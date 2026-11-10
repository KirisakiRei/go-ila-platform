import { useEffect, useMemo, useState } from 'react'
import {
  ArrowLeft,
  BadgeCheck,
  Check,
  CheckCircle2,
  Clock,
  Copy,
  Download,
  QrCode,
  ShieldCheck,
  Sparkles,
  Tag,
  Wallet,
} from 'lucide-react'
import { Button, Skeleton } from '@/components/ui'
import { formatIDR } from '@/features/admin/lib/format'
import { cn } from '@/lib/utils'
import type { PlanBenefit } from '@/pages/public/lib/plan-benefits'
import type { TimePair } from '@/types/api'

export type MobilePaymentMethod = {
  id: string
  name: string
  category: 'qris' | 'va'
  bank?: string
  accountNumber: string
  icon: typeof QrCode
  description: string
}

export type MobileCheckoutInvoice = {
  number: string
  total_idr: number
  status: string
  expires_at?: TimePair
}

type CheckoutMode = 'form' | 'pending' | 'receipt'

type CheckoutMobileViewProps = {
  mode: CheckoutMode
  planCode: string
  planName: string
  planAudience: string
  benefits: PlanBenefit[]
  plansLoading: boolean
  methods: MobilePaymentMethod[]
  selectedMethod: string
  onSelectMethod: (id: string) => void
  currentMethod: MobilePaymentMethod
  copied: boolean
  onCopy: (text: string) => void
  basePrice: number
  finalPrice: number
  coupon: string
  onCouponChange: (value: string) => void
  appliedCoupon: string | null
  couponDiscount: number
  onApplyCoupon: () => void
  onRemoveCoupon: () => void
  useCredit: boolean
  onUseCreditChange: (value: boolean) => void
  isSubmitting: boolean
  errorMessage: string | null
  onSubmit: () => void
  onBack: () => void
  onOpenInvoices: () => void
  onOpenDashboard: () => void
  invoice: MobileCheckoutInvoice | null
}

const FALLBACK_WINDOW_MS = 24 * 60 * 60 * 1000

function formatCountdown(ms: number): string {
  if (ms <= 0) return '00:00:00'
  const totalSeconds = Math.floor(ms / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return [hours, minutes, seconds].map((part) => String(part).padStart(2, '0')).join(':')
}

/** Mengunduh bukti pembayaran sederhana langsung di perangkat, tanpa endpoint tambahan. */
function downloadReceipt(invoice: MobileCheckoutInvoice, planName: string, methodName: string, paidAt?: string) {
  const lines = [
    'ILA PLATFORM — BUKTI PEMBAYARAN',
    '----------------------------------------',
    `No. Invoice      : ${invoice.number}`,
    `Paket            : ${planName}`,
    `Metode Bayar     : ${methodName}`,
    `Status           : LUNAS`,
    `Tanggal          : ${paidAt ?? '-'}`,
    `Total Dibayar    : ${formatIDR(invoice.total_idr)}`,
    '----------------------------------------',
    'Bukti pembayaran sah dan tercatat pada sistem ILA Platform.',
  ]
  const blob = new Blob([lines.join('\n')], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `bukti-${invoice.number}.txt`
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  URL.revokeObjectURL(url)
}

export function CheckoutMobileView({
  mode,
  planCode,
  planName,
  planAudience,
  benefits,
  plansLoading,
  methods,
  selectedMethod,
  onSelectMethod,
  currentMethod,
  copied,
  onCopy,
  basePrice,
  finalPrice,
  coupon,
  onCouponChange,
  appliedCoupon,
  couponDiscount,
  onApplyCoupon,
  onRemoveCoupon,
  useCredit,
  onUseCreditChange,
  isSubmitting,
  errorMessage,
  onSubmit,
  onBack,
  onOpenInvoices,
  onOpenDashboard,
  invoice,
}: CheckoutMobileViewProps) {
  const expiresAt = useMemo(() => {
    if (!invoice?.expires_at?.utc) return null
    const parsed = new Date(invoice.expires_at.utc).getTime()
    return Number.isFinite(parsed) ? parsed : null
  }, [invoice?.expires_at?.utc])

  const [remainingMs, setRemainingMs] = useState(() =>
    expiresAt ? Math.max(0, expiresAt - Date.now()) : FALLBACK_WINDOW_MS,
  )

  useEffect(() => {
    if (!expiresAt) {
      setRemainingMs(FALLBACK_WINDOW_MS)
      return
    }
    setRemainingMs(Math.max(0, expiresAt - Date.now()))
    const timer = window.setInterval(() => {
      setRemainingMs(Math.max(0, expiresAt - Date.now()))
    }, 1000)
    return () => window.clearInterval(timer)
  }, [expiresAt])

  const isVoucherApplied = Boolean(appliedCoupon)

  // Tagihan lunas tanpa gateway (mis. ditutup saldo kredit) tercatat pada saat
  // halaman bukti dirender, sehingga waktu lokal WIB ini dipakai sebagai waktu bayar.
  const paidAtLabel = useMemo(
    () =>
      new Intl.DateTimeFormat('id-ID', {
        timeZone: 'Asia/Jakarta',
        day: '2-digit',
        month: 'short',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }).format(new Date()),
    [],
  )

  return (
    <div className="flex flex-col gap-4 pb-2 lg:hidden">
      {/* Header */}
      <div className="flex items-start gap-3">
        <button
          type="button"
          onClick={onBack}
          aria-label="Kembali"
          className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-border bg-card text-foreground active:scale-95"
        >
          <ArrowLeft className="h-4 w-4" />
        </button>
        <div className="min-w-0">
          <h1 className="text-lg font-bold tracking-tight text-foreground">
            {mode === 'receipt' ? 'Status Pembayaran' : mode === 'pending' ? 'Detail Pembayaran' : 'Ringkasan Tagihan'}
          </h1>
          <p className="mt-0.5 text-[11.5px] leading-relaxed text-muted-foreground">
            {mode === 'receipt'
              ? 'Konfirmasi pembayaran langganan Anda.'
              : mode === 'pending'
                ? 'Selesaikan pembayaran sebelum batas waktu berakhir.'
                : 'Periksa kembali pesanan sebelum melanjutkan.'}
          </p>
        </div>
      </div>

      {errorMessage && (
        <div className="rounded-xl border border-destructive/40 bg-destructive/5 px-3 py-2.5 text-[11.5px] font-medium text-destructive">
          {errorMessage}
        </div>
      )}

      {mode === 'form' && (
        <>
          {/* Kartu paket */}
          <div className="rounded-2xl border border-border bg-card p-4 shadow-xs">
            {plansLoading ? (
              <Skeleton className="h-28 w-full rounded-xl" />
            ) : (
              <>
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-foreground text-background">
                      <Sparkles className="h-4 w-4" />
                    </span>
                    <div>
                      <p className="text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
                        Paket Dipilih
                      </p>
                      <h2 className="text-sm font-bold tracking-tight text-foreground">{planName}</h2>
                    </div>
                  </div>
                  <span className="rounded-md border border-border px-2 py-0.5 text-[9.5px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {planCode}
                  </span>
                </div>

                <p className="mt-2 text-[11.5px] leading-relaxed text-muted-foreground">{planAudience}</p>

                <div className="mt-3 flex items-baseline justify-between gap-2 border-t border-border/70 pt-3">
                  <span className="text-[11px] text-muted-foreground">Masa aktif 30 hari</span>
                  <span className="text-base font-bold tabular-nums text-foreground">{formatIDR(basePrice)}</span>
                </div>

                {benefits.length > 0 && (
                  <ul className="mt-3 grid grid-cols-1 gap-1.5 border-t border-border/70 pt-3">
                    {benefits.map((benefit) => (
                      <li key={benefit.label} className="flex items-start gap-2 text-[11.5px] text-foreground">
                        <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" />
                        <span className="leading-snug">{benefit.label}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </>
            )}
          </div>

          {/* Metode pembayaran */}
          <div className="rounded-2xl border border-border bg-card shadow-xs">
            <div className="flex items-center gap-2 border-b border-border px-3.5 py-3">
              <Wallet className="h-4 w-4 text-muted-foreground" />
              <h2 className="text-[13px] font-bold tracking-tight text-foreground">Metode Pembayaran</h2>
            </div>
            <div className="space-y-2 p-3.5">
              {methods.map((method) => {
                const isSelected = selectedMethod === method.id
                const Icon = method.icon
                return (
                  <button
                    key={method.id}
                    type="button"
                    onClick={() => onSelectMethod(method.id)}
                    aria-pressed={isSelected}
                    className={cn(
                      'flex w-full items-center justify-between gap-2 rounded-xl border p-3 text-left transition-all active:scale-[0.99]',
                      isSelected ? 'border-foreground bg-secondary/60' : 'border-border bg-card',
                    )}
                  >
                    <div className="flex min-w-0 items-center gap-2.5">
                      <span
                        className={cn(
                          'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border',
                          isSelected
                            ? 'border-foreground bg-foreground text-background'
                            : 'border-border bg-secondary text-foreground',
                        )}
                      >
                        <Icon className="h-4 w-4" />
                      </span>
                      <div className="min-w-0">
                        <p className="truncate text-[12px] font-semibold text-foreground">{method.name}</p>
                        <p className="truncate text-[10.5px] text-muted-foreground">{method.description}</p>
                      </div>
                    </div>
                    <span
                      className={cn(
                        'flex h-4 w-4 shrink-0 items-center justify-center rounded-full border',
                        isSelected ? 'border-foreground bg-foreground' : 'border-zinc-300',
                      )}
                    >
                      {isSelected && <span className="h-1.5 w-1.5 rounded-full bg-background" />}
                    </span>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Voucher & kredit */}
          <div className="rounded-2xl border border-border bg-card shadow-xs">
            <div className="flex items-center gap-2 border-b border-border px-3.5 py-3">
              <Tag className="h-4 w-4 text-muted-foreground" />
              <h2 className="text-[13px] font-bold tracking-tight text-foreground">Voucher &amp; Kredit</h2>
            </div>
            <div className="space-y-3 p-3.5">
              <div className="flex gap-2">
                <input
                  value={coupon}
                  onChange={(event) => onCouponChange(event.target.value.toUpperCase())}
                  placeholder="Kode promo"
                  aria-label="Kode voucher"
                  className="min-w-0 flex-1 rounded-lg border border-input bg-background px-3 py-2 font-mono text-[11.5px] uppercase outline-none focus-visible:ring-2 focus-visible:ring-ring"
                />
                <Button
                  size="sm"
                  variant="outline"
                  onClick={onApplyCoupon}
                  disabled={!coupon.trim() || isVoucherApplied}
                  className="shrink-0 text-[11px] font-semibold"
                >
                  Terapkan
                </Button>
              </div>

              {isVoucherApplied && (
                <div className="flex items-center justify-between rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-2 text-[11px] font-medium text-emerald-800 dark:text-emerald-300">
                  <span className="truncate">Voucher {appliedCoupon} diterapkan</span>
                  <button type="button" onClick={onRemoveCoupon} className="ml-2 shrink-0 underline">
                    Hapus
                  </button>
                </div>
              )}

              <label className="flex items-start gap-2 text-[11.5px] text-foreground">
                <input
                  type="checkbox"
                  checked={useCredit}
                  onChange={(event) => onUseCreditChange(event.target.checked)}
                  className="mt-0.5 rounded border-zinc-300"
                />
                <span className="leading-snug">Gunakan saldo kredit akun saya jika tersedia</span>
              </label>
            </div>
          </div>

          {/* Rincian harga */}
          <div className="rounded-2xl border border-border bg-card p-3.5 shadow-xs">
            <h2 className="text-[13px] font-bold tracking-tight text-foreground">Rincian Harga</h2>
            <div className="mt-2.5 space-y-2 text-[11.5px]">
              <div className="flex justify-between gap-2">
                <span className="text-muted-foreground">Harga paket</span>
                <span className="font-medium tabular-nums text-foreground">{formatIDR(basePrice)}</span>
              </div>
              {couponDiscount > 0 && (
                <div className="flex justify-between gap-2 text-emerald-700 dark:text-emerald-400">
                  <span>Diskon voucher</span>
                  <span className="font-semibold tabular-nums">− {formatIDR(couponDiscount)}</span>
                </div>
              )}
              {useCredit && (
                <div className="flex justify-between gap-2">
                  <span className="text-muted-foreground">Saldo kredit</span>
                  <span className="text-muted-foreground">Diterapkan saat checkout</span>
                </div>
              )}
              <div className="flex justify-between gap-2">
                <span className="text-muted-foreground">Pajak &amp; biaya layanan</span>
                <span className="font-medium text-foreground">Termasuk</span>
              </div>
              <div className="flex items-baseline justify-between gap-2 border-t border-border pt-2.5">
                <span className="text-[12.5px] font-bold text-foreground">Total Pembayaran</span>
                <span className="text-lg font-bold tabular-nums text-foreground">{formatIDR(finalPrice)}</span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2 rounded-xl border border-border bg-secondary/40 px-3 py-2.5">
            <ShieldCheck className="h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
            <p className="text-[10.5px] leading-relaxed text-muted-foreground">
              Transaksi terenkripsi 256-bit. Dana dikembalikan penuh bila paket gagal diaktifkan.
            </p>
          </div>

          {/* Bilah aksi tetap */}
          <div className="sticky bottom-20 z-20 -mx-4 mt-1 border-t border-border bg-card/95 px-4 py-3 backdrop-blur-xl">
            <div className="flex items-center gap-3">
              <div className="min-w-0">
                <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Total Bayar</p>
                <p className="text-base font-bold tabular-nums text-foreground">{formatIDR(finalPrice)}</p>
              </div>
              <Button
                className="ml-auto h-10 flex-1 gap-1.5 text-xs font-semibold"
                size="sm"
                disabled={isSubmitting}
                onClick={onSubmit}
              >
                {isSubmitting ? 'Memproses…' : 'Bayar Sekarang'}
              </Button>
            </div>
            <p className="mt-1.5 text-center text-[10px] text-muted-foreground">
              Dengan melanjutkan, Anda menyetujui Ketentuan Layanan ILA Platform.
            </p>
          </div>
        </>
      )}

      {mode === 'pending' && invoice && (
        <>
          {/* Banner menunggu */}
          <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-3.5">
            <div className="flex items-start gap-2.5">
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-amber-500/40 bg-background/70 text-amber-700 dark:text-amber-400">
                <Clock className="h-4 w-4" />
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-[12.5px] font-bold text-amber-800 dark:text-amber-300">Menunggu Pembayaran</p>
                <p className="mt-0.5 text-[11px] leading-relaxed text-amber-800/80 dark:text-amber-300/80">
                  Selesaikan pembayaran agar paket langsung aktif otomatis.
                </p>
                <div className="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-amber-500/30 bg-background/70 px-2 py-1">
                  <Clock className="h-3 w-3 text-amber-700 dark:text-amber-400" />
                  <span className="text-[10.5px] font-medium text-amber-800 dark:text-amber-300">
                    Sisa waktu
                  </span>
                  <span className="font-mono text-[11.5px] font-bold tabular-nums text-amber-900 dark:text-amber-200">
                    {formatCountdown(remainingMs)}
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* Nominal */}
          <div className="rounded-2xl border border-border bg-card p-4 text-center shadow-xs">
            <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
              Total Yang Harus Dibayar
            </p>
            <p className="mt-1 text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {formatIDR(invoice.total_idr)}
            </p>
            <p className="mt-1 text-[10.5px] leading-relaxed text-muted-foreground">
              Nomor invoice <span className="font-mono font-semibold text-foreground">{invoice.number}</span>
            </p>
          </div>

          {/* Instrumen pembayaran */}
          <div className="rounded-2xl border border-border bg-card p-3.5 shadow-xs">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-secondary text-foreground">
                  {currentMethod.category === 'qris' ? (
                    <QrCode className="h-4 w-4" />
                  ) : (
                    <Wallet className="h-4 w-4" />
                  )}
                </span>
                <p className="text-[12px] font-semibold text-foreground">{currentMethod.name}</p>
              </div>
              <span className="rounded-md border border-border px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-muted-foreground">
                {currentMethod.category}
              </span>
            </div>

            <div className="mt-3 rounded-xl border border-border bg-secondary/30 p-3">
              <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {currentMethod.category === 'qris' ? 'Kode Referensi QRIS' : `Nomor Virtual Account ${currentMethod.bank ?? ''}`}
              </p>
              <div className="mt-1 flex items-center justify-between gap-2">
                <p className="truncate font-mono text-[15px] font-bold tracking-wide text-foreground">
                  {currentMethod.accountNumber}
                </p>
                <Button
                  size="sm"
                  variant="outline"
                  className="shrink-0 gap-1 text-[10.5px] font-semibold"
                  onClick={() => onCopy(currentMethod.accountNumber)}
                >
                  {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                  {copied ? 'Tersalin' : 'Salin'}
                </Button>
              </div>
              <p className="mt-1.5 text-[10px] leading-relaxed text-muted-foreground">
                {currentMethod.category === 'qris'
                  ? 'Pindai kode QRIS pada aplikasi e-wallet atau m-banking Anda, lalu masukkan nominal persis.'
                  : 'Pastikan nama penerima sesuai dan nominal transfer sama persis hingga digit terakhir.'}
              </p>
            </div>
          </div>

          {/* Panduan */}
          <div className="rounded-2xl border border-border bg-card p-3.5 shadow-xs">
            <p className="text-[12px] font-bold tracking-tight text-foreground">Panduan Pembayaran</p>
            <ol className="mt-2.5 space-y-2.5">
              {[
                'Buka aplikasi m-banking atau e-wallet pilihan Anda.',
                currentMethod.category === 'qris'
                  ? 'Pilih menu bayar QRIS lalu pindai kode yang tertera.'
                  : 'Pilih menu Transfer ke Virtual Account, lalu masukkan nomor di atas.',
                'Masukkan nominal sesuai total tagihan hingga digit terakhir.',
                'Konfirmasi pembayaran; paket aktif otomatis dalam 1–3 menit.',
              ].map((step, index) => (
                <li key={step} className="flex items-start gap-2.5">
                  <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-border bg-secondary text-[10px] font-bold text-foreground">
                    {index + 1}
                  </span>
                  <span className="text-[11.5px] leading-snug text-muted-foreground">{step}</span>
                </li>
              ))}
            </ol>
          </div>

          {/* Aksi tetap */}
          <div className="sticky bottom-20 z-20 -mx-4 mt-1 border-t border-border bg-card/95 px-4 py-3 backdrop-blur-xl">
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                className="flex-1 gap-1 text-[11px] font-semibold"
                onClick={() => onCopy(currentMethod.accountNumber)}
              >
                <Copy className="h-3.5 w-3.5" />
                Salin Nomor
              </Button>
              <Button
                size="sm"
                className="flex-1 gap-1 text-[11px] font-semibold"
                onClick={onOpenInvoices}
              >
                <BadgeCheck className="h-3.5 w-3.5" />
                Cek Status
              </Button>
            </div>
            <p className="mt-1.5 text-center text-[10px] text-muted-foreground">
              Status diperbarui otomatis setelah pembayaran diterima.
            </p>
          </div>
        </>
      )}

      {mode === 'receipt' && invoice && (
        <>
          {/* Lencana sukses */}
          <div className="flex flex-col items-center rounded-2xl border border-border bg-card px-4 py-6 text-center shadow-xs">
            <span className="flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/12 text-emerald-600 dark:text-emerald-400">
              <CheckCircle2 className="h-9 w-9" strokeWidth={2.2} />
            </span>
            <h2 className="mt-3 text-[17px] font-bold tracking-tight text-foreground">Pembayaran Berhasil</h2>
            <p className="mt-1 max-w-[260px] text-[11.5px] leading-relaxed text-muted-foreground">
              Terima kasih, pembayaran Anda telah kami terima dan paket langsung aktif.
            </p>
            <p className="mt-3 text-2xl font-bold tracking-tight tabular-nums text-foreground">
              {formatIDR(invoice.total_idr)}
            </p>
            <span className="mt-2 inline-flex items-center gap-1 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-400">
              <Check className="h-3 w-3" />
              Lunas
            </span>
          </div>

          {/* Rincian transaksi */}
          <div className="rounded-2xl border border-border bg-card p-3.5 shadow-xs">
            <h2 className="text-[12.5px] font-bold tracking-tight text-foreground">Rincian Transaksi</h2>
            <dl className="mt-2.5 space-y-2 text-[11.5px]">
              <div className="flex items-start justify-between gap-3">
                <dt className="text-muted-foreground">No. Invoice</dt>
                <dd className="flex items-center gap-1 font-mono font-semibold text-foreground">
                  {invoice.number}
                  <button
                    type="button"
                    onClick={() => onCopy(invoice.number)}
                    aria-label="Salin nomor invoice"
                    className="text-muted-foreground active:scale-95"
                  >
                    {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
                  </button>
                </dd>
              </div>
              <div className="flex items-start justify-between gap-3">
                <dt className="text-muted-foreground">Metode Pembayaran</dt>
                <dd className="text-right font-medium text-foreground">{currentMethod.name}</dd>
              </div>
              <div className="flex items-start justify-between gap-3">
                <dt className="text-muted-foreground">Tanggal &amp; Waktu</dt>
                <dd className="text-right font-medium text-foreground">{paidAtLabel} WIB</dd>
              </div>
            </dl>

            <div className="mt-3 rounded-xl border border-border bg-secondary/30 p-3">
              <div className="flex items-center justify-between gap-2">
                <span className="text-[11.5px] font-semibold text-foreground">{planName}</span>
                <span className="text-[11.5px] tabular-nums text-muted-foreground">{formatIDR(invoice.total_idr)}</span>
              </div>
              <div className="mt-2 space-y-1.5 border-t border-border/70 pt-2 text-[11px]">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Subtotal</span>
                  <span className="tabular-nums text-foreground">{formatIDR(invoice.total_idr)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Pajak</span>
                  <span className="text-foreground">Termasuk</span>
                </div>
                <div className="flex items-baseline justify-between border-t border-border/70 pt-1.5">
                  <span className="font-bold text-foreground">Total Dibayar</span>
                  <span className="font-bold tabular-nums text-foreground">{formatIDR(invoice.total_idr)}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Catatan bukti */}
          <div className="rounded-2xl border border-dashed border-border bg-card p-3.5">
            <div className="flex items-start gap-2">
              <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
              <p className="text-[10.5px] leading-relaxed text-muted-foreground">
                Bukti pembayaran sah telah tercatat pada sistem ILA Platform dan dapat diunduh untuk keperluan
                administrasi Anda.
              </p>
            </div>
          </div>

          {/* Aksi */}
          <div className="flex flex-col gap-2">
            <Button className="w-full text-xs font-semibold" size="sm" onClick={onOpenDashboard}>
              Buka Dashboard
            </Button>
            <Button
              variant="outline"
              className="w-full gap-1.5 text-xs font-semibold"
              size="sm"
              onClick={() => downloadReceipt(invoice, planName, currentMethod.name, paidAtLabel)}
            >
              <Download className="h-3.5 w-3.5" />
              Unduh Bukti
            </Button>
          </div>
        </>
      )}
    </div>
  )
}
