import { describe, expect, it, vi } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen } from '@testing-library/react'
import { JournalSummary } from '@/features/journal/components/JournalSummary'
import { formatDateTime, formatNumber, formatPrice } from '@/lib/format'
import { JournalSkippedList } from '@/features/journal/components/JournalSkippedList'
import { SimulasiBadge } from '@/features/journal/components/SimulasiBadge'
import type { JournalSkipped, JournalStats } from '@/types/api'

vi.mock('@/features/charts/SkippedEvidenceChart', () => ({
  SkippedEvidenceChart: () => <div data-testid="skipped-chart">Chart simulasi</div>,
}))

/** statsDasar membuat ringkasan dengan nilai yang bisa ditimpa per uji. */
function statsDasar(overrides: Partial<JournalStats> = {}): JournalStats {
  return {
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
    ...overrides,
  }
}

describe('JournalSummary', () => {
  it('menampilkan tanda pisah saat belum ada posisi yang ditutup', () => {
    render(<JournalSummary stats={statsDasar({ open_positions: 2, total_positions: 2 })} />)

    // Winrate belum bisa dihitung: harus "—", bukan "0%" yang menyesatkan.
    expect(screen.getByText('Winrate')).toBeInTheDocument()
    expect(screen.getByText('Belum ada posisi yang ditutup')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('menghitung winrate hanya dari penutupan yang punya hasil', () => {
    render(
      <JournalSummary
        stats={statsDasar({
          total_positions: 4,
          closed_tp: 1,
          closed_sl: 1,
          closed_manual: 1,
          wins: 1,
          losses: 1,
          win_rate: 50,
          total_r: 1.5,
          average_r: 0.75,
          total_profit: 50,
        })}
      />,
    )

    // 1 menang dari 2 yang hasilnya diketahui; tutup manual tanpa laporan
    // broker tidak boleh menurunkan winrate.
    // Format mengikuti locale aplikasi (id-ID): "50,0%", bukan "50.0%".
    expect(screen.getByText(`${formatNumber(50, 1)}%`)).toBeInTheDocument()
    expect(screen.getByText('1 menang dari 2 ditutup')).toBeInTheDocument()
  })

  it('menandai jumlah posisi simulasi secara terpisah', () => {
    render(<JournalSummary stats={statsDasar({ total_positions: 3, dry_run_positions: 2 })} />)

    expect(screen.getByText('Termasuk simulasi')).toBeInTheDocument()
  })
})

describe('SimulasiBadge', () => {
  it('menjelaskan bahwa posisi simulasi bukan uang sungguhan', () => {
    render(<SimulasiBadge />)

    expect(screen.getByText('Simulasi')).toBeInTheDocument()
    // Keterangan tersedia untuk pembaca layar, bukan hanya lewat tooltip.
    expect(
      screen.getByText(/tidak dikirim ke broker, hasilnya dihitung dari harga pasar/i),
    ).toBeInTheDocument()
  })
})

describe('JournalSkippedList', () => {
  const terlewat: JournalSkipped = {
    signal_id: 'sig-uji-1',
    symbol: 'XAUUSD',
    strategy_version: 'ila-2.0',
    direction: 'BUY',
    htf: 'H4',
    ltf: 'M15',
    entry_price: 4051.315,
    stop_loss: 4013.705,
    take_profit: 4145.34,
    risk_reward: 2.5,
    status: 'REJECTED',
    reason: 'Harga sudah bergerak terlalu jauh dari level entry.',
    occurred_at: '2026-11-12T11:46:03Z',
    occurred_at_pair: {
      utc: '2026-11-12 11:46 UTC',
      wib: '2026-11-12 18:46 WIB',
      ny: '2026-11-12 06:46 NY',
    },
  }

  it('menampilkan alasan penolakan apa adanya agar bisa dipahami trader', () => {
    render(<JournalSkippedList items={[terlewat]} />)

    expect(screen.getByText('XAUUSD')).toBeInTheDocument()
    expect(screen.getByText('H4 → M15')).toBeInTheDocument()
    expect(screen.getByText('Tidak tereksekusi')).toBeInTheDocument()
    expect(
      screen.getByText('Harga sudah bergerak terlalu jauh dari level entry.'),
    ).toBeInTheDocument()
  })

  it('memberi keterangan bawaan saat peluang menunggu keputusan', () => {
    render(
      <JournalSkippedList
        items={[{ ...terlewat, signal_id: 'sig-uji-2', status: 'OPPORTUNITY', reason: '' }]}
      />,
    )

    expect(screen.getByText('Menunggu keputusan')).toBeInTheDocument()
    expect(screen.getByText(/Eksekusi otomatis tidak aktif/i)).toBeInTheDocument()
  })

  it.each([
    ['PENDING', 'Menunggu penilaian'],
    ['RUNNING', 'Sedang berjalan'],
    ['TP', 'TP (+2.00R)'],
    ['SL', 'SL (-1R)'],
    ['NO_ENTRY', 'Tidak masuk'],
    ['UNKNOWN', 'Tidak dapat dipastikan'],
  ] as const)('menampilkan status simulasi %s tanpa menimpa alasan akun', (eval_status, label) => {
    render(
      <JournalSkippedList
        items={[
          {
            ...terlewat,
            eval_status,
            eval_r_lower: 2,
            eval_note: 'Catatan penilaian harga.',
          },
        ]}
      />,
    )

    expect(screen.getByText(`Simulasi · ${label}`)).toBeInTheDocument()
    expect(screen.getByText(terlewat.reason!)).toBeInTheDocument()
    expect(screen.getByText('Catatan simulasi: Catatan penilaian harga.')).toBeInTheDocument()
    expect(screen.getByText(/tidak menjadi transaksi akun.*Bukan PnL akun/)).toBeInTheDocument()
    expect(screen.queryByText(/\+\+/)).not.toBeInTheDocument()
  })

  it('menampilkan waktu WIB, harga sesuai pair, dan exit legacy', () => {
    const entry = '2026-11-12T12:00:00Z'
    const exit = '2026-11-12T14:30:00Z'
    render(
      <JournalSkippedList
        items={[
          {
            ...terlewat,
            eval_status: 'TP',
            eval_entry_at: entry,
            eval_at: exit,
            eval_entry_price: 4051.32,
            eval_price: 4145.34,
          },
        ]}
      />,
    )

    expect(
      screen.getByText(`Waktu sinyal: ${formatDateTime(terlewat.occurred_at)}`),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        `Entry simulasi: ${formatDateTime(entry)} · Harga ${formatPrice(4051.32, 2)}`,
      ),
    ).toBeInTheDocument()
    expect(
      screen.getByText(`Exit simulasi: ${formatDateTime(exit)} · Harga ${formatPrice(4145.34, 2)}`),
    ).toBeInTheDocument()
    expect(screen.getByText('Durasi simulasi: 2 jam 30 menit')).toBeInTheDocument()
    expect(screen.getByText(/Rencana entry/)).toHaveTextContent(
      formatPrice(terlewat.entry_price, 2),
    )
    expect(screen.getByText(/Rencana entry/)).not.toHaveTextContent('4051,31500')
  })

  it('entry forward memakai level publikasi dan tidak mengklaim fill broker', () => {
    render(
      <JournalSkippedList
        items={[
          {
            ...terlewat,
            eval_status: 'RUNNING',
            eval_evidence: '{"method":"signal_forward_v1"}',
            eval_entry_at: terlewat.occurred_at,
          },
        ]}
      />,
    )
    expect(screen.getByText(/Entry simulasi \(teoritis\):/)).toHaveTextContent(
      formatPrice(terlewat.entry_price, 2),
    )
    expect(screen.getByText(/bukan fill broker atau waktu retest/)).toBeInTheDocument()
  })

  it.each([undefined, '', '{invalid', 'null', '{"method":"legacy"}'])(
    'evidence lama atau invalid %s tidak diklaim teoritis',
    (eval_evidence) => {
      render(
        <JournalSkippedList
          items={[
            {
              ...terlewat,
              eval_evidence,
              eval_entry_at: terlewat.occurred_at,
            },
          ]}
        />,
      )
      expect(
        screen.getByText(`Entry simulasi: ${formatDateTime(terlewat.occurred_at)}`),
      ).toBeInTheDocument()
      expect(screen.queryByText(/Entry simulasi \(teoritis\)/)).not.toBeInTheDocument()
    },
  )

  it('mengutamakan harga exit additive dibanding field legacy', () => {
    render(
      <JournalSkippedList
        items={[
          {
            ...terlewat,
            eval_status: 'TP',
            eval_price: 1,
            eval_exit_price: 4145.34,
          },
        ]}
      />,
    )
    expect(screen.getByText(/Exit simulasi:/)).toHaveTextContent(formatPrice(4145.34, 2))
  })

  it('durasi running dibatasi penilaian terakhir dan eval_at bukan exit', () => {
    render(
      <JournalSkippedList
        items={[
          {
            ...terlewat,
            eval_status: 'RUNNING',
            eval_entry_at: '2026-11-12T12:00:00Z',
            eval_checked_at: '2026-11-12T14:30:00Z',
            eval_at: '2026-11-12T14:00:00Z',
            eval_price: 4100,
          },
        ]}
      />,
    )
    expect(screen.getByText(/Durasi berjalan:/)).toHaveTextContent('2 jam 30 menit')
    expect(screen.getByText('Exit simulasi: —')).toBeInTheDocument()
    expect(screen.getByText(/Terakhir dinilai:/)).toHaveTextContent('WIB')
  })

  it('tidak mengarang durasi atau harga entry pada DTO lama', () => {
    render(
      <JournalSkippedList
        items={[
          {
            ...terlewat,
            eval_status: 'RUNNING',
            eval_entry_at: 'invalid',
          },
        ]}
      />,
    )
    expect(screen.getByText('Entry simulasi: —')).toBeInTheDocument()
    expect(screen.getByText(/Durasi berjalan:/)).toHaveTextContent('Durasi berjalan: —')
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument()
  })

  it('membuka chart dengan keyboard dan hanya satu detail sekaligus', async () => {
    const user = userEvent.setup()
    render(<JournalSkippedList items={[terlewat, { ...terlewat, signal_id: 'sig-2' }]} />)
    expect(screen.queryByTestId('skipped-chart')).not.toBeInTheDocument()
    const buttons = screen.getAllByRole('button', { name: 'Lihat bukti harga' })
    await user.tab()
    expect(buttons[0]).toHaveFocus()
    await user.keyboard('{Enter}')
    expect(buttons[0]).toHaveAttribute('aria-expanded', 'true')
    expect(document.getElementById(buttons[0]!.getAttribute('aria-controls')!)).toContainElement(
      screen.getByTestId('skipped-chart'),
    )
    await user.click(buttons[1]!)
    expect(screen.getAllByTestId('skipped-chart')).toHaveLength(1)
    expect(buttons[0]).toHaveAttribute('aria-expanded', 'false')
    await user.click(buttons[1]!)
    expect(screen.queryByTestId('skipped-chart')).not.toBeInTheDocument()
  })

  it('menampilkan keadaan kosong yang menjelaskan', () => {
    render(<JournalSkippedList items={[]} />)

    expect(screen.getByText('Tidak ada peluang terlewat')).toBeInTheDocument()
  })
})
