import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { accountFlowApi } from '@/lib/auth-api'
import { Button } from '@/components/ui/button'

/**
 * LegalReconsentGate meminta persetujuan ulang saat dokumen wajib berubah
 * versi. Selama belum disetujui, eksekusi dan pembayaran ditahan server,
 * jadi pemberitahuan ini tidak bisa diabaikan.
 */
export function LegalReconsentGate() {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)

  const status = useQuery({
    queryKey: ['me', 'legal'],
    queryFn: accountFlowApi.legal,
    // Cukup diperiksa sekali per sesi; dokumen jarang berubah.
    staleTime: 5 * 60 * 1000,
  })

  const accept = useMutation({
    mutationFn: (payload: { slug: string; version: number }) =>
      accountFlowApi.acceptLegal(payload.slug, payload.version),
    onSuccess: () => {
      setNotice(null)
      void queryClient.invalidateQueries({ queryKey: ['me', 'legal'] })
    },
    onError: (error) =>
      setNotice(error instanceof Error ? error.message : 'Persetujuan gagal disimpan.'),
  })

  const pending = status.data?.pending ?? []
  if (pending.length === 0) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 p-4 backdrop-blur-sm"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="legal-title"
    >
      <div className="w-full max-w-lg rounded-lg border border-border bg-card p-6 shadow-lg">
        <h2 id="legal-title" className="text-lg font-semibold tracking-tight">
          Dokumen diperbarui
        </h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Kami memperbarui dokumen berikut. Sinyal baru dan pembayaran ditahan sampai Anda
          menyetujuinya kembali.
        </p>

        <ul className="mt-4 flex flex-col gap-2">
          {pending.map((item) => (
            <li
              key={item.slug}
              className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
            >
              <div className="min-w-0">
                <p className="truncate text-[13px] font-medium">{item.title}</p>
                <Link
                  to={`/legal/${item.slug}`}
                  className="text-micro text-muted-foreground underline underline-offset-2"
                >
                  Baca versi {item.current_version}
                </Link>
              </div>
              <Button
                size="sm"
                disabled={accept.isPending}
                onClick={() => accept.mutate({ slug: item.slug, version: item.current_version })}
              >
                Setujui
              </Button>
            </li>
          ))}
        </ul>

        {notice && <p className="mt-3 text-[12px] text-destructive">{notice}</p>}

        <p className="mt-4 text-xs text-muted-foreground">
          Dengan menyetujui, Anda menerima versi terbaru dokumen tersebut.
        </p>
      </div>
    </div>
  )
}
