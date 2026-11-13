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
  mtf,
  ltf,
  className,
}: {
  htf?: string
  mtf?: string
  ltf?: string
  className?: string
}) {
  const tfs = [htf, mtf, ltf].map((t) => (t ?? '').trim()).filter(Boolean)
  if (tfs.length === 0) return null

  const label = tfs.join(' · ')

  return (
    <Badge variant="muted" className={className ? `text-[10px] ${className}` : 'text-[10px]'}>
      {label}
    </Badge>
  )
}
