import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import PositionsPage from '@/pages/PositionsPage'
import { tradingApi } from '@/lib/api'
import type { JournalResponse } from '@/types/api'

/**
 * Tes ini mengunci perilaku filter halaman jurnal.
 *
 * Yang ingin dijaga: tanggal bukan sekadar pilihan ekspor. Sebelumnya tanggal
 * hanya ada di bagian bawah dan tidak menyaring daftar, sehingga pengguna
 * mengira daftarnya sudah tersaring padahal hanya berkas CSV yang berubah.
 */

function jurnalKosong(): JournalResponse {
  return {
    positions: [],
    skipped: [],
    skipped_meta: { page: 1, per_page: 100, total: 0, total_pages: 0, has_more: false },
    stats: {
      total_positions: 0,
      open_positions: 0,
      closed_tp: 0,
      closed_sl: 0,
      closed_manual: 0,
      untracked: 0,
      wins: 0,
      losses: 0,
      win_rate: 0,
      total_r: 0,
      average_r: 0,
      total_profit: 0,
      dry_run_positions: 0,
    },
    filters: {},
  }
}

function bungkus() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <PositionsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Filter jurnal eksekusi', () => {
  it('mengirim periode hari saat pengguna belum memilih tanggal', async () => {
    const journal = vi.spyOn(tradingApi, 'journal').mockResolvedValue(jurnalKosong())

    bungkus()

    await waitFor(() => expect(journal).toHaveBeenCalled())
    const kiriman = journal.mock.calls[0]?.[0]
    expect(kiriman?.days).toBe(30)
    expect(kiriman?.from).toBeUndefined()
    expect(kiriman?.to).toBeUndefined()
  })

  it('menyaring daftar memakai tanggal, bukan hanya mengekspornya', async () => {
    const journal = vi.spyOn(tradingApi, 'journal').mockResolvedValue(jurnalKosong())
    const pengguna = userEvent.setup()

    bungkus()
    await waitFor(() => expect(journal).toHaveBeenCalled())

    const dari = screen.getByLabelText('Dari tanggal')
    await pengguna.type(dari, '2026-11-01')

    // Permintaan berikutnya harus membawa rentang tanggal, bukan `days`.
    await waitFor(() => {
      const terakhir = journal.mock.calls[journal.mock.calls.length - 1]?.[0]
      expect(terakhir?.from).toBe('2026-11-01')
      expect(terakhir?.days).toBeUndefined()
    })
  })

  it('menampilkan tanggal dan ekspor dalam satu tempat di atas', async () => {
    vi.spyOn(tradingApi, 'journal').mockResolvedValue(jurnalKosong())

    bungkus()

    // Keduanya bagian dari toolbar filter, bukan kartu terpisah di bawah.
    expect(await screen.findByLabelText('Dari tanggal')).toBeInTheDocument()
    expect(screen.getByLabelText('Sampai tanggal')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Export CSV/i })).toBeInTheDocument()
  })

  it('reset filter mengembalikan periode dan mengosongkan tanggal', async () => {
    const journal = vi.spyOn(tradingApi, 'journal').mockResolvedValue(jurnalKosong())
    const pengguna = userEvent.setup()

    bungkus()
    await waitFor(() => expect(journal).toHaveBeenCalled())

    await pengguna.type(screen.getByLabelText('Dari tanggal'), '2026-11-01')
    await waitFor(() => expect(screen.getByLabelText('Dari tanggal')).toHaveValue('2026-11-01'))

    await pengguna.click(screen.getByRole('button', { name: /Reset filter/i }))

    await waitFor(() => {
      const terakhir = journal.mock.calls[journal.mock.calls.length - 1]?.[0]
      expect(terakhir?.days).toBe(30)
      expect(terakhir?.from).toBeUndefined()
    })
    expect(screen.getByLabelText('Dari tanggal')).toHaveValue('')
  })

  it('mengirim nomor halaman saat pengguna membuka halaman berikutnya', async () => {
    const journal = vi.spyOn(tradingApi, 'journal').mockResolvedValue({
      ...jurnalKosong(),
      skipped_meta: { page: 1, per_page: 100, total: 250, total_pages: 3, has_more: true },
    })
    const pengguna = userEvent.setup()

    bungkus()
    await waitFor(() => expect(journal).toHaveBeenCalled())

    // Buka tab "Tidak Tereksekusi" tempat pagination berada.
    await pengguna.click(screen.getByRole('tab', { name: /Tidak Tereksekusi/i }))
    await pengguna.click(await screen.findByRole('button', { name: 'Halaman berikutnya' }))

    await waitFor(() => {
      const terakhir = journal.mock.calls[journal.mock.calls.length - 1]?.[0]
      expect(terakhir?.skippedPage).toBe(2)
    })
  })

  it('kembali ke halaman pertama saat filter diganti', async () => {
    const journal = vi.spyOn(tradingApi, 'journal').mockResolvedValue({
      ...jurnalKosong(),
      skipped_meta: { page: 1, per_page: 100, total: 250, total_pages: 3, has_more: true },
    })
    const pengguna = userEvent.setup()

    bungkus()
    await waitFor(() => expect(journal).toHaveBeenCalled())

    await pengguna.click(screen.getByRole('tab', { name: /Tidak Tereksekusi/i }))
    await pengguna.click(await screen.findByRole('button', { name: '3' }))
    await waitFor(() => {
      const terakhir = journal.mock.calls[journal.mock.calls.length - 1]?.[0]
      expect(terakhir?.skippedPage).toBe(3)
    })

    // Mengganti metode harus mengembalikan ke halaman 1.
    await pengguna.selectOptions(screen.getByLabelText('Metode'), 'ila-3.0')
    await waitFor(() => {
      const terakhir = journal.mock.calls[journal.mock.calls.length - 1]?.[0]
      expect(terakhir?.skippedPage).toBe(1)
      expect(terakhir?.strategy).toBe('ila-3.0')
    })
  })
})
