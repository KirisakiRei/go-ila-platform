import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ManualJournalPage from '@/pages/ManualJournalPage'
import { journalApi } from './api'
import { emptyTrade } from './model'
import { TradeForm } from './TradeForm'
import { CSVImport } from './CSVImport'
import { CalendarView } from './JournalViews'
import { AccountManager } from './AccountManager'
import { AccountBalance } from './AccountBalance'
import { JournalKpiStrip } from './JournalAnalytics'
import type { JournalAccount, JournalStats, JournalTrade } from './types'

const account: JournalAccount = {
  id: 'a1',
  name: 'Akun mandiri',
  broker: 'Manual',
  currency: 'USD',
  initial_capital: '1000',
  initial_date: '2026-11-01',
  current_realized_balance: '1012.3456',
  balance_complete: true,
  unknown_closed_pnl: 0,
  notes: '',
  archived: false,
  checklist: ['Risiko sesuai'],
}

const trade: JournalTrade = {
  ...emptyTrade('a1'),
  id: 't1',
  created_at: '2026-11-25T18:00:00Z',
  asset: 'XAUUSD',
  status: 'closed',
  closed_at: '2026-11-25T18:00:00Z',
  net_pnl: '12.3456',
}

const stats: JournalStats = {
  currency: 'USD',
  trades: 1,
  wins: 1,
  losses: 0,
  breakeven: 0,
  win_rate: '100',
  profit_factor: null,
  average_win: '12.3456',
  average_loss: '0',
  net_pnl: '12.3456',
  max_drawdown: '0',
  curve: [],
  calendar: { '2026-11-26': { trades: 1, wins: 1, net_pnl: '12.3456' } },
  breakdowns: { setup: { Breakout: { trades: 1, wins: 1, net_pnl: '12.3456' } } },
}

function wrap(component: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{component}</QueryClientProvider>)
}

beforeEach(() => {
  vi.restoreAllMocks()
  vi.spyOn(journalApi, 'accounts').mockResolvedValue({ data: [account] })
  vi.spyOn(journalApi, 'entries').mockResolvedValue({ data: [trade] })
  vi.spyOn(journalApi, 'stats').mockResolvedValue(stats)
  vi.spyOn(journalApi, 'insights').mockResolvedValue([
    { code: 'low_sample', message: 'Sampel masih kecil.', count: 1 },
  ])
  vi.spyOn(journalApi, 'screenshots').mockResolvedValue([])
  vi.spyOn(journalApi, 'review').mockResolvedValue({
    account_id: 'a1',
    date: '2026-11-26',
    revision: 'r1',
    status: 'ready',
    text: 'Tetap jaga risiko.',
    source: 'rules',
    updated_at: '2026-11-26T00:00:00Z',
  })
  vi.spyOn(journalApi, 'reviewSettings').mockResolvedValue({
    account_id: 'a1',
    ai_enabled: false,
    effective_ai_enabled: false,
  })
})

describe('standalone journal UI', () => {
  it('renders Jurnal title, compact balance tooltip, and single settings modal', async () => {
    wrap(<ManualJournalPage />)
    expect(await screen.findByLabelText('Balance Akun mandiri')).toHaveTextContent(
      '1.012,3456',
    )
    expect(screen.getByRole('heading', { name: 'Jurnal' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Info balance Akun mandiri' }))
    expect(screen.getByRole('tooltip')).toHaveTextContent(/Modal awal \+ deposit/)

    fireEvent.click(screen.getByRole('button', { name: 'Pengaturan jurnal' }))
    expect(screen.getByText('Pengaturan jurnal')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'AI' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'CSV' })).toBeInTheDocument()
  })

  it('blocks account form controls during a delayed save, then allows editing the next account', async () => {
    const other = { ...account, id: 'a2', name: 'Akun kedua' }
    let finish!: (value: JournalAccount) => void
    const save = vi.spyOn(journalApi, 'saveAccount').mockReturnValue(
      new Promise((resolve) => {
        finish = resolve
      }),
    )
    wrap(<AccountManager accounts={[account, other]} />)
    fireEvent.click(screen.getAllByRole('button', { name: /^Edit$/ })[0]!)
    fireEvent.change(screen.getByLabelText('Nama jurnal'), { target: { value: 'Draft pertama' } })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan perubahan' }))
    await waitFor(() => expect(save).toHaveBeenCalledOnce())

    expect(screen.getByRole('button', { name: 'Batal' })).toBeDisabled()
    expect(screen.getByLabelText('Nama jurnal')).toBeDisabled()

    await act(async () => {
      finish(account)
    })
    await waitFor(() => expect(screen.getAllByRole('button', { name: /^Edit$/ })[1]).toBeEnabled())
    fireEvent.click(screen.getAllByRole('button', { name: /^Edit$/ })[1]!)
    expect(screen.getByLabelText('Nama jurnal')).toHaveValue(other.name)
    expect(save).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Draft pertama' }),
      account.id,
    )
  })

  it.each(['save', 'delete'] as const)(
    'blocks capital controls during delayed %s without browser confirm popups',
    async (operation) => {
      const movement = {
        id: 'c1',
        account_id: account.id,
        kind: 'deposit' as const,
        amount: '100',
        date: '2026-11-01',
        notes: 'Modal pertama',
      }
      const otherMovement = { ...movement, id: 'c2', amount: '200', notes: 'Modal kedua' }
      vi.spyOn(journalApi, 'capital').mockResolvedValue({ data: [movement, otherMovement] })
      let finish!: () => void
      const pending = new Promise<void>((resolve) => {
        finish = resolve
      })
      const save = vi.spyOn(journalApi, 'saveCapital').mockImplementation(async () => {
        await pending
        return movement
      })
      const remove = vi.spyOn(journalApi, 'deleteCapital').mockReturnValue(pending)

      wrap(<AccountManager accounts={[account, { ...account, id: 'a2', name: 'Akun kedua' }]} />)
      fireEvent.click(screen.getByRole('button', { name: 'Deposit & Withdrawal' }))
      const row = (await screen.findByText('Modal pertama')).closest('li')!
      fireEvent.click(within(row).getByRole('button', { name: 'Edit' }))
      fireEvent.change(screen.getByLabelText('Jumlah (USD)'), { target: { value: '150' } })

      if (operation === 'save') {
        fireEvent.click(screen.getByRole('button', { name: 'Simpan perubahan' }))
      } else {
        fireEvent.click(within(row).getByRole('button', { name: 'Hapus' }))
        fireEvent.click(within(row).getByRole('button', { name: 'Yakin?' }))
      }

      await waitFor(() => expect(operation === 'save' ? save : remove).toHaveBeenCalledOnce())
      expect(screen.getByLabelText('Jurnal')).toBeDisabled()
      expect(screen.getByLabelText('Jumlah (USD)')).toBeDisabled()

      await act(async () => {
        finish()
      })
      await waitFor(() => expect(screen.getByLabelText('Jurnal')).toBeEnabled())
      fireEvent.click(
        within(screen.getByText('Modal kedua').closest('li')!).getByRole('button', {
          name: 'Edit',
        }),
      )
      expect(screen.getByLabelText('Jumlah (USD)')).toHaveValue('200')
    },
  )

  it('locks account controls while archiving via inline confirmation', async () => {
    let finish!: () => void
    const archive = vi.spyOn(journalApi, 'archiveAccount').mockReturnValue(
      new Promise((resolve) => {
        finish = resolve
      }),
    )
    wrap(<AccountManager accounts={[account]} />)
    fireEvent.click(screen.getByRole('button', { name: 'Arsipkan' }))
    fireEvent.click(screen.getByRole('button', { name: 'Yakin arsip?' }))
    await waitFor(() => expect(archive).toHaveBeenCalledOnce())
    expect(screen.getByRole('button', { name: 'Edit' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Tambah jurnal' })).toBeDisabled()
    await act(async () => {
      finish()
    })
    await waitFor(() => expect(screen.getByRole('button', { name: 'Edit' })).toBeEnabled())
  })

  it('deletes a journal permanently via inline confirmation', async () => {
    const del = vi.spyOn(journalApi, 'deleteAccount').mockResolvedValue(undefined)
    wrap(<AccountManager accounts={[account]} />)
    fireEvent.click(screen.getByRole('button', { name: 'Hapus' }))
    fireEvent.click(screen.getByRole('button', { name: 'Hapus permanen?' }))
    await waitFor(() => expect(del).toHaveBeenCalledOnce())
    expect(del.mock.calls[0]?.[0]).toBe(account.id)
  })

  it('never renders null or incomplete balance as zero, but renders a known zero', () => {
    const view = wrap(
      <AccountBalance
        account={{
          ...account,
          current_realized_balance: null,
          balance_complete: false,
          unknown_closed_pnl: 2,
        }}
      />,
    )
    expect(screen.getByText('Belum lengkap')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Info balance Akun mandiri' }))
    expect(screen.getByRole('tooltip')).toHaveTextContent(/2 trade closed belum punya P&L/)
    view.unmount()

    wrap(
      <AccountBalance
        account={{ ...account, current_realized_balance: '0.0000', balance_complete: true }}
      />,
    )
    expect(screen.queryByText('Belum lengkap')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Balance Akun mandiri')).toHaveTextContent('0,00')
  })

  it('provides hover and focus tooltips for all 4 KPI items without inline disclosure arrows', () => {
    wrap(<JournalKpiStrip stats={stats} />)
    const triggers = screen.getAllByLabelText(/^Definisi /)
    expect(triggers).toHaveLength(4)
    expect(document.querySelectorAll('details')).toHaveLength(0)

    const winRate = screen.getByRole('button', { name: 'Definisi Win rate' })
    fireEvent.focus(winRate)
    expect(screen.getByRole('tooltip')).toHaveTextContent(/Breakeven tidak dihitung/i)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('shows independent journal onboarding and does not request stats without journal', async () => {
    vi.mocked(journalApi.accounts).mockResolvedValue({ data: [] })
    wrap(<ManualJournalPage />)
    expect(await screen.findByRole('button', { name: 'Tambah jurnal' })).toBeInTheDocument()
    expect(journalApi.stats).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Tambah jurnal' }))
    expect(screen.getByLabelText('Nama jurnal')).toBeInTheDocument()
  })

  it('opens detail directly on 1-trade calendar day and opens modal list on multi-trade day', () => {
    const select = vi.fn()
    const single = wrap(
      <CalendarView
        trades={[trade]}
        accounts={[account]}
        month="2026-11"
        onMonth={() => {}}
        onSelect={select}
        calendar={stats.calendar}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /2026-11-26 WIB, 1 trade/ }))
    expect(select).toHaveBeenCalledWith(trade)
    single.unmount()

    const secondTrade = { ...trade, id: 't2', asset: 'BTCUSD' }
    select.mockClear()
    wrap(
      <CalendarView
        trades={[trade, secondTrade]}
        accounts={[account]}
        month="2026-11"
        onMonth={() => {}}
        onSelect={select}
        calendar={stats.calendar}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /2026-11-26 WIB, 2 trade/ }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Rekap 2026-11-26 · WIB')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /BTCUSD/ }))
    expect(select).toHaveBeenCalledWith(secondTrade)
  })

  it('switches workspace tabs and shares filters across table, analytics, and stats', async () => {
    wrap(<ManualJournalPage />)
    expect(await screen.findByText('Tetap jaga risiko.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /^Tabel$/ }))
    fireEvent.click(screen.getByRole('button', { name: /^XAUUSD$/ }))
    expect(await screen.findByText('Detail trade')).toBeInTheDocument()
    expect(screen.getByText('Screenshot chart')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Tutup dialog' }))

    fireEvent.click(screen.getByRole('button', { name: /^Filter/ }))
    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'closed' } })
    await waitFor(() =>
      expect(journalApi.stats).toHaveBeenLastCalledWith(
        expect.objectContaining({ account_id: 'a1', status: 'closed' }),
        expect.any(AbortSignal),
      ),
    )

    fireEvent.click(screen.getByRole('button', { name: /^Analitik$/ }))
    expect(await screen.findByText('Sampel masih kecil.')).toBeInTheDocument()
  })

  it('uses Buy/Sell toggle, Status segmented control, live RR preview, and chip inputs preserving commas', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    wrap(
      <TradeForm
        initial={{
          ...emptyTrade('a1'),
          status: 'closed',
          opened_at: '2026-11-25T10:00:00Z',
          closed_at: '2026-11-25T11:30:00Z',
          size_unit: 'lot',
        }}
        accounts={[account]}
        pending={false}
        onSave={save}
        onCancel={() => {}}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Sell / Short' }))
    fireEvent.change(screen.getByLabelText('Pair / Aset'), { target: { value: 'xauusd' } })
    fireEvent.change(screen.getByLabelText('Entry'), { target: { value: '3500' } })
    fireEvent.change(screen.getByLabelText('Stop loss'), { target: { value: '3510' } })
    fireEvent.change(screen.getByLabelText('Take profit'), { target: { value: '3480' } })
    fireEvent.change(screen.getByLabelText('Size'), { target: { value: '0.1' } })
    fireEvent.change(screen.getByLabelText('Net P&L aktual (USD)'), { target: { value: '150' } })

    expect(screen.getByLabelText('Preview kalkulasi otomatis')).toHaveTextContent('1:2.00')
    expect(screen.getByLabelText('Preview kalkulasi otomatis')).toHaveTextContent('1j 30m')

    fireEvent.click(screen.getByRole('button', { name: /Evaluasi & Bukti/ }))
    fireEvent.click(screen.getByRole('button', { name: '+ Disiplin' }))
    const tagInput = screen.getByLabelText('Tag')
    fireEvent.change(tagInput, { target: { value: 'London, NY' } })
    fireEvent.keyDown(tagInput, { key: 'Enter' })

    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({
          asset: 'XAUUSD',
          side: 'short',
          emotions: ['Disiplin'],
          tags: ['London, NY'],
          net_pnl: '150',
        }),
        [],
      ),
    )
  })

  it('disables account change during edit and preserves comma labels in checklist', async () => {
    const original = { ...account, checklist: ['London, NY', 'Risk, then entry'] }
    const save = vi.spyOn(journalApi, 'saveAccount').mockResolvedValue(original)
    wrap(<AccountManager accounts={[original]} />)
    fireEvent.click(screen.getByRole('button', { name: /^Edit$/ }))
    fireEvent.change(screen.getByLabelText('Nama jurnal'), { target: { value: 'Renamed' } })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan perubahan' }))
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({ name: 'Renamed', checklist: original.checklist }),
        account.id,
      ),
    )
  })

  it('previews CSV with account names and confirms with single-use token', async () => {
    const preview = vi.spyOn(journalApi, 'preview').mockResolvedValue({
      count: 1,
      token: 'tok-1',
      expires_at: new Date(Date.now() + 60000).toISOString(),
      rows: [{ ...emptyTrade('a1'), asset: 'XAUUSD', status: 'closed', net_pnl: '25' }],
    })
    const commit = vi.spyOn(journalApi, 'import').mockResolvedValue({ count: 1 })
    const done = vi.fn()
    wrap(<CSVImport accounts={[account]} onDone={done} />)

    const file = new File(['account_id,asset\na1,XAUUSD\n'], 'journal.csv', { type: 'text/csv' })
    Object.defineProperty(file, 'text', { value: async () => 'account_id,asset\na1,XAUUSD\n' })
    fireEvent.change(screen.getByLabelText('Berkas CSV'), { target: { files: [file] } })

    await waitFor(() => expect(preview).toHaveBeenCalledOnce())
    expect(await screen.findByText('Akun mandiri')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Konfirmasi impor 1 trade' }))
    await waitFor(() => expect(commit).toHaveBeenCalledOnce())
    expect(commit.mock.calls[0]?.[0]).toEqual({
      csv: 'account_id,asset\na1,XAUUSD\n',
      token: 'tok-1',
    })
    expect(done).toHaveBeenCalledOnce()
  })

  it('keeps InfoTooltip open on pointerDown+focus+click sequence without closing immediately', async () => {
    wrap(<AccountBalance account={account} />)
    const trigger = screen.getByRole('button', { name: 'Info balance Akun mandiri' })
    fireEvent.pointerDown(trigger)
    fireEvent.focus(trigger)
    fireEvent.click(trigger)
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
  })

  it('handles draft empty side, exit_price reset, leading + pnl, unsubmitted chips, and cross-tab validation switch', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    wrap(
      <TradeForm
        initial={{
          ...emptyTrade('a1'),
          status: 'draft',
          side: '',
          exit_price: '3550',
          size_unit: 'lot',
        }}
        accounts={[account]}
        pending={false}
        onSave={save}
        onCancel={() => {}}
      />,
    )

    expect(screen.getByRole('button', { name: 'Buy / Long' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Sell / Short' })).toHaveAttribute('aria-pressed', 'false')

    // Switch to Open: defaults empty side to long and clears exit_price
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))
    expect(screen.getByRole('button', { name: 'Buy / Long' })).toHaveAttribute('aria-pressed', 'true')

    // Switch to Closed and verify exit_price was cleared + leading '+' stripped
    fireEvent.click(screen.getByRole('button', { name: 'Closed' }))
    expect(screen.getByLabelText('Exit aktual')).toHaveValue('')
    fireEvent.change(screen.getByLabelText('Pair / Aset'), { target: { value: 'EURUSD' } })
    fireEvent.change(screen.getByLabelText('Entry'), { target: { value: '1.1000' } })
    fireEvent.change(screen.getByLabelText('Size'), { target: { value: '0.2' } })
    fireEvent.change(screen.getByLabelText('Net P&L aktual (USD)'), { target: { value: '+85.25' } })

    // Switch to Evaluasi & Bukti, type chip draft without pressing Enter, and type invalid TradingView URL
    fireEvent.click(screen.getByRole('button', { name: /Evaluasi & Bukti/ }))
    fireEvent.change(screen.getByLabelText('Tag'), { target: { value: 'NY-open' } })
    fireEvent.change(screen.getByPlaceholderText('https://www.tradingview.com/x/...'), {
      target: { value: 'http://evil.example.com/chart' },
    })

    // Switch back to Inti Trade -> submit should validate and auto-switch back to review tab
    fireEvent.click(screen.getByRole('button', { name: 'Inti Trade' }))
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))
    expect(await screen.findByText(/Link TradingView: Gunakan link HTTPS/i)).toBeInTheDocument()
    expect(screen.getByLabelText('Tag')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('https://www.tradingview.com/x/...')).toHaveAttribute(
      'aria-invalid',
      'true',
    )

    // Remove invalid URL, drag-and-drop a chart image, and submit again
    fireEvent.click(screen.getByRole('button', { name: /Hapus http:\/\/evil\.example\.com\/chart/ }))
    fireEvent.change(screen.getByLabelText('Psikologi / Emosi'), { target: { value: 'Fokus' } })
    const dropzone = screen.getByLabelText('Pilih gambar').closest('label')!
    const droppedImg = new File(['png'], 'drop.png', { type: 'image/png' })
    fireEvent.drop(dropzone, { dataTransfer: { files: [droppedImg] } })
    expect(screen.getByAltText('drop.png')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))

    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({
          asset: 'EURUSD',
          side: 'long',
          exit_price: null,
          net_pnl: '85.25',
          tags: ['NY-open'],
          emotions: ['Fokus'],
        }),
        [droppedImg],
      ),
    )
  })

  it('shows error alert if deleting a trade fails and locks TradeForm during queued screenshot upload', async () => {
    vi.spyOn(journalApi, 'deleteEntry').mockRejectedValueOnce(new Error('Gagal hapus server'))
    wrap(<ManualJournalPage />)
    expect(await screen.findByText('Tetap jaga risiko.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /^Tabel$/ }))
    fireEvent.click(screen.getByRole('button', { name: /^XAUUSD$/ }))
    expect(await screen.findByText('Detail trade')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Hapus trade' }))
    fireEvent.click(screen.getByRole('button', { name: 'Yakin hapus?' }))
    expect(await screen.findByText('Gagal hapus server')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Tutup dialog' }))

    let finishUpload!: () => void
    vi.spyOn(journalApi, 'saveEntry').mockResolvedValueOnce(trade)
    vi.spyOn(journalApi, 'uploadScreenshot').mockReturnValueOnce(
      new Promise((resolve) => {
        finishUpload = () =>
          resolve({
            id: 's1',
            entry_id: 't1',
            file_name: 'chart.png',
            mime_type: 'image/png',
            size_bytes: 10,
          })
      }),
    )

    fireEvent.click(screen.getByRole('button', { name: 'Catat trade' }))
    fireEvent.change(screen.getByLabelText('Pair / Aset'), { target: { value: 'XAUUSD' } })
    fireEvent.change(screen.getByLabelText('Entry'), { target: { value: '3500' } })
    fireEvent.change(screen.getByLabelText('Size'), { target: { value: '0.1' } })
    fireEvent.change(screen.getByLabelText('Net P&L aktual (USD)'), { target: { value: '50' } })
    fireEvent.click(screen.getByRole('button', { name: /Evaluasi & Bukti/ }))
    const img = new File(['png'], 'chart.png', { type: 'image/png' })
    fireEvent.change(screen.getByLabelText('Pilih gambar'), { target: { files: [img] } })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))

    await waitFor(() => expect(journalApi.uploadScreenshot).toHaveBeenCalledOnce())
    expect(screen.getByRole('button', { name: 'Memproses…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Batal' })).toBeDisabled()

    await act(async () => {
      finishUpload()
    })
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('resets capital edit state when deleting the currently edited item and flushes checklist draft on submit', async () => {
    const movement = {
      id: 'c1',
      account_id: account.id,
      kind: 'deposit' as const,
      amount: '100',
      date: '2026-11-01',
      notes: 'Modal pertama',
    }
    vi.spyOn(journalApi, 'capital').mockResolvedValue({ data: [movement] })
    vi.spyOn(journalApi, 'deleteCapital').mockResolvedValue(undefined)
    const saveAccount = vi.spyOn(journalApi, 'saveAccount').mockResolvedValue(account)

    wrap(<AccountManager accounts={[account]} />)
    // Flush checklist draft without pressing Enter
    fireEvent.click(screen.getByRole('button', { name: /^Edit$/ }))
    fireEvent.change(screen.getByLabelText('Template pre-trade checklist'), {
      target: { value: 'Tunggu close H1' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan perubahan' }))
    await waitFor(() =>
      expect(saveAccount).toHaveBeenCalledWith(
        expect.objectContaining({ checklist: ['Risiko sesuai', 'Tunggu close H1'] }),
        account.id,
      ),
    )

    // Delete currently edited capital movement -> exits edit mode
    fireEvent.click(screen.getByRole('button', { name: 'Deposit & Withdrawal' }))
    const row = (await screen.findByText('Modal pertama')).closest('li')!
    fireEvent.click(within(row).getByRole('button', { name: 'Edit' }))
    expect(screen.getByText('Edit pergerakan modal')).toBeInTheDocument()
    fireEvent.click(within(row).getByRole('button', { name: 'Hapus' }))
    fireEvent.click(within(row).getByRole('button', { name: 'Yakin?' }))
    await waitFor(() => expect(screen.getByText('Catat deposit / withdrawal')).toBeInTheDocument())
  })

  it('highlights invalid capital_snapshot in red with human label and accepts comma decimal after fix', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    wrap(
      <TradeForm
        initial={{ ...emptyTrade('a1'), status: 'draft', size_unit: 'lot' }}
        accounts={[account]}
        pending={false}
        onSave={save}
        onCancel={() => {}}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /Evaluasi & Bukti/ }))
    const capInput = screen.getByLabelText(/Modal saat entry/)
    fireEvent.change(capInput, { target: { value: 'abc' } })
    fireEvent.click(screen.getByRole('button', { name: 'Inti Trade' }))
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))

    expect(await screen.findByText(/Modal saat entry:/)).toBeInTheDocument()
    expect(screen.queryByText(/capital_snapshot/)).not.toBeInTheDocument()
    const invalidCap = screen.getByLabelText(/Modal saat entry/)
    expect(invalidCap).toHaveAttribute('aria-invalid', 'true')
    expect(invalidCap.className).toContain('border-destructive')

    fireEvent.change(invalidCap, { target: { value: '1000,50' } })
    expect(invalidCap).not.toHaveAttribute('aria-invalid', 'true')
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))

    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({ capital_snapshot: '1000.50' }),
        [],
      ),
    )
  })

  it('blocks inverted Buy TP/SL with red outline and clears error when toggling to Sell', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    wrap(
      <TradeForm
        initial={{ ...emptyTrade('a1'), status: 'draft', side: 'long', size_unit: 'lot' }}
        accounts={[account]}
        pending={false}
        onSave={save}
        onCancel={() => {}}
      />,
    )

    fireEvent.change(screen.getByLabelText('Entry'), { target: { value: '3500' } })
    fireEvent.change(screen.getByLabelText('Stop loss'), { target: { value: '3510' } })
    fireEvent.change(screen.getByLabelText('Take profit'), { target: { value: '3480' } })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))

    expect(await screen.findByText(/Untuk Buy, Stop loss harus lebih rendah dari Entry/i)).toBeInTheDocument()
    expect(screen.getByLabelText('Take profit')).toHaveAttribute('aria-invalid', 'true')
    expect(save).not.toHaveBeenCalled()

    // Switching to Sell makes SL=3510 > Entry=3500 > TP=3480 valid and clears red outlines
    fireEvent.click(screen.getByRole('button', { name: 'Sell / Short' }))
    expect(screen.getByLabelText('Take profit')).not.toHaveAttribute('aria-invalid', 'true')
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({ side: 'short', entry_price: '3500', stop_loss: '3510', take_profit: '3480' }),
        [],
      ),
    )
  })
})
