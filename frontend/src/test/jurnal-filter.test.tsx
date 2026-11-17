import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import PositionsPage from '@/pages/PositionsPage'
import { tradingApi } from '@/lib/api'
import { queryKeys } from '@/lib/queries'
import { request } from '@/lib/http'

vi.mock('@/lib/http', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/http')>()),
  request: vi.fn(),
}))
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

  it('filter simulasi dikirim server-side, reset halaman, dan mempertahankan pilihan lain', async () => {
    const journal = vi.spyOn(tradingApi, 'journal').mockImplementation(async (params) => ({
      ...jurnalKosong(),
      filters: { eval_status: params.evalStatus },
      skipped_meta: {
        page: params.skippedPage ?? 1,
        per_page: params.skippedLimit ?? 20,
        total: params.evalStatus ? 7 : 250,
        total_pages: params.evalStatus ? 1 : 13,
        has_more: !params.evalStatus,
      },
    }))
    const pengguna = userEvent.setup()
    bungkus()
    await pengguna.selectOptions(screen.getByLabelText('Metode'), 'ila-3.0')
    await pengguna.type(screen.getByLabelText('Dari tanggal'), '2026-11-01')
    await pengguna.click(screen.getByRole('tab', { name: /Tidak Tereksekusi/i }))
    await pengguna.selectOptions(screen.getByLabelText(/per halaman/i), '50')
    await pengguna.click(await screen.findByRole('button', { name: 'Halaman berikutnya' }))
    await waitFor(() => expect(journal.mock.lastCall?.[0].skippedPage).toBe(2))
    await pengguna.selectOptions(screen.getByLabelText('Status simulasi'), 'PENDING')
    await waitFor(() =>
      expect(journal.mock.lastCall?.[0]).toMatchObject({
        evalStatus: 'PENDING',
        skippedPage: 1,
        skippedLimit: 50,
        strategy: 'ila-3.0',
        from: '2026-11-01',
      }),
    )
    expect(screen.getByRole('tab', { name: /Tidak Tereksekusi/i })).toHaveTextContent('7')
    expect(
      screen.getByText('Simulasi hasil sinyal yang tidak menjadi transaksi akun. Bukan PnL akun.'),
    ).toBeInTheDocument()
    await pengguna.click(screen.getByRole('tab', { name: /Posisi Aktif/i }))
    await pengguna.click(screen.getByRole('tab', { name: /Tidak Tereksekusi/i }))
    expect(screen.getByLabelText('Status simulasi')).toHaveValue('PENDING')
    for (const status of ['RUNNING', 'TP', 'SL', 'NO_ENTRY', 'UNKNOWN', '']) {
      await pengguna.selectOptions(screen.getByLabelText('Status simulasi'), status)
      await waitFor(() => expect(journal.mock.lastCall?.[0].evalStatus).toBe(status))
    }
    await pengguna.selectOptions(screen.getByLabelText('Status simulasi'), 'TP')
    await pengguna.click(screen.getByRole('button', { name: /Reset filter/i }))
    expect(screen.getByLabelText('Status simulasi')).toHaveValue('')
  })

  it('query key membedakan status dan API mengirim eval_status, blank tanpa filter', async () => {
    expect(queryKeys.journal({ evalStatus: 'TP' })).not.toEqual(
      queryKeys.journal({ evalStatus: 'SL' }),
    )
    await tradingApi.journal({ evalStatus: 'PENDING', skippedPage: 2 })
    expect(request).toHaveBeenLastCalledWith(
      '/journal',
      expect.objectContaining({
        query: expect.objectContaining({ eval_status: 'PENDING', skipped_page: 2 }),
      }),
    )
    await tradingApi.journal({ evalStatus: '' })
    expect(request).toHaveBeenLastCalledWith(
      '/journal',
      expect.objectContaining({
        query: expect.objectContaining({ eval_status: undefined }),
      }),
    )
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
