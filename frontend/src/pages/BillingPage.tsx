import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, ArrowRight, Check, Sparkles, Tag } from 'lucide-react'
import { request } from '@/lib/http'
import { publicPlanApi } from '@/lib/admin-api'
import { formatIDR, formatInvoice } from '@/features/admin/lib/format'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  EmptyState,
  Input,
  Label,
  Modal,
  PageHeader,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import { formatPlanBenefits, getPlanAudience } from '@/pages/public/lib/plan-benefits'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import type { TimePair } from '@/types/api'

type Invoice = {
  id: string
  number: string
  plan_code: string
  total_idr: number
  status: string
  created_at: TimePair
  paid_at?: TimePair
}

const STATUS_TEXT: Record<string, string> = {
  pending: 'Menunggu pembayaran',
  paid: 'Lunas',
  expired: 'Kedaluwarsa',
  cancelled: 'Dibatalkan',
  refunded: 'Dikembalikan',
}

export default function BillingPage() {
  const navigate = useNavigate()
  const { user } = useAuthContext()
  const queryClient = useQueryClient()

  const [modalRedeem, setModalRedeem] = useState(false)
  const [creditCode, setCreditCode] = useState('')
  const [redeemNotice, setRedeemNotice] = useState<string | null>(null)

  const plans = useQuery({ queryKey: ['public', 'plans'], queryFn: publicPlanApi.list })
  const invoices = useQuery({
    queryKey: ['billing', 'invoices'],
    queryFn: () => request<Invoice[]>('/billing/invoices'),
  })
  const subscription = useQuery({
    queryKey: ['billing', 'subscription'],
    queryFn: () =>
      request<{ plan_code: string; period_end: TimePair; status: string } | null>(
        '/billing/subscription',
      ),
  })

  const redeem = useMutation({
    mutationFn: () =>
      request<{ balance_idr: number }>('/billing/credit/redeem', {
        method: 'POST',
        body: { code: creditCode.trim().toUpperCase() },
      }),
    onSuccess: (data) => {
      setRedeemNotice(`Kode berhasil ditukar! Saldo kredit Anda: ${formatIDR(data.balance_idr)}.`)
      setCreditCode('')
      void queryClient.invalidateQueries({ queryKey: ['billing'] })
    },
    onError: (err) =>
      setRedeemNotice(err instanceof Error ? err.message : 'Kode kredit tidak valid atau sudah kedaluwarsa.'),
  })

  const currentTier = user?.tier ?? 'free'

  return (
    <>
      <PageHeader
        title="Paket & Langganan"
        description="Kelola paket langganan aktif, bandingkan benefit secara transparan, dan riwayat tagihan resmi."
        actions={
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => navigate('/dashboard')} className="gap-1.5 text-xs">
              <ArrowLeft className="h-3.5 w-3.5" />
              Kembali ke Dashboard
            </Button>
            <Button variant="outline" size="sm" onClick={() => setModalRedeem(true)} className="gap-1.5 text-xs">
              <Tag className="h-3.5 w-3.5" />
              Tukar Kode Kredit
            </Button>
          </div>
        }
      />

      {/* Langganan Aktif */}
      {subscription.data && (
        <Card className="border-border bg-gradient-to-r from-card to-secondary/30">
          <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between sm:p-5">
            <div className="flex items-center gap-3">
              <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-foreground text-background">
                <Sparkles className="h-5 w-5" />
              </span>
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-semibold tracking-tight">
                    Langganan Aktif: Paket {subscription.data.plan_code.toUpperCase()}
                  </h3>
                  <Badge variant="solid" className="text-[10px]">Aktif</Badge>
                </div>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  Masa aktif berlaku sampai dengan{' '}
                  <strong className="font-semibold text-foreground">
                    {formatInvoice(subscription.data.period_end.wib)} WIB
                  </strong>.
                </p>
              </div>
            </div>

            <Button size="sm" variant="outline" asChild className="gap-1 text-xs">
              <Link to="/langganan/bayar?paket=plus">
                Upgrade Paket
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Pilihan Paket Lengkap & Transparan */}
      <div className="space-y-4">
        <div>
          <h2 className="text-base font-semibold tracking-tight">Pilihan Paket Langganan</h2>
          <p className="text-xs text-muted-foreground">
            Semua paket berbayar memiliki masa aktif 30 hari. Harga sudah termasuk pajak sesuai ketentuan resmi.
          </p>
        </div>

        {plans.isLoading ? (
          <div className="grid gap-4 sm:grid-cols-3">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-96 w-full rounded-2xl" />
            ))}
          </div>
        ) : (
          <div className="grid gap-4 lg:grid-cols-3">
            {(plans.data ?? []).map((plan) => {
              const isCurrent = currentTier === plan.code
              const isPlus = plan.code === 'plus'
              const benefits = formatPlanBenefits(plan)

              return (
                <div
                  key={plan.code}
                  className={
                    isPlus
                      ? 'relative flex flex-col justify-between rounded-2xl border-2 border-foreground bg-card p-6 shadow-sm'
                      : 'relative flex flex-col justify-between rounded-2xl border border-border bg-card p-6 transition-colors hover:border-zinc-300'
                  }
                >
                  <div className="space-y-4">
                    <div className="flex items-center justify-between gap-2">
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="text-base font-bold text-foreground">{plan.name}</h3>
                          {plan.ila3_coming_soon && <Badge variant="outline">ILA 3.0 Segera</Badge>}
                        </div>
                        <p className="mt-1 text-xs text-muted-foreground leading-relaxed">
                          {getPlanAudience(plan)}
                        </p>
                      </div>

                      {isPlus && (
                        <span className="rounded-full bg-foreground px-2.5 py-0.5 text-[10px] font-semibold text-background">
                          Terlengkap
                        </span>
                      )}
                    </div>

                    <div className="border-t border-border pt-4">
                      <div className="flex items-baseline gap-1">
                        <span className="text-2xl font-bold tracking-tight text-foreground tabular-nums">
                          {formatIDR(plan.price_idr)}
                        </span>
                        <span className="text-xs text-muted-foreground">/ 30 hari</span>
                      </div>
                      <p className="mt-1 text-[11px] text-muted-foreground">
                        {plan.code === 'free' ? 'Akses selamanya tanpa biaya' : 'Pembayaran per bulan, batalkan kapan saja'}
                      </p>
                    </div>

                    {/* Benefit Checklist */}
                    <div className="space-y-2 border-t border-border pt-4">
                      <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                        Benefit Termasuk:
                      </p>
                      <ul className="space-y-2 text-xs">
                        {benefits.map((benefit) => (
                          <li key={benefit.label} className="flex items-start gap-2">
                            <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" />
                            <span className="text-foreground">{benefit.label}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  </div>

                  <div className="mt-6 border-t border-border pt-4">
                    {isCurrent ? (
                      <Button className="w-full" size="sm" variant="outline" disabled>
                        Paket Anda Saat Ini
                      </Button>
                    ) : plan.code === 'free' ? (
                      <Button className="w-full" size="sm" variant="outline" disabled>
                        Paket Dasar
                      </Button>
                    ) : (
                      <Button
                        className="w-full gap-1.5 font-semibold"
                        size="sm"
                        variant={isPlus ? 'default' : 'outline'}
                        asChild
                      >
                        <Link to={`/langganan/bayar?paket=${plan.code}`}>
                          Lanjut ke Checkout
                          <ArrowRight className="h-3.5 w-3.5" />
                        </Link>
                      </Button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Riwayat Tagihan */}
      <div className="space-y-3 pt-4">
        <div>
          <h2 className="text-base font-semibold tracking-tight">Riwayat Tagihan &amp; Invoice</h2>
          <p className="text-xs text-muted-foreground">
            Daftar seluruh invoice transaksi langganan akun Anda.
          </p>
        </div>

        <Card>
          <CardContent className="p-0">
            {invoices.isLoading && <Skeleton className="h-32 w-full" />}
            {invoices.data && invoices.data.length === 0 && (
              <div className="p-6">
                <EmptyState
                  title="Belum ada tagihan"
                  description="Riwayat tagihan akan otomatis muncul setelah Anda menyelesaikan checkout paket."
                />
              </div>
            )}
            {invoices.data && invoices.data.length > 0 && (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Nomor Tagihan</TableHead>
                      <TableHead>Paket</TableHead>
                      <TableHead className="text-right">Total Biaya</TableHead>
                      <TableHead>Status Pembayaran</TableHead>
                      <TableHead>Waktu Terbit (WIB)</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {invoices.data.map((row) => (
                      <TableRow key={row.id}>
                        <TableCell className="font-mono text-xs font-semibold">{row.number}</TableCell>
                        <TableCell className="text-xs font-medium uppercase">{row.plan_code}</TableCell>
                        <TableCell className="text-right text-xs font-semibold tabular-nums">
                          {formatIDR(row.total_idr)}
                        </TableCell>
                        <TableCell className="text-xs">
                          <Badge
                            variant={row.status === 'paid' ? 'solid' : row.status === 'pending' ? 'outline' : 'muted'}
                            className="text-[10.5px]"
                          >
                            {STATUS_TEXT[row.status] ?? row.status}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {formatInvoice(row.created_at.wib)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Modal Redeem Kode Kredit */}
      <Modal
        open={modalRedeem}
        onClose={() => {
          setModalRedeem(false)
          setRedeemNotice(null)
          setCreditCode('')
        }}
        title="Tukar Kode Kredit"
        description="Punya voucher atau kode kredit promosi? Masukkan di bawah untuk menambah saldo pemotong tagihan Anda."
        footer={
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setModalRedeem(false)
                setRedeemNotice(null)
                setCreditCode('')
              }}
            >
              Tutup
            </Button>
            <Button
              size="sm"
              disabled={creditCode.trim() === '' || redeem.isPending}
              onClick={() => redeem.mutate()}
            >
              {redeem.isPending ? 'Menukar...' : 'Tukar Sekarang'}
            </Button>
          </div>
        }
      >
        <div className="space-y-3 py-2">
          {redeemNotice && <Alert>{redeemNotice}</Alert>}
          <div className="space-y-1.5">
            <Label htmlFor="modal-credit-code">Kode Kredit Promosi</Label>
            <Input
              id="modal-credit-code"
              value={creditCode}
              placeholder="Contoh: ILA-KREDIT-2026"
              onChange={(e) => setCreditCode(e.target.value.toUpperCase())}
              className="font-mono text-xs"
            />
          </div>
          <p className="text-[11.5px] text-muted-foreground leading-relaxed">
            Saldo kredit yang Anda peroleh dapat digunakan untuk memotong total tagihan saat melakukan checkout paket.
          </p>
        </div>
      </Modal>
    </>
  )
}
