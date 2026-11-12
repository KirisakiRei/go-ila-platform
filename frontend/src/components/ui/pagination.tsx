import { Button } from './button'
import { cn } from '@/lib/utils'
import { PILIHAN_UKURAN } from './use-paginasi-lokal'

type PaginationProps = {
  /** Halaman yang sedang tampil, dimulai dari 1. */
  page: number
  totalPages: number
  /** Jumlah baris yang sedang tampil, untuk keterangan ringkas. */
  shown: number
  total: number
  /** Unit yang dihitung, mis. "peluang". */
  unit?: string
  onPageChange: (page: number) => void
  /** Pilihan jumlah baris per halaman. Bila kosong, pemilih tidak tampil. */
  pageSize?: number
  pageSizeOptions?: number[]
  onPageSizeChange?: (size: number) => void
  disabled?: boolean
  className?: string
}

/**
 * Pagination menelusuri daftar yang lebih panjang dari satu layar.
 *
 * Nomor halaman disembunyikan bila hanya ada satu halaman: kontrol yang tidak
 * bisa dipakai hanya menambah keramaian tanpa menambah kemampuan.
 *
 * Daftar halaman dipangkas di sekitar halaman aktif (1 … 4 5 6 … 20). Deretan
 * angka penuh tidak muat di layar ponsel dan justru menyulitkan menemukan
 * halaman sekarang.
 */
export function Pagination({
  page,
  totalPages,
  shown,
  total,
  unit = 'baris',
  onPageChange,
  pageSize,
  pageSizeOptions = [...PILIHAN_UKURAN],
  onPageSizeChange,
  disabled = false,
  className,
}: PaginationProps) {
  const adaPemilihUkuran = Boolean(pageSize && onPageSizeChange)

  // Pemilih ukuran tetap ditampilkan walau hanya ada satu halaman: pengguna
  // yang ingin melihat 50 baris sekali jalan perlu memilihnya sebelum daftar
  // bertambah panjang.
  if (totalPages <= 1 && !adaPemilihUkuran) {
    return (
      <p className={cn('text-[11px] text-muted-foreground', className)}>
        {total} {unit}
      </p>
    )
  }

  if (totalPages <= 1) {
    return (
      <div className={cn('flex flex-wrap items-center justify-between gap-2', className)}>
        <p className="text-[11px] text-muted-foreground">
          Menampilkan {shown} dari {total} {unit}
        </p>
        {adaPemilihUkuran && (
          <PemilihUkuran
            value={pageSize as number}
            options={pageSizeOptions}
            disabled={disabled}
            onChange={onPageSizeChange as (size: number) => void}
          />
        )}
      </div>
    )
  }

  const nomor = halamanSekitar(page, totalPages)
  const bisaMundur = page > 1 && !disabled
  const bisaMaju = page < totalPages && !disabled

  return (
    <nav
      className={cn(
        'flex flex-wrap items-center justify-between gap-2 border-t border-border pt-2',
        className,
      )}
      aria-label="Navigasi halaman"
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <p className="text-[11px] text-muted-foreground">
          Menampilkan {shown} dari {total} {unit} · halaman {page} dari {totalPages}
        </p>
        {adaPemilihUkuran && (
          <PemilihUkuran
            value={pageSize as number}
            options={pageSizeOptions}
            disabled={disabled}
            onChange={onPageSizeChange as (size: number) => void}
          />
        )}
      </div>

      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="sm"
          disabled={!bisaMundur}
          onClick={() => onPageChange(page - 1)}
          aria-label="Halaman sebelumnya"
        >
          Sebelumnya
        </Button>

        {nomor.map((item, index) =>
          item === '…' ? (
            <span
              // Jeda bisa muncul dua kali; indeks dipakai agar kuncinya unik.
              key={`jeda-${index}`}
              className="px-1 text-[12px] text-muted-foreground"
              aria-hidden="true"
            >
              …
            </span>
          ) : (
            <button
              key={item}
              type="button"
              onClick={() => onPageChange(item)}
              disabled={disabled || item === page}
              aria-current={item === page ? 'page' : undefined}
              className={cn(
                'h-8 min-w-8 rounded-md px-2 text-[12.5px] font-medium tabular transition-colors',
                item === page
                  ? 'bg-primary text-primary-foreground'
                  : 'border border-input hover:bg-secondary',
              )}
            >
              {item}
            </button>
          ),
        )}

        <Button
          variant="outline"
          size="sm"
          disabled={!bisaMaju}
          onClick={() => onPageChange(page + 1)}
          aria-label="Halaman berikutnya"
        >
          Berikutnya
        </Button>
      </div>
    </nav>
  )
}

/**
 * PemilihUkuran memilih berapa baris tampil per halaman.
 *
 * Daftar panjang lebih mudah dipindai bila pengguna bisa memilih sekaligus 50
 * baris, dan lebih ringan bila hanya 10. Pilihannya dibuat kecil dan menyatu
 * dengan navigasi agar tidak menambah baris baru di halaman.
 */
function PemilihUkuran({
  value,
  options,
  disabled,
  onChange,
}: {
  value: number
  options: number[]
  disabled?: boolean
  onChange: (size: number) => void
}) {
  return (
    <label className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground">
      <span>Tampilkan</span>
      <select
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(Number(event.target.value))}
        aria-label="Jumlah baris per halaman"
        className="h-7 rounded-md border border-input bg-transparent px-1.5 text-[12px] tabular focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      >
        {options.map((ukuran) => (
          <option key={ukuran} value={ukuran}>
            {ukuran}
          </option>
        ))}
      </select>
      <span>baris</span>
    </label>
  )
}

/**
 * halamanSekitar memilih nomor halaman yang ditampilkan.
 *
 * Selalu menyertakan halaman pertama dan terakhir agar pengguna bisa melompat,
 * serta dua halaman di kiri-kanan halaman aktif untuk langkah dekat.
 */
export function halamanSekitar(aktif: number, total: number): (number | '…')[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, index) => index + 1)
  }

  const hasil: (number | '…')[] = [1]
  const mulai = Math.max(2, aktif - 1)
  const selesai = Math.min(total - 1, aktif + 1)

  if (mulai > 2) hasil.push('…')
  for (let i = mulai; i <= selesai; i += 1) hasil.push(i)
  if (selesai < total - 1) hasil.push('…')

  hasil.push(total)
  return hasil
}
