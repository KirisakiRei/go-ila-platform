import { useState, useMemo } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import {
  ArrowLeft,
  Check,
  CheckCircle2,
  Clock,
  Copy,
  CreditCard,
  QrCode,
  ShieldCheck,
  Sparkles,
  Tag,
  Wallet,
} from 'lucide-react'
import { request } from '@/lib/http'
import { publicPlanApi } from '@/lib/admin-api'
import { formatIDR } from '@/features/admin/lib/format'
import { cn } from '@/lib/utils'
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
import { formatPlanBenefits, getPlanAudience } from '@/pages/public/lib/plan-benefits'
import { CheckoutMobileView } from '@/pages/components/CheckoutMobileView'
import type { TimePair } from '@/types/api'

type CheckoutResult = {
  invoice_number: string
  total_idr: number
  payment_url?: string
  expires_at?: TimePair
  status: string
}

type PaymentMethod = {
  id: string
  name: string
  category: 'qris' | 'va'
  bank?: string
  accountNumber: string
  icon: typeof QrCode
  description: string
}

const PAYMENT_METHODS: PaymentMethod[] = [
  {
    id: 'qris',
    name: 'QRIS Instant (Semua Bank & E-Wallet)',
    category: 'qris',
    accountNumber: 'ID1020030040050',
    icon: QrCode,
    description: 'BCA, Mandiri, BRI, BNI, GoPay, OVO, Dana, LinkAja',
  },
  {
    id: 'va_bca',
    name: 'BCA Virtual Account',
    category: 'va',
    bank: 'BCA',
    accountNumber: '8277081299381204',
    icon: CreditCard,
    description: 'Verifikasi instan otomatis 24 jam',
  },
  {
    id: 'va_mandiri',
    name: 'Mandiri Virtual Account',
    category: 'va',
    bank: 'Mandiri',
    accountNumber: '8890812993812041',
    icon: CreditCard,
    description: 'Transfer via Livin by Mandiri atau ATM',
  },
  {
    id: 'va_bni',
    name: 'BNI Virtual Account',
    category: 'va',
    bank: 'BNI',
    accountNumber: '9888081299381204',
    icon: CreditCard,
    description: 'Transfer via BNI Mobile Banking atau ATM',
  },
  {
    id: 'va_bri',
    name: 'BRI Virtual Account (BRIVA)',
    category: 'va',
    bank: 'BRI',
    accountNumber: '1280081299381204',
    icon: CreditCard,
    description: 'Transfer via BRImo atau ATM BRI',
  },
]

export default function CheckoutPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const planCode = params.get('paket') ?? 'basic'

  const [selectedMethod, setSelectedMethod] = useState<string>('qris')
  const [coupon, setCoupon] = useState('')
  const [appliedCoupon, setAppliedCoupon] = useState<string | null>(null)
  const [couponDiscount, setCouponDiscount] = useState<number>(0)
  const [useCredit, setUseCredit] = useState(false)
  const [copied, setCopied] = useState(false)

  // Ambil katalog paket
  const plans = useQuery({ queryKey: ['public', 'plans'], queryFn: publicPlanApi.list })

  const activePlan = useMemo(() => {
    return (plans.data ?? []).find((p) => p.code === planCode) ?? null
  }, [plans.data, planCode])

  const benefits = activePlan ? formatPlanBenefits(activePlan) : []

  const checkout = useMutation({
    mutationFn: () =>
      request<CheckoutResult>('/billing/checkout', {
        method: 'POST',
        body: {
          plan_code: planCode,
          coupon_code: appliedCoupon ?? (coupon.trim() ? coupon.trim().toUpperCase() : undefined),
          use_credit: useCredit,
        },
      }),
    onSuccess: (result) => {
      if (result.payment_url) {
        window.location.assign(result.payment_url)
      }
    },
  })

  const basePrice = activePlan?.price_idr ?? 299000
  const finalPrice = Math.max(0, basePrice - couponDiscount)

  const handleApplyCoupon = () => {
    if (!coupon.trim()) return
    // Simulasi potongan kupon preview jika user memasukkan voucher
    setAppliedCoupon(coupon.trim().toUpperCase())
    setCouponDiscount(Math.round(basePrice * 0.1)) // 10% diskon preview
  }

  const handleCopy = (text: string) => {
    void navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const currentMethod = PAYMENT_METHODS.find((m) => m.id === selectedMethod) ?? PAYMENT_METHODS[0]!

  // Status tampilan mobile: form → pending → bukti lunas.
  const mobileMode = !checkout.data
    ? 'form'
    : checkout.data.status === 'paid' || checkout.data.status === 'success'
      ? 'receipt'
      : 'pending'

  return (
    <>
      <CheckoutMobileView
        mode={mobileMode}
        planCode={planCode}
        planName={activePlan?.name ?? 'Paket Langganan'}
        planAudience={
          activePlan ? getPlanAudience(activePlan) : 'Akses pemantauan dan eksekusi sinyal trading.'
        }
        benefits={benefits}
        plansLoading={plans.isLoading}
        methods={PAYMENT_METHODS}
        selectedMethod={selectedMethod}
        onSelectMethod={setSelectedMethod}
        currentMethod={currentMethod}
        copied={copied}
        onCopy={handleCopy}
        basePrice={basePrice}
        finalPrice={finalPrice}
        coupon={coupon}
        onCouponChange={setCoupon}
        appliedCoupon={appliedCoupon}
        couponDiscount={couponDiscount}
        onApplyCoupon={handleApplyCoupon}
        onRemoveCoupon={() => {
          setAppliedCoupon(null)
          setCouponDiscount(0)
          setCoupon('')
        }}
        useCredit={useCredit}
        onUseCreditChange={setUseCredit}
        isSubmitting={checkout.isPending}
        errorMessage={
          checkout.isError
            ? checkout.error instanceof Error
              ? checkout.error.message
              : 'Gagal membuat tagihan transaksi.'
            : null
        }
        onSubmit={() => checkout.mutate()}
        onBack={() => navigate('/langganan')}
        onOpenInvoices={() => navigate('/langganan')}
        onOpenDashboard={() => navigate('/dashboard')}
        invoice={
          checkout.data
            ? {
                number: checkout.data.invoice_number,
                total_idr: checkout.data.total_idr,
                status: checkout.data.status,
                expires_at: checkout.data.expires_at,
              }
            : null
        }
      />

      {/* Tampilan Desktop (>= lg) */}
      <div className="hidden flex-col gap-6 lg:flex">
        <PageHeader
          title="Checkout & Pembayaran"
          description="Lengkapi pesanan langganan paket Anda dan pilih metode pembayaran resmi yang aman."
          actions={
            <Button variant="outline" size="sm" onClick={() => navigate('/langganan')} className="gap-1.5 text-xs">
              <ArrowLeft className="h-3.5 w-3.5" />
              Kembali ke Pilihan Paket
            </Button>
          }
        />

        {checkout.isError && (
          <Alert variant="destructive">
            {checkout.error instanceof Error ? checkout.error.message : 'Gagal membuat tagihan transaksi.'}
          </Alert>
        )}

        {/* KONDISI SETELAH INVOICE TERBIT */}
        {checkout.data && !checkout.data.payment_url ? (
          <div className="mx-auto max-w-2xl space-y-6">
            <Card className="border-border">
              <CardHeader className="border-b border-border bg-secondary/30 pb-4">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div>
                    <div className="flex items-center gap-2">
                      <CheckCircle2 className="h-5 w-5 text-emerald-600" />
                      <CardTitle className="text-base font-bold">Tagihan Berhasil Dibuat</CardTitle>
                    </div>
                    <CardDescription className="mt-1 text-xs">
                      Nomor Invoice: <strong className="font-mono text-foreground">{checkout.data.invoice_number}</strong>
                    </CardDescription>
                  </div>
                  <Badge variant="outline" className="gap-1 text-xs">
                    <Clock className="h-3 w-3" />
                    Batas Waktu 24 Jam
                  </Badge>
                </div>
              </CardHeader>

              <CardContent className="space-y-6 pt-6">
                <div className="rounded-xl border border-border bg-card p-4 text-center space-y-3">
                  <p className="text-xs text-muted-foreground uppercase font-medium">Total Yang Harus Dibayar</p>
                  <p className="text-3xl font-extrabold tracking-tight text-foreground tabular-nums">
                    {formatIDR(checkout.data.total_idr)}
                  </p>
                  <p className="text-caption text-muted-foreground">
                    Pastikan nominal transfer sesuai persis hingga digit terakhir agar otomatis terverifikasi.
                  </p>
                </div>

                {/* Instruksi Metode Pembayaran Terpilih */}
                <div className="space-y-3 rounded-xl border border-border p-4 bg-secondary/20">
                  <div className="flex items-center justify-between">
                    <p className="text-xs font-semibold text-foreground">{currentMethod.name}</p>
                    <Badge variant="outline" className="text-micro uppercase">{currentMethod.category}</Badge>
                  </div>

                  <div className="flex items-center justify-between gap-2 rounded-lg border border-border bg-background p-3">
                    <div>
                      <p className="text-micro text-muted-foreground uppercase">Nomor Rekening / Virtual Account</p>
                      <p className="font-mono text-base font-bold tracking-wider text-foreground">
                        {currentMethod.accountNumber}
                      </p>
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      className="gap-1.5 text-xs font-semibold"
                      onClick={() => handleCopy(currentMethod.accountNumber)}
                    >
                      <Copy className="h-3.5 w-3.5" />
                      {copied ? 'Tersalin!' : 'Salin'}
                    </Button>
                  </div>

                  <div className="space-y-1.5 text-xs text-muted-foreground pt-1">
                    <p className="font-semibold text-foreground">Panduan Transfer:</p>
                    <ol className="list-decimal pl-4 space-y-1">
                      <li>Buka aplikasi m-Banking atau ATM bank Anda.</li>
                      <li>Pilih menu <strong>Transfer / Virtual Account</strong>.</li>
                      <li>Masukkan nomor Virtual Account di atas.</li>
                      <li>Periksa nama akun tertera dan nominal bayar, lalu konfirmasi pembayaran.</li>
                      <li>Paket Anda akan otomatis aktif dalam 1-3 menit setelah transfer diterima.</li>
                    </ol>
                  </div>
                </div>

                <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-4">
                  <Button variant="outline" size="sm" asChild>
                    <Link to="/langganan">Lihat Riwayat Tagihan</Link>
                  </Button>
                  <Button size="sm" asChild>
                    <Link to="/dashboard">Kembali ke Dashboard</Link>
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        ) : (
          /* FORM CHECKOUT E-COMMERCE STANDAR (2 KOLOM) */
          <div className="grid gap-6 lg:grid-cols-12">
            {/* KOLOM KIRI (7 Kolom): Rincian Produk & Pilihan Metode Pembayaran */}
            <div className="space-y-6 lg:col-span-7">
              {/* Kartu Produk */}
              <Card>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Sparkles className="h-4 w-4 text-foreground" />
                      <CardTitle className="text-sm font-semibold">Produk yang Dipilih</CardTitle>
                    </div>
                    <Badge variant="outline" className="text-xs uppercase">{planCode}</Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  {plans.isLoading ? (
                    <Skeleton className="h-24 w-full" />
                  ) : (
                    <div className="rounded-xl border border-border bg-secondary/20 p-4 space-y-3">
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <h4 className="font-bold text-base text-foreground">{activePlan?.name ?? 'Paket Langganan'}</h4>
                          <p className="text-xs text-muted-foreground leading-relaxed mt-0.5">
                            {activePlan ? getPlanAudience(activePlan) : 'Akses pemantauan dan eksekusi sinyal trading.'}
                          </p>
                        </div>
                        <div className="text-right shrink-0">
                          <p className="text-base font-bold tabular-nums text-foreground">{formatIDR(basePrice)}</p>
                          <p className="text-micro text-muted-foreground">/ 30 hari</p>
                        </div>
                      </div>

                      <div className="border-t border-border/80 pt-3">
                        <p className="text-micro font-semibold uppercase text-muted-foreground mb-1.5">Fitur Unggulan:</p>
                        <ul className="grid gap-1.5 sm:grid-cols-2 text-xs">
                          {benefits.map((b) => (
                            <li key={b.label} className="flex items-center gap-1.5 text-foreground">
                              <Check className="h-3 w-3 text-emerald-600 shrink-0" />
                              <span className="truncate">{b.label}</span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>

              {/* Pilihan Metode Pembayaran E-Commerce */}
              <Card>
                <CardHeader className="pb-3">
                  <div className="flex items-center gap-2">
                    <Wallet className="h-4 w-4 text-foreground" />
                    <CardTitle className="text-sm font-semibold">Pilih Metode Pembayaran</CardTitle>
                  </div>
                  <CardDescription className="text-xs">
                    Semua transaksi diproses aman dan terverifikasi otomatis.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-2.5">
                  {PAYMENT_METHODS.map((method) => {
                    const isSelected = selectedMethod === method.id
                    const Icon = method.icon

                    return (
                      <button
                        key={method.id}
                        type="button"
                        onClick={() => setSelectedMethod(method.id)}
                        className={cn(
                          'flex w-full items-center justify-between rounded-xl border p-3.5 text-left transition-all',
                          isSelected
                            ? 'border-foreground bg-secondary/60 shadow-sm'
                            : 'border-border bg-card hover:border-zinc-300',
                        )}
                      >
                        <div className="flex items-center gap-3">
                          <span
                            className={cn(
                              'flex h-9 w-9 items-center justify-center rounded-lg border text-foreground',
                              isSelected ? 'border-foreground bg-foreground text-background' : 'border-border bg-secondary',
                            )}
                          >
                            <Icon className="h-4 w-4" />
                          </span>
                          <div>
                            <p className="text-xs font-semibold text-foreground">{method.name}</p>
                            <p className="text-micro text-muted-foreground">{method.description}</p>
                          </div>
                        </div>

                        <div
                          className={cn(
                            'flex h-4 w-4 items-center justify-center rounded-full border',
                            isSelected ? 'border-foreground bg-foreground' : 'border-zinc-300',
                          )}
                        >
                          {isSelected && <span className="h-1.5 w-1.5 rounded-full bg-background" />}
                        </div>
                      </button>
                    )
                  })}
                </CardContent>
              </Card>

              {/* Voucher & Kode Diskon */}
              <Card>
                <CardHeader className="pb-3">
                  <div className="flex items-center gap-2">
                    <Tag className="h-4 w-4 text-foreground" />
                    <CardTitle className="text-sm font-semibold">Voucher &amp; Saldo Kredit</CardTitle>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex gap-2">
                    <div className="flex-1 space-y-1">
                      <Label htmlFor="coupon-input" className="sr-only">Kode Voucher</Label>
                      <Input
                        id="coupon-input"
                        value={coupon}
                        onChange={(e) => setCoupon(e.target.value.toUpperCase())}
                        placeholder="Masukkan kode promo (misal: HEMAT20)"
                        className="font-mono text-xs"
                      />
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={handleApplyCoupon}
                      disabled={!coupon.trim()}
                      className="text-xs font-semibold"
                    >
                      Terapkan
                    </Button>
                  </div>

                  {appliedCoupon && (
                    <div className="flex items-center justify-between rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-xs font-medium text-emerald-800 dark:text-emerald-300">
                      <span>Voucher <strong>{appliedCoupon}</strong> diterapkan!</span>
                      <button
                        type="button"
                        className="text-muted-foreground hover:text-foreground text-micro underline"
                        onClick={() => {
                          setAppliedCoupon(null)
                          setCouponDiscount(0)
                          setCoupon('')
                        }}
                      >
                        Hapus
                      </button>
                    </div>
                  )}

                  <label className="flex items-center gap-2 text-xs text-foreground cursor-pointer pt-1">
                    <input
                      type="checkbox"
                      checked={useCredit}
                      onChange={(e) => setUseCredit(e.target.checked)}
                      className="rounded border-zinc-300"
                    />
                    <span>Gunakan saldo kredit akun saya jika tersedia</span>
                  </label>
                </CardContent>
              </Card>
            </div>

            {/* KOLOM KANAN (5 Kolom): Ringkasan Pesanan (Order Summary) */}
            <div className="space-y-4 lg:col-span-5">
              <Card className="sticky top-20 border-border">
                <CardHeader className="border-b border-border pb-3">
                  <CardTitle className="text-sm font-bold">Ringkasan Tagihan</CardTitle>
                  <CardDescription className="text-xs">Rincian harga sebelum transaksi diproses.</CardDescription>
                </CardHeader>

                <CardContent className="space-y-4 pt-4">
                  <div className="space-y-2.5 text-xs">
                    <div className="flex justify-between text-muted-foreground">
                      <span>Harga Paket ({activePlan?.name ?? planCode})</span>
                      <span className="font-medium text-foreground tabular-nums">{formatIDR(basePrice)}</span>
                    </div>

                    {couponDiscount > 0 && (
                      <div className="flex justify-between text-emerald-600 font-medium">
                        <span>Potongan Voucher Promo</span>
                        <span className="tabular-nums">- {formatIDR(couponDiscount)}</span>
                      </div>
                    )}

                    {useCredit && (
                      <div className="flex justify-between text-muted-foreground">
                        <span>Potongan Saldo Kredit</span>
                        <span>Diterapkan saat checkout</span>
                      </div>
                    )}

                    <div className="flex justify-between text-muted-foreground">
                      <span>Pajak &amp; Biaya Layanan</span>
                      <span className="font-medium text-foreground">Termasuk</span>
                    </div>

                    <div className="border-t border-border pt-3 flex items-baseline justify-between">
                      <div>
                        <p className="font-bold text-sm text-foreground">Total Pembayaran</p>
                        <p className="text-micro text-muted-foreground">Masa aktif 30 hari</p>
                      </div>
                      <span className="text-2xl font-bold tracking-tight text-foreground tabular-nums">
                        {formatIDR(finalPrice)}
                      </span>
                    </div>
                  </div>

                  <div className="space-y-2 pt-2">
                    <Button
                      className="w-full text-xs font-semibold gap-1.5"
                      size="sm"
                      disabled={checkout.isPending}
                      onClick={() => checkout.mutate()}
                    >
                      {checkout.isPending ? 'Memproses Pesanan...' : 'Konfirmasi & Lanjut Bayar'}
                    </Button>

                    <p className="text-micro text-center text-muted-foreground leading-relaxed">
                      Dengan melanjutkan, Anda menyetujui Ketentuan Layanan dan Kebijakan Pengembalian Dana ILA Platform.
                    </p>
                  </div>

                  <div className="flex items-center justify-center gap-2 border-t border-border pt-3 text-micro text-muted-foreground">
                    <ShieldCheck className="h-3.5 w-3.5 text-emerald-600" />
                    <span>Enkripsi 256-bit aman &amp; terpercaya</span>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        )}
      </div>
    </>
  )
}
