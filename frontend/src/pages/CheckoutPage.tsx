import { useMutation } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { request } from '@/lib/http'
import { formatIDR } from '@/features/admin/lib/format'
import {
  Alert,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Label,
  PageHeader,
} from '@/components/ui'
import { useState } from 'react'

type CheckoutResult = {
  invoice_number: string
  total_idr: number
  payment_url?: string
  status: string
}

/**
 * CheckoutPage membuat tagihan dan mengarahkan ke halaman bayar gateway.
 *
 * Tombol ini tidak pernah menaikkan tier sendiri. Bila tagihan bermasalah,
 * tier tetap seperti sebelumnya sampai webhook resmi diterima.
 */
export default function CheckoutPage() {
  const [params] = useSearchParams()
  const planCode = params.get('paket') ?? 'basic'
  const [coupon, setCoupon] = useState('')
  const [useCredit, setUseCredit] = useState(false)

  const checkout = useMutation({
    mutationFn: () =>
      request<CheckoutResult>('/billing/checkout', {
        method: 'POST',
        body: { plan_code: planCode, coupon_code: coupon, use_credit: useCredit },
      }),
    onSuccess: (result) => {
      // Pengalihan ke gateway hanya bila URL memang ada.
      if (result.payment_url) {
        window.location.assign(result.payment_url)
      }
    },
  })

  return (
    <>
      <PageHeader
        title="Bayar langganan"
        description={`Paket ${planCode.toUpperCase()}. Tagihan berlaku 24 jam sejak dibuat.`}
      />

      {checkout.isError && (
        <Alert>{checkout.error instanceof Error ? checkout.error.message : 'Gagal membuat tagihan.'}</Alert>
      )}

      {checkout.data && !checkout.data.payment_url && (
        <Alert>
          Tagihan {checkout.data.invoice_number} sebesar {formatIDR(checkout.data.total_idr)} sudah lunas
          tanpa pembayaran. Langganan aktif setelah konfirmasi server.
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Rincian</CardTitle>
          <CardDescription>Kupon dan kredit diterapkan sebelum tagihan terbit.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <Label htmlFor="coupon">Kode kupon (opsional)</Label>
            <Input
              id="coupon"
              value={coupon}
              onChange={(event) => setCoupon(event.target.value.toUpperCase())}
              placeholder="HEMAT20"
            />
          </div>

          <label className="flex items-center gap-2 text-[12px]">
            <input
              type="checkbox"
              checked={useCredit}
              onChange={(event) => setUseCredit(event.target.checked)}
            />
            <span>Gunakan saldo kredit saya</span>
          </label>

          <div>
            <Button disabled={checkout.isPending} onClick={() => checkout.mutate()}>
              {checkout.isPending ? 'Membuat tagihan…' : 'Lanjut ke pembayaran'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </>
  )
}
