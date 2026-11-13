import { useEffect, useState } from 'react'
import { Check, Clock, Loader2, X } from 'lucide-react'
import { Button } from '@/components/ui'
import { useConfirmApproval, useDeclineApproval, useIssueApproval } from '@/lib/queries'
import type { Signal } from '@/types/api'
import { cn } from '@/lib/utils'

/**
 * sisaWaktu memecah selisih waktu menjadi menit dan detik yang aman ditampilkan.
 * Nilai negatif berarti batas waktu sudah lewat.
 */
function sisaWaktu(expiresAt: string, sekarang: number) {
  const selesai = new Date(expiresAt).getTime()
  if (!Number.isFinite(selesai)) return { habis: true, detik: 0 }
  const detik = Math.floor((selesai - sekarang) / 1000)
  return { habis: detik <= 0, detik: Math.max(0, detik) }
}

function formatSisa(detik: number) {
  const menit = Math.floor(detik / 60)
  const sisaDetik = detik % 60
  return `${menit}m ${String(sisaDetik).padStart(2, '0')}s`
}

/**
 * TombolEksekusi menampilkan pilihan keputusan untuk sinyal yang menunggu user.
 *
 * Dipakai pada sinyal berstatus READY_TO_ENTRY dan OPPORTUNITY. Hitung mundur
 * ditampilkan supaya user tahu batas waktunya, bukan menebak. Setelah lewat,
 * tombol dimatikan dengan alasan yang jelas — bukan dibiarkan gagal saat ditekan.
 */
export function TombolEksekusi({
  signal,
  onSelesai,
}: {
  signal: Signal
  onSelesai?: () => void
}) {
  const issue = useIssueApproval()
  const konfirmasi = useConfirmApproval()
  const tolak = useDeclineApproval()

  // Waktu sekarang diperbarui tiap detik hanya selama masih ada sisa waktu.
  const [sekarang, setSekarang] = useState(() => Date.now())
  useEffect(() => {
    const sisa = sisaWaktu(signal.expires_at, Date.now())
    if (sisa.habis) return
    const timer = window.setInterval(() => setSekarang(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [signal.expires_at])

  const sisa = sisaWaktu(signal.expires_at, sekarang)
  const sibuk = issue.isPending || konfirmasi.isPending || tolak.isPending
  const pesan = konfirmasi.data?.message ?? konfirmasi.error?.message ?? tolak.data?.message ?? tolak.error?.message

  const bisaDieksekusi = signal.can_confirm && !sisa.habis

  const eksekusi = async () => {
    // Token diterbitkan saat tombol ditekan, bukan saat halaman dibuka: token
    // berumur pendek, dan menerbitkannya lebih awal hanya membuatnya basi.
    const hasil = await issue.mutateAsync(signal.signal_id)
    if (!hasil.can_confirm) return
    await konfirmasi.mutateAsync(hasil.token)
    onSelesai?.()
  }

  const lewati = async () => {
    const hasil = await issue.mutateAsync(signal.signal_id)
    if (!hasil.token) return
    await tolak.mutateAsync(hasil.token)
    onSelesai?.()
  }

  const blokir = issue.data && !issue.data.can_confirm ? issue.data.blocked_reason : ''

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        {sisa.habis ? (
          <span className="inline-flex items-center gap-1.5 rounded-md border border-border bg-secondary/60 px-2.5 py-1 text-micro text-muted-foreground">
            <Clock className="h-3 w-3" />
            Batas waktu sudah lewat
          </span>
        ) : (
          <span className="inline-flex items-center gap-1.5 rounded-md border border-border bg-secondary/60 px-2.5 py-1 text-micro font-medium tabular-nums">
            <Clock className="h-3 w-3" />
            Berlaku {formatSisa(sisa.detik)} lagi
          </span>
        )}

        <Button
          size="sm"
          className="h-8 gap-1.5 text-xs"
          disabled={!bisaDieksekusi || sibuk}
          onClick={() => void eksekusi()}
        >
          {konfirmasi.isPending ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Check className="h-3.5 w-3.5" />
          )}
          Eksekusi
        </Button>

        <Button
          size="sm"
          variant="outline"
          className="h-8 gap-1.5 text-xs"
          disabled={sisa.habis || sibuk}
          onClick={() => void lewati()}
        >
          <X className="h-3.5 w-3.5" />
          Lewati
        </Button>
      </div>

      {blokir && (
        <p className="text-micro text-muted-foreground">{blokir}</p>
      )}
      {pesan && (
        <p
          className={cn(
            'text-micro',
            konfirmasi.isError || tolak.isError ? 'text-rose-600' : 'text-emerald-600',
          )}
        >
          {pesan}
        </p>
      )}
    </div>
  )
}
