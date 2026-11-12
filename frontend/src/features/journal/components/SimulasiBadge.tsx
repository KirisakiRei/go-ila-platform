import { Info } from 'lucide-react'
import { Badge } from '@/components/ui'

/**
 * SimulasiBadge menandai posisi yang tidak pernah dikirim ke broker.
 *
 * Posisi simulasi (dry run) dihitung dari harga pasar, bukan dari order nyata,
 * sehingga hasilnya tidak boleh dibaca sebagai uang sungguhan. Keterangan ini
 * diletakkan di tooltip karena penjelasan panjang di dalam baris tabel akan
 * mendesak angka yang justru ingin dibaca.
 */
export function SimulasiBadge({ className }: { className?: string }) {
  return (
    <span className="inline-flex items-center gap-1" title="Posisi simulasi (dry run): tidak dikirim ke broker, hasilnya dihitung dari harga pasar. Bukan uang sungguhan.">
      <Badge variant="muted" className={className}>
        Simulasi
      </Badge>
      <Info className="h-3 w-3 text-muted-foreground" aria-hidden="true" />
      <span className="sr-only">
        Posisi simulasi (dry run): tidak dikirim ke broker, hasilnya dihitung dari harga
        pasar. Bukan uang sungguhan.
      </span>
    </span>
  )
}
