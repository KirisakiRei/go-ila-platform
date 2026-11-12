import { describe, expect, it } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { usePaginasiLokal } from '@/components/ui'

/** Daftar 1..n sebagai data uji. */
const daftar = (n: number) => Array.from({ length: n }, (_, i) => i + 1)

describe('usePaginasiLokal', () => {
  it('memotong data sesuai ukuran halaman', () => {
    const { result } = renderHook(() => usePaginasiLokal(daftar(45), 20))

    expect(result.current.total).toBe(45)
    expect(result.current.totalPages).toBe(3)
    expect(result.current.baris).toEqual(daftar(20))
  })

  it('berpindah halaman dan menampilkan sisa baris terakhir', () => {
    const { result } = renderHook(() => usePaginasiLokal(daftar(45), 20))

    act(() => result.current.setPage(3))

    expect(result.current.page).toBe(3)
    // Halaman terakhir hanya berisi 5 baris (41..45).
    expect(result.current.baris).toEqual([41, 42, 43, 44, 45])
  })

  it('mengubah ukuran halaman mengembalikan ke halaman pertama', () => {
    const { result } = renderHook(() => usePaginasiLokal(daftar(100), 20))
    act(() => result.current.setPage(4))
    expect(result.current.page).toBe(4)

    act(() => result.current.setPageSize(50))

    expect(result.current.page).toBe(1)
    expect(result.current.totalPages).toBe(2)
    expect(result.current.baris).toHaveLength(50)
  })

  it('menjepit nomor halaman yang melewati batas data menyusut', () => {
    // Kasus nyata: pengguna di halaman 5 lalu filter dipersempit sampai hanya
    // tersisa 1 halaman. Tanpa penjepitan, tabel tampil kosong.
    const { result, rerender } = renderHook(
      ({ data }: { data: number[] }) => usePaginasiLokal(data, 10),
      { initialProps: { data: daftar(50) } },
    )
    act(() => result.current.setPage(5))
    expect(result.current.page).toBe(5)

    rerender({ data: daftar(8) })

    expect(result.current.totalPages).toBe(1)
    expect(result.current.page).toBe(1)
    expect(result.current.baris).toEqual(daftar(8))
  })

  it('menangani data kosong dan undefined tanpa galat', () => {
    const kosong = renderHook(() => usePaginasiLokal<number>([], 20))
    expect(kosong.result.current.page).toBe(1)
    expect(kosong.result.current.totalPages).toBe(0)
    expect(kosong.result.current.baris).toEqual([])

    const belum = renderHook(() => usePaginasiLokal<number>(undefined, 20))
    expect(belum.result.current.total).toBe(0)
    expect(belum.result.current.baris).toEqual([])
  })

  it('menolak nomor halaman di luar jangkauan', () => {
    const { result } = renderHook(() => usePaginasiLokal(daftar(30), 10))

    act(() => result.current.setPage(99))
    expect(result.current.page).toBe(3)

    act(() => result.current.setPage(-4))
    expect(result.current.page).toBe(1)
  })
})
