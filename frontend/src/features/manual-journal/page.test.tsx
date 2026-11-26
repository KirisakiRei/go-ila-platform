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
import { JournalAnalytics } from './JournalAnalytics'
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
  it('shows computed realized balance for selected account and account manager with broker-equity caveat', async () => {
    const view = wrap(<ManualJournalPage />)
    expect(await screen.findByLabelText('Saldo terealisasi Akun mandiri')).toHaveTextContent(
      '1.012,3456',
    )
    expect(screen.getByText(/bukan equity broker dan tidak termasuk floating/)).toBeInTheDocument()
    view.unmount()
    wrap(<AccountManager accounts={[account]} />)
    expect(screen.getByLabelText('Saldo terealisasi Akun mandiri')).toHaveTextContent('1.012,3456')
    expect(screen.queryByText(/otomatis belum tersedia/)).not.toBeInTheDocument()
  })
  it('blocks account switches and resets during a delayed save, then allows editing the next account', async () => {
    const other = { ...account, id: 'a2', name: 'Akun kedua' }
    let finish!: (value: JournalAccount) => void
    const save = vi.spyOn(journalApi, 'saveAccount').mockReturnValue(
      new Promise((resolve) => {
        finish = resolve
      }),
    )
    const archive = vi.spyOn(journalApi, 'archiveAccount')
    wrap(<AccountManager accounts={[account, other]} />)
    fireEvent.click(screen.getAllByRole('button', { name: /^Edit$/ })[0]!)
    fireEvent.change(screen.getByLabelText('Nama akun'), { target: { value: 'Draft pertama' } })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan akun' }))
    await waitFor(() => expect(save).toHaveBeenCalledOnce())
    const editOther = screen.getAllByRole('button', { name: /^Edit$/ })[1]!
    expect(editOther).toBeDisabled()
    fireEvent.click(editOther)
    const cancel = screen.getByRole('button', { name: 'Batal edit' })
    expect(cancel).toBeDisabled()
    fireEvent.click(cancel)
    for (const button of screen.getAllByRole('button', { name: /^(Arsip|Modal)$/ })) {
      expect(button).toBeDisabled()
      fireEvent.click(button)
    }
    expect(screen.getByLabelText('Akun modal')).toBeDisabled()
    expect(screen.getByLabelText('Nama akun')).toHaveValue('Draft pertama')
    expect(archive).not.toHaveBeenCalled()
    await act(async () => {
      finish(account)
    })
    await waitFor(() => expect(editOther).toBeEnabled())
    fireEvent.click(editOther)
    expect(screen.getByLabelText('Nama akun')).toHaveValue(other.name)
    fireEvent.change(screen.getByLabelText('Nama akun'), { target: { value: 'Draft kedua' } })
    expect(screen.getByLabelText('Nama akun')).toHaveValue('Draft kedua')
    expect(save).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Draft pertama' }),
      account.id,
    )
  })
  it.each(['save', 'delete'] as const)(
    'blocks capital target switches and resets during delayed %s',
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
      vi.spyOn(window, 'confirm').mockReturnValue(true)
      wrap(<AccountManager accounts={[account, { ...account, id: 'a2', name: 'Akun kedua' }]} />)
      fireEvent.change(screen.getByLabelText('Akun modal'), { target: { value: account.id } })
      const row = (await screen.findByText('Modal pertama')).closest('li')!
      fireEvent.click(within(row).getByRole('button', { name: 'Edit' }))
      fireEvent.change(screen.getByLabelText('Jumlah (USD)'), { target: { value: '150' } })
      fireEvent.click(
        operation === 'save'
          ? screen.getByRole('button', { name: 'Simpan perubahan modal' })
          : within(row).getByRole('button', { name: 'Hapus' }),
      )
      await waitFor(() => expect(operation === 'save' ? save : remove).toHaveBeenCalledOnce())
      for (const button of screen.getAllByRole('button', {
        name: /^(Edit|Hapus|Arsip|Modal|Batal edit)$/,
      })) {
        expect(button).toBeDisabled()
        fireEvent.click(button)
      }
      expect(screen.getByLabelText('Akun modal')).toBeDisabled()
      expect(screen.getByLabelText('Nama akun')).toBeDisabled()
      expect(screen.getByLabelText('Jumlah (USD)')).toHaveValue('150')
      await act(async () => {
        finish()
      })
      await waitFor(() => expect(screen.getByLabelText('Akun modal')).toBeEnabled())
      fireEvent.click(
        within(screen.getByText('Modal kedua').closest('li')!).getByRole('button', {
          name: 'Edit',
        }),
      )
      expect(screen.getByLabelText('Jumlah (USD)')).toHaveValue('200')
      expect(save).toHaveBeenCalledTimes(operation === 'save' ? 1 : 0)
      expect(remove).toHaveBeenCalledTimes(operation === 'delete' ? 1 : 0)
    },
  )
  it('locks account and capital controls while archiving', async () => {
    let finish!: () => void
    const archive = vi.spyOn(journalApi, 'archiveAccount').mockReturnValue(
      new Promise((resolve) => {
        finish = resolve
      }),
    )
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    wrap(<AccountManager accounts={[account]} />)
    fireEvent.click(screen.getByRole('button', { name: 'Arsip' }))
    await waitFor(() => expect(archive).toHaveBeenCalledOnce())
    expect(screen.getByRole('button', { name: 'Edit' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Modal' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Arsip' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Tambah akun' })).toBeDisabled()
    expect(screen.getByLabelText('Nama akun')).toBeDisabled()
    expect(screen.getByLabelText('Akun modal')).toBeDisabled()
    await act(async () => {
      finish()
    })
    await waitFor(() => expect(screen.getByRole('button', { name: 'Edit' })).toBeEnabled())
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
    expect(screen.getByText('Belum dapat ditentukan')).toBeInTheDocument()
    expect(screen.getByText(/2 trade closed belum memiliki net P&L/)).toBeInTheDocument()
    view.unmount()
    wrap(
      <AccountBalance
        account={{ ...account, current_realized_balance: '0.0000', balance_complete: true }}
      />,
    )
    expect(screen.queryByText('Belum dapat ditentukan')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Saldo terealisasi Akun mandiri')).toHaveTextContent('0,00')
  })
  it('provides keyboard-focusable expandable definitions for every KPI', () => {
    wrap(<JournalAnalytics stats={stats} insights={[]} />)
    const summaries = screen.getAllByLabelText(/^Definisi /)
    expect(summaries).toHaveLength(8)
    for (const summary of summaries) expect(summary.tagName).toBe('SUMMARY')
    const winRate = screen.getByLabelText('Definisi Win rate')
    winRate.focus()
    expect(winRate).toHaveFocus()
    fireEvent.click(winRate)
    expect(winRate.parentElement).toHaveAttribute('open')
    expect(winRate.parentElement).toHaveTextContent('Breakeven tidak masuk penyebut')
    const drawdown = screen.getByLabelText('Definisi Max drawdown')
    fireEvent.click(drawdown)
    expect(drawdown.parentElement).toHaveTextContent('Bukan drawdown equity broker')
  })
  it('shows independent account onboarding and does not request stats without account', async () => {
    vi.mocked(journalApi.accounts).mockResolvedValue({ data: [] })
    wrap(<ManualJournalPage />)
    expect(await screen.findByRole('button', { name: 'Buat akun jurnal' })).toBeInTheDocument()
    expect(journalApi.stats).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Buat akun jurnal' }))
    expect(screen.getByLabelText('Nama akun')).toBeInTheDocument()
  })
  it('opens daily recap and trade cards on calendar selection', () => {
    const select = vi.fn()
    wrap(
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
    expect(screen.getByText('Rekap 2026-11-26 · WIB')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /XAUUSD/ }))
    expect(select).toHaveBeenCalledWith(trade)
  })
  it('shares filters across reports and table and labels rules honestly', async () => {
    wrap(<ManualJournalPage />)
    expect(await screen.findByText('Sampel masih kecil.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /^Tabel$/ }))
    fireEvent.click(screen.getByRole('button', { name: /^XAUUSD$/ }))
    expect(await screen.findByText('Detail trade')).toBeInTheDocument()
    expect(screen.getByText('Screenshot sebelum / sesudah')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Tutup dialog' }))
    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'closed' } })
    await waitFor(() =>
      expect(journalApi.stats).toHaveBeenLastCalledWith(
        expect.objectContaining({ account_id: 'a1', status: 'closed' }),
        expect.any(AbortSignal),
      ),
    )
    expect(await screen.findByText(/Evaluasi ini tidak menggunakan AI/)).toBeInTheDocument()
  })
  it('keeps zero and exact decimal strings on save and disables changing account during edit', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const input = {
      ...emptyTrade('a1'),
      status: 'closed' as const,
      asset: 'XAUUSD',
      opened_at: '2026-11-25T10:00:00Z',
      closed_at: '2026-11-25T11:00:00Z',
      entry_price: '2600.12345678',
      size: '0.01',
      size_unit: 'lot',
      net_pnl: '0',
    }
    wrap(
      <TradeForm
        initial={input}
        editing
        accounts={[account]}
        pending={false}
        onSave={save}
        onCancel={() => {}}
      />,
    )
    expect(screen.getByLabelText('Akun jurnal')).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({ entry_price: '2600.12345678', net_pnl: '0' }),
      ),
    )
  })
  it('does not import until server preview is explicitly confirmed', async () => {
    const preview = vi.spyOn(journalApi, 'preview').mockResolvedValue({
      rows: [emptyTrade('a1')],
      count: 1,
      token: 'preview-token',
      expires_at: '2099-01-01T00:00:00Z',
    })
    const commit = vi.spyOn(journalApi, 'import').mockResolvedValue({ count: 1 })
    const done = vi.fn()
    wrap(<CSVImport onDone={done} />)
    const file = new File(['csv payload'], 'journal.csv', { type: 'text/csv' })
    Object.defineProperty(file, 'text', { value: async () => 'csv payload' })
    fireEvent.change(screen.getByLabelText('Berkas CSV'), { target: { files: [file] } })
    expect(await screen.findByText('1 baris valid siap diimpor')).toBeInTheDocument()
    expect(preview).toHaveBeenCalledWith('csv payload')
    expect(commit).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Konfirmasi impor 1 trade' }))
    await waitFor(() => expect(done).toHaveBeenCalled())
    expect(commit).toHaveBeenCalledWith(
      { csv: 'csv payload', token: 'preview-token' },
      expect.anything(),
    )
  })
  it('submits the last typed lists without waiting for blur and retains old checklist snapshots', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    wrap(
      <TradeForm
        initial={{ ...emptyTrade('a1'), checklist: [{ label: 'Historical rule', checked: true }] }}
        editing
        accounts={[account]}
        pending={false}
        onSave={save}
        onCancel={() => {}}
      />,
    )
    fireEvent.change(screen.getByLabelText('Tag (satu per baris)'), {
      target: { value: 'London\nbreakout' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Advanced: evaluasi & bukti' }))
    expect(screen.getByLabelText('Historical rule')).toBeChecked()
    fireEvent.change(screen.getByLabelText('Emosi (satu per baris)'), {
      target: { value: 'Tenang\nfokus' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({
          tags: ['London', 'breakout'],
          emotions: ['Tenang', 'fokus'],
          checklist: [{ label: 'Historical rule', checked: true }],
        }),
      ),
    )
  })
  it('round trips labels containing commas when saving unrelated trade fields', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const input = {
      ...emptyTrade('a1'),
      tags: ['London, NY', 'Asia'],
      emotions: ['Calm, focused'],
      mistakes: ['Late, rushed'],
      checklist: [{ label: 'London, NY confirmed', checked: true }],
    }
    wrap(
      <TradeForm
        initial={input}
        editing
        accounts={[account]}
        pending={false}
        onSave={save}
        onCancel={() => {}}
      />,
    )
    fireEvent.change(screen.getByLabelText('Simbol'), { target: { value: 'EURUSD' } })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))
    await waitFor(() => expect(save).toHaveBeenCalledWith({ ...input, asset: 'EURUSD' }))
  })
  it('round trips account checklist labels containing commas on an unrelated account edit', async () => {
    const original = { ...account, checklist: ['London, NY', 'Risk, then entry'] }
    const save = vi.spyOn(journalApi, 'saveAccount').mockResolvedValue(original)
    wrap(<AccountManager accounts={[original]} />)
    fireEvent.click(screen.getByRole('button', { name: /^Edit$/ }))
    fireEvent.change(screen.getByLabelText('Nama akun'), { target: { value: 'Renamed' } })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan akun' }))
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({ name: 'Renamed', checklist: original.checklist }),
        account.id,
      ),
    )
  })
  it('edits newline lists while preserving commas inside every custom label', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    wrap(
      <TradeForm
        initial={emptyTrade('a1')}
        accounts={[account]}
        pending={false}
        onSave={save}
        onCancel={() => {}}
      />,
    )
    fireEvent.change(screen.getByLabelText('Tag (satu per baris)'), {
      target: { value: 'London, NY\nAsia' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Advanced: evaluasi & bukti' }))
    fireEvent.change(screen.getByLabelText('Emosi (satu per baris)'), {
      target: { value: 'Calm, focused\nPatient' },
    })
    fireEvent.change(screen.getByLabelText('Kesalahan (satu per baris)'), {
      target: { value: 'Late, rushed' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan trade' }))
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({
          tags: ['London, NY', 'Asia'],
          emotions: ['Calm, focused', 'Patient'],
          mistakes: ['Late, rushed'],
        }),
      ),
    )
  })
  it('edits account checklist one item per line without splitting embedded commas', async () => {
    const save = vi.spyOn(journalApi, 'saveAccount').mockResolvedValue(account)
    wrap(<AccountManager accounts={[account]} />)
    fireEvent.click(screen.getByRole('button', { name: /^Edit$/ }))
    fireEvent.change(screen.getByLabelText('Checklist akun (satu per baris)'), {
      target: { value: 'London, NY\nRisk, then entry' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Simpan akun' }))
    await waitFor(() =>
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({ checklist: ['London, NY', 'Risk, then entry'] }),
        account.id,
      ),
    )
  })
  it('sends emotion and mistake filters consistently and renders losing streak insight generically', async () => {
    vi.mocked(journalApi.insights).mockResolvedValue([
      {
        code: 'losing_streak',
        message: 'Tiga trade terakhir rugi; tinjau batas risiko.',
        count: 3,
      },
    ])
    wrap(<ManualJournalPage />)
    expect(
      await screen.findByText('Tiga trade terakhir rugi; tinjau batas risiko.'),
    ).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Emosi'), { target: { value: 'Calm, focused' } })
    fireEvent.change(screen.getByLabelText('Kesalahan'), { target: { value: 'Late, rushed' } })
    await waitFor(() => {
      for (const method of [journalApi.entries, journalApi.stats, journalApi.insights])
        expect(method).toHaveBeenLastCalledWith(
          expect.objectContaining({ emotion: 'Calm, focused', mistake: 'Late, rushed' }),
          expect.any(AbortSignal),
        )
    })
    const exportSpy = vi.spyOn(journalApi, 'export').mockResolvedValue('manual-journal.csv')
    fireEvent.click(screen.getByRole('button', { name: 'Ekspor CSV' }))
    await waitFor(() =>
      expect(exportSpy).toHaveBeenCalledWith(
        expect.objectContaining({ emotion: 'Calm, focused', mistake: 'Late, rushed' }),
      ),
    )
  })
  it('locks immutable currency while preserving archived account restore controls', () => {
    wrap(<AccountManager accounts={[{ ...account, archived: true }]} />)
    fireEvent.click(screen.getByRole('button', { name: /^Edit$/ }))
    expect(screen.getByLabelText('Mata uang')).toBeDisabled()
    expect(screen.getByLabelText('Arsipkan akun')).toBeChecked()
    fireEvent.click(screen.getByLabelText('Arsipkan akun'))
    expect(screen.getByLabelText('Arsipkan akun')).not.toBeChecked()
  })
  it('uses selected account currency for empty statistics and never combines account currencies', async () => {
    vi.mocked(journalApi.accounts).mockResolvedValue({
      data: [
        { ...account, currency: 'IDR' },
        { ...account, id: 'a2', currency: 'USD' },
      ],
    })
    vi.mocked(journalApi.stats).mockResolvedValue({
      ...stats,
      currency: '',
      trades: 0,
      net_pnl: '0',
      calendar: {},
      breakdowns: {},
    })
    wrap(<ManualJournalPage />)
    expect((await screen.findAllByText(/Rp/)).length).toBeGreaterThan(0)
    expect(journalApi.stats).toHaveBeenCalledWith(
      expect.objectContaining({ account_id: 'a1' }),
      expect.any(AbortSignal),
    )
    expect(screen.queryByRole('option', { name: /Semua akun/ })).not.toBeInTheDocument()
  })
  it('surfaces server failure without pretending the account list is empty', async () => {
    vi.mocked(journalApi.accounts).mockRejectedValue(new Error('Server tidak tersedia'))
    wrap(<ManualJournalPage />)
    expect(await screen.findByText('Server tidak tersedia')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Buat akun jurnal' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Coba lagi' })).toBeInTheDocument()
  })
})
