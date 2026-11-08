import { useQuery } from '@tanstack/react-query'
import { publicContentApi } from '@/lib/system-api'

/**
 * usePageContent membaca isi halaman publik dari server.
 *
 * Halaman yang belum diisi atau belum diterbitkan admin menghasilkan keadaan
 * kosong, bukan isi cadangan di dalam kode: satu sumber kebenaran, yaitu
 * basis data yang dikelola admin.
 */
export function usePageContent(slug: string) {
  return useQuery({
    queryKey: ['public', 'page', slug],
    queryFn: () => publicContentApi.page(slug),
    // Isi halaman jarang berubah; menyimpannya sebentar membuat pindah
    // halaman terasa langsung tanpa permintaan ulang.
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}
