import { useQuery } from '@tanstack/react-query'
import { publicContentApi } from '@/lib/system-api'
import { formatRelativeTime } from '@/lib/format'
import { SignalOverviewCard, SignalOverviewCardSkeleton } from './SignalOverviewCards'
import { useInView } from './use-in-view'

/**
 * Ringkasan pantauan pasar untuk halaman publik.
 *
 * Menampilkan tiga instrumen utama. Bila belum ada sinyal, kartu tetap tampil
 * dengan keterangan "sedang dipantau" — bukan pesan galat dan bukan ruang
 * kosong, karena keadaan itu normal dan tidak boleh terlihat seperti halaman
 * yang rusak.
 */

/** Simbol yang dipantau, sekaligus urutan tampilannya. */
const SYMBOL_DIPANTAU = ['XAUUSD', 'EURUSD', 'NAS100'] as const

/** Grid tiga kolom di layar lebar, dua di tablet, satu di ponsel. */
const GRID = 'grid gap-4 sm:grid-cols-2 lg:grid-cols-3'

export function SignalOverviewGrid() {
  const [gridRef, gridTerlihat] = useInView<HTMLDivElement>()
  const preview = useQuery({
    queryKey: ['public', 'signal-preview'],
    queryFn: () => publicContentApi.signalPreview([...SYMBOL_DIPANTAU]),
    // Ringkasan boleh sedikit tertinggal; menahan permintaan ulang menahan
    // beban pada API publik yang dibatasi per alamat IP.
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  })

  if (preview.isLoading) {
    return (
      <div className={GRID} aria-busy="true" aria-label="Memuat ringkasan pantauan pasar">
        {SYMBOL_DIPANTAU.map((symbol) => (
          <SignalOverviewCardSkeleton key={symbol} />
        ))}
      </div>
    )
  }

  // Galat tidak ditampilkan sebagai pesan teknis. Pengunjung tetap melihat
  // tiga instrumen yang dipantau, hanya tanpa keterangan waktu.
  const data = preview.data
  const marketOpen = data?.market.open ?? false
  const marketLabel = data?.market.label ?? 'Jadwal pasar belum tersedia'

  const items =
    data?.items && data.items.length > 0
      ? data.items
      : SYMBOL_DIPANTAU.map((symbol) => ({
          symbol,
          display_name: '',
          category: 'forex' as const,
          has_signal: false,
        }))

  return (
    <div ref={gridRef} className="space-y-3">
      <div className={GRID}>
        {items.map((item, i) => (
          <div
            key={item.symbol}
            style={{
              opacity: gridTerlihat ? 1 : 0,
              transform: gridTerlihat ? 'translateY(0)' : 'translateY(6px)',
              transition: `opacity var(--motion-enter) var(--ease-out) ${i * 60}ms, transform var(--motion-enter) var(--ease-out) ${i * 60}ms`,
            }}
          >
            <SignalOverviewCard
              item={item}
              marketOpen={marketOpen}
              marketLabel={marketLabel}
            />
          </div>
        ))}
      </div>

      <p className="text-[11px] text-[#71717a]">
        {data
          ? `Diperbarui ${formatRelativeTime(data.generated_at)} · Khusus informasi arah pasar tanpa parameter eksekusi.`
          : 'Khusus informasi arah pasar tanpa parameter eksekusi.'}
      </p>
    </div>
  )
}
