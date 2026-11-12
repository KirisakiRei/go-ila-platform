import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Pagination, halamanSekitar } from '@/components/ui'

describe('halamanSekitar', () => {
  it('menampilkan semua nomor bila halaman sedikit', () => {
    expect(halamanSekitar(1, 5)).toEqual([1, 2, 3, 4, 5])
    expect(halamanSekitar(4, 7)).toEqual([1, 2, 3, 4, 5, 6, 7])
  })

  it('memangkas deretan panjang di sekitar halaman aktif', () => {
    expect(halamanSekitar(1, 20)).toEqual([1, 2, '…', 20])
    expect(halamanSekitar(2, 20)).toEqual([1, 2, 3, '…', 20])
    expect(halamanSekitar(10, 20)).toEqual([1, '…', 9, 10, 11, '…', 20])
    expect(halamanSekitar(20, 20)).toEqual([1, '…', 19, 20])
  })

  it('tidak pernah mengulang nomor halaman', () => {
    for (let aktif = 1; aktif <= 30; aktif += 1) {
      const nomor = halamanSekitar(aktif, 30).filter((item): item is number => item !== '…')
      expect(new Set(nomor).size).toBe(nomor.length)
      expect(nomor[0]).toBe(1)
      expect(nomor[nomor.length - 1]).toBe(30)
      expect(nomor).toContain(aktif)
    }
  })
})

describe('Pagination', () => {
  it('menyembunyikan kontrol bila hanya satu halaman', () => {
    render(<Pagination page={1} totalPages={1} shown={3} total={3} unit="peluang" onPageChange={() => {}} />)

    // Ringkasan tetap ada, tombol navigasi tidak.
    expect(screen.getByText('3 peluang')).toBeInTheDocument()
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  })

  it('menonaktifkan mundur di halaman pertama dan maju di halaman terakhir', () => {
    const { rerender } = render(
      <Pagination page={1} totalPages={3} shown={50} total={120} onPageChange={() => {}} />,
    )
    expect(screen.getByRole('button', { name: 'Halaman sebelumnya' })).toBeDisabled()

    rerender(<Pagination page={3} totalPages={3} shown={20} total={120} onPageChange={() => {}} />)
    expect(screen.getByRole('button', { name: 'Halaman berikutnya' })).toBeDisabled()
  })

  it('memanggil onPageChange dengan nomor yang benar', async () => {
    const pengguna = userEvent.setup()
    const ubah = vi.fn()

    render(<Pagination page={2} totalPages={5} shown={50} total={230} unit="peluang" onPageChange={ubah} />)

    await pengguna.click(screen.getByRole('button', { name: 'Halaman berikutnya' }))
    expect(ubah).toHaveBeenCalledWith(3)

    await pengguna.click(screen.getByRole('button', { name: 'Halaman sebelumnya' }))
    expect(ubah).toHaveBeenCalledWith(1)

    await pengguna.click(screen.getByRole('button', { name: '5' }))
    expect(ubah).toHaveBeenCalledWith(5)
  })

  it('menandai halaman aktif untuk pembaca layar', () => {
    render(<Pagination page={3} totalPages={5} shown={50} total={230} onPageChange={() => {}} />)

    const aktif = screen.getByRole('button', { name: '3' })
    expect(aktif).toHaveAttribute('aria-current', 'page')
    expect(aktif).toBeDisabled()
  })

  it('menampilkan ringkasan jumlah dan posisi halaman', () => {
    render(
      <Pagination page={2} totalPages={4} shown={50} total={175} unit="peluang" onPageChange={() => {}} />,
    )

    expect(screen.getByText('Menampilkan 50 dari 175 peluang · halaman 2 dari 4')).toBeInTheDocument()
  })

  it('menampilkan pemilih ukuran dan meneruskan pilihannya', async () => {
    const pengguna = userEvent.setup()
    const ubah = vi.fn()

    render(
      <Pagination
        page={1}
        totalPages={3}
        shown={20}
        total={60}
        unit="peluang"
        onPageChange={() => {}}
        pageSize={20}
        onPageSizeChange={ubah}
      />,
    )

    const pemilih = screen.getByLabelText('Jumlah baris per halaman')
    expect(pemilih).toHaveValue('20')
    // Pilihan yang lazim tersedia tanpa perlu mengetik.
    expect(screen.getByRole('option', { name: '10' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: '50' })).toBeInTheDocument()

    await pengguna.selectOptions(pemilih, '50')
    expect(ubah).toHaveBeenCalledWith(50)
  })

  it('tetap menampilkan pemilih ukuran walau hanya satu halaman', () => {
    render(
      <Pagination
        page={1}
        totalPages={1}
        shown={5}
        total={5}
        unit="peluang"
        onPageChange={() => {}}
        pageSize={20}
        onPageSizeChange={() => {}}
      />,
    )

    expect(screen.getByLabelText('Jumlah baris per halaman')).toBeInTheDocument()
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  })
})
