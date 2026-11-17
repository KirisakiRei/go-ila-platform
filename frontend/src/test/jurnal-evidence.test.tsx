import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { JournalSkippedList } from '@/features/journal/components/JournalSkippedList'
import { formatDateTime } from '@/lib/format'
import type { JournalSkipped } from '@/types/api'

vi.mock('@/features/charts/SkippedEvidenceChart', () => ({
  SkippedEvidenceChart: () => <p>Data candle sedang tidak tersedia.</p>,
}))

const item: JournalSkipped = {
  signal_id: 'persisted',
  symbol: 'XAUUSD',
  strategy_version: 'ila-2.0',
  direction: 'BUY',
  htf: 'H4',
  ltf: 'M15',
  entry_price: 4050,
  stop_loss: 4000,
  take_profit: 4150,
  risk_reward: 2,
  status: 'REJECTED',
  occurred_at: '2026-11-12T12:00:00Z',
  occurred_at_pair: { utc: '', wib: '', ny: '' },
  eval_status: 'TP',
  eval_entry_at: '2026-11-12T12:00:00Z',
}
const candle = { time: 1794485700, open: 4050, high: 4150, low: 4040, close: 4140, volume: 12 }
// Shape serialized by SignalEvalRepository.Save, including a JSON-encoded evaluation string.
const persisted = {
  setup: { symbol: 'XAUUSD', entry_price: 4050, stop_loss: 4000, take_profit: 4150 },
  entry_at: item.eval_entry_at,
  entry_price: 4050,
  entry_basis: 'signal_level_at_publication',
  evaluation: JSON.stringify({ method: 'signal_forward_v1', candle_seconds: 900, candle }),
}

function renderEvidence(evidence: unknown) {
  render(<JournalSkippedList items={[{ ...item, eval_evidence: JSON.stringify(evidence) }]} />)
}

describe('Persisted journal evidence', () => {
  it('recognizes the real envelope and keeps stored candle proof when chart is unavailable', async () => {
    const user = userEvent.setup()
    renderEvidence(persisted)
    expect(screen.getByText(/Entry simulasi \(teoritis\):/)).toHaveTextContent('Harga 4050,00')
    expect(screen.queryByText(/Candle tersimpan:/)).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Lihat bukti harga' }))
    expect(screen.getByText('Data candle sedang tidak tersedia.')).toBeInTheDocument()
    const proof = screen.getByText(/Candle tersimpan:/)
    expect(proof).toHaveTextContent(formatDateTime(new Date(candle.time * 1000).toISOString()))
    expect(proof).toHaveTextContent('O 4050,00 · H 4150,00 · L 4040,00 · C 4140,00')
    expect(screen.queryByText(/candle_seconds|entry_basis/)).not.toBeInTheDocument()
  })

  it('recognizes nested evaluation without entry_basis', () => {
    renderEvidence({ evaluation: persisted.evaluation })
    expect(screen.getByText(/Entry simulasi \(teoritis\):/)).toBeInTheDocument()
  })

  it.each([
    { entry_at: item.eval_entry_at, entry_price: 4050 },
    { evaluation: JSON.stringify({ method: 'legacy', candle }) },
    { entry_basis: 'legacy_retest', evaluation: persisted.evaluation },
    { evaluation: '{broken' },
    { evaluation: null },
    { evaluation: [] },
    null,
  ])('does not relabel legacy or malformed evidence as theoretical: %j', (evidence) => {
    renderEvidence(evidence)
    expect(
      screen.getByText(`Entry simulasi: ${formatDateTime(item.eval_entry_at)}`),
    ).toBeInTheDocument()
    expect(screen.queryByText(/Entry simulasi \(teoritis\):/)).not.toBeInTheDocument()
  })

  it.each([
    '{broken',
    JSON.stringify({ method: 'signal_forward_v1', incomplete: true }),
    JSON.stringify({ candle: { ...candle, time: 1e99 } }),
    JSON.stringify({ candle: { ...candle, high: '4150' } }),
    JSON.stringify({ candle: { ...candle, low: 5000 } }),
  ])('invalid or incomplete candle evidence degrades safely: %s', async (evaluation) => {
    const user = userEvent.setup()
    renderEvidence({ ...persisted, evaluation })
    await user.click(screen.getByRole('button', { name: 'Lihat bukti harga' }))
    expect(screen.getByText('Bukti candle tersimpan tidak tersedia.')).toBeInTheDocument()
    expect(screen.queryByText(/NaN|Invalid Date/)).not.toBeInTheDocument()
    if (evaluation.includes('incomplete')) {
      expect(screen.getByText(/Bukti tidak lengkap/)).toBeInTheDocument()
    }
  })
})
