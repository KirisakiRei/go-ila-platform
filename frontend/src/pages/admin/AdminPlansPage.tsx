import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminPlanApi } from '@/lib/admin-api'
import { formatIDR } from '@/features/admin/lib/format'
import { usePermissions } from '@/features/admin/hooks/use-permissions'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Input,
  Label,
  Modal,
  PageHeader,
  Skeleton,
} from '@/components/ui'

/**
 * AdminPlansPage mengubah harga dan fitur paket.
 *
 * Harga tidak boleh turun di bawah batas bawah: satu salah ketik tidak boleh
 * menjual paket berharga hampir gratis (plans/14 §6).
 */
export default function AdminPlansPage() {
  const queryClient = useQueryClient()
  const { can } = usePermissions()
  const { data, isLoading } = useQuery({ queryKey: ['admin', 'plans'], queryFn: adminPlanApi.list })

  const [editing, setEditing] = useState<string | null>(null)
  const [price, setPrice] = useState('')
  const [reason, setReason] = useState('')
  const [guard, setGuard] = useState('')
  const [notice, setNotice] = useState<string | null>(null)

  const save = useMutation({
    mutationFn: async (code: string) => {
      const value = Number(price.replace(/[^\d]/g, ''))
      return adminPlanApi.update(code, {
        name: data?.find((plan) => plan.code === code)?.name ?? code,
        price_idr: value,
        description: data?.find((plan) => plan.code === code)?.description ?? '',
        features: data?.find((plan) => plan.code === code)?.features ?? {},
        reason,
      })
    },
    onSuccess: () => {
      setNotice('Harga diperbarui. Invoice lama tidak berubah.')
      setEditing(null)
      setPrice('')
      setReason('')
      void queryClient.invalidateQueries({ queryKey: ['admin', 'plans'] })
      void queryClient.invalidateQueries({ queryKey: ['public', 'plans'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan.'),
  })

  const saveGuard = useMutation({
    mutationFn: (code: string) =>
      adminPlanApi.setGuard(code, {
        min_price_idr: Number(guard.replace(/[^\d]/g, '')),
        reason,
      }),
    onSuccess: () => {
      setNotice('Batas bawah harga diperbarui.')
      setGuard('')
      setReason('')
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan.'),
  })

  if (isLoading) return <Skeleton className="h-64 w-full" />
  if (!data || data.length === 0) {
    return <EmptyState title="Belum ada paket" description="Jalankan migrasi untuk mengisi paket bawaan." />
  }

  return (
    <>
      <PageHeader
        title="Paket & Harga"
        description="Harga di sini menjadi acuan invoice berikutnya. Invoice yang sudah terbit tidak berubah."
      />

      {notice && <Alert>{notice}</Alert>}

      <div className="grid gap-4 lg:grid-cols-3">
        {data.map((plan) => (
          <Card key={plan.code}>
            <CardHeader>
              <div className="flex items-center justify-between gap-2">
                <CardTitle>{plan.name}</CardTitle>
                <Badge variant={plan.active ? 'solid' : 'default'}>
                  {plan.active ? 'Dijual' : 'Nonaktif'}
                </Badge>
              </div>
              <CardDescription>{plan.description}</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              <p className="text-lg font-semibold tabular-nums">{formatIDR(plan.price_idr)}</p>

              <dl className="flex flex-col gap-1 text-[12px] text-muted-foreground">
                <div className="flex justify-between">
                  <dt>Batas akun</dt>
                  <dd>{plan.features.account_limit ?? '—'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt>Simbol</dt>
                  <dd>{plan.features.max_symbols ?? '—'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt>Eksekusi otomatis</dt>
                  <dd>{plan.features.auto_execute === 'true' ? 'Ya' : 'Tidak'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt>ILA 3.0</dt>
                  <dd>{plan.ila3_coming_soon ? 'Segera hadir' : 'Tersedia'}</dd>
                </div>
              </dl>

              {can('billing.plan') && (
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-2"
                  onClick={() => {
                    setEditing(plan.code)
                    setPrice(String(plan.price_idr))
                    setGuard('')
                    setReason('')
                  }}
                >
                  Ubah harga
                </Button>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <Modal
        open={Boolean(editing)}
        onClose={() => setEditing(null)}
        title={editing ? `Ubah Harga Paket ${data.find((p) => p.code === editing)?.name ?? editing}` : 'Ubah Harga'}
        description="Harga di sini menjadi acuan invoice berikutnya. Invoice yang sudah terbit tidak berubah."
        size="md"
      >
        {editing && (
          <div className="space-y-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor={`price-${editing}`}>Harga baru (IDR)</Label>
              <Input
                id={`price-${editing}`}
                value={price}
                inputMode="numeric"
                onChange={(event) => setPrice(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor={`guard-${editing}`}>Batas bawah harga</Label>
              <Input
                id={`guard-${editing}`}
                value={guard}
                inputMode="numeric"
                onChange={(event) => setGuard(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor={`reason-${editing}`}>Alasan perubahan (min. 10 karakter)</Label>
              <Input
                id={`reason-${editing}`}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
              />
            </div>
            <div className="flex justify-end gap-2 border-t border-border pt-4">
              <Button variant="outline" size="sm" type="button" onClick={() => setEditing(null)}>
                Batal
              </Button>
              {guard && (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={reason.trim().length < 10 || saveGuard.isPending}
                  onClick={() => saveGuard.mutate(editing)}
                >
                  {saveGuard.isPending ? 'Menyimpan…' : 'Simpan batas'}
                </Button>
              )}
              <Button
                size="sm"
                disabled={reason.trim().length < 10 || save.isPending}
                onClick={() => save.mutate(editing)}
              >
                {save.isPending ? 'Menyimpan…' : 'Simpan harga'}
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </>
  )
}
