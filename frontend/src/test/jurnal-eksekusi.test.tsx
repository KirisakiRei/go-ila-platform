import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { JournalSummary } from '@/features/journal/components/JournalSummary'
import { formatNumber } from '@/lib/format'
import { JournalSkippedList } from '@/features/journal/components/JournalSkippedList'
import { SimulasiBadge } from '@/features/journal/components/SimulasiBadge'
import type { JournalSkipped, JournalStats } from '@/types/api'

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
        items={[
          { ...terlewat, signal_id: 'sig-uji-2', status: 'OPPORTUNITY', reason: '' },
        ]}
      />,
    )

    expect(screen.getByText('Menunggu keputusan')).toBeInTheDocument()
    expect(screen.getByText(/Eksekusi otomatis tidak aktif/i)).toBeInTheDocument()
  })

  it('menampilkan keadaan kosong yang menjelaskan', () => {
    render(<JournalSkippedList items={[]} />)

    expect(screen.getByText('Tidak ada peluang terlewat')).toBeInTheDocument()
  })
})
