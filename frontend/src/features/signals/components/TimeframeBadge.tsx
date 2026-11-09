import { Badge } from '@/components/ui'

/**
 * TimeframeBadge menampilkan pasangan timeframe sumber sinyal, mis. "H4 · M15".
 *
 * Komponen ini sengaja tidak menampilkan apa pun bila engine belum mengirim
 * timeframe. Menampilkan "—" atau "UNKNOWN" akan membuat user mengira fitur
 * pemilihan timeframe sudah berlaku, padahal datanya belum ada.
 */
export function TimeframeBadge({
  htf,
  ltf,
  className,
}: {
  htf?: string
  ltf?: string
  className?: string
}) {
  const atas = (htf ?? '').trim()
  const bawah = (ltf ?? '').trim()
  if (!atas && !bawah) return null

  const label = atas && bawah ? `${atas} · ${bawah}` : atas || bawah

  return (
    <Badge variant="muted" className={className ? `text-[10px] ${className}` : 'text-[10px]'}>
      {label}
    </Badge>
  )
}
