import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TimeframeRadarMatrix } from '@/features/signals/components/TimeframeRadarMatrix'
import type { Signal } from '@/types/api'

/**
 * sinyal membuat objek Signal minimal. Hanya kolom yang dibaca matriks diisi.
 */
function sinyal(over: Partial<Signal> & Pick<Signal, 'signal_id'>): Signal {
  return {
    symbol: 'XAUUSD',
    strategy_version: 'ila-2.0',
    engine_version: '2.0.0',
    htf: 'H4',
    ltf: 'M15',
    direction: 'SELL',
    entry_price: 0,
    stop_loss: 0,
    take_profit: 0,
    risk_reward: 0,
    status: 'WAIT_HTF_SWEEP',
    summary: '',
    next_action: '',
    reject_reason: '',
    published_at: '2026-11-09T14:59:00Z',
    updated_at: '2026-11-13T08:00:00Z',
    expires_at: '2026-11-13T09:00:00Z',
    can_confirm: false,
    delayed: false,
    ...over,
  }
}

describe('TimeframeRadarMatrix — penanda tidak dipantau', () => {
  it('sinyal lama tidak menutupi status tidak dipantau', () => {
    // Skenario produksi: slot dimatikan pada 11 Nov tetapi sinyal lama masih
    // tersimpan. Sebelum perbaikan, sinyal itulah yang tampil sehingga radar
    // terbaca sedang memindai padahal engine sudah berhenti.
    const lama = sinyal({ signal_id: 'status-XAUUSD-H4-M15' })

    render(
      <TimeframeRadarMatrix
        signals={[lama]}
        strategy="ila-2.0"
        symbol="XAUUSD"
        // Semua dimatikan: kondisi paling jelas untuk memastikan penanda muncul.
        timeframeAktif={new Set()}
      />,
    )

    expect(screen.getByRole('button', { name: /aktifkan pemantauan/i })).toBeInTheDocument()
  })

  it('slot yang dipantau tetap menampilkan sinyalnya', () => {
    const aktif = sinyal({ signal_id: 'status-XAUUSD-H4-M15' })

    render(
      <TimeframeRadarMatrix
        signals={[aktif]}
        strategy="ila-2.0"
        symbol="XAUUSD"
        // Seluruh slot dipantau: tidak boleh ada tuduhan "dimatikan" di mana pun.
        timeframeAktif={new Set(['W|H4', 'D1|H1', 'H4|M15', 'H1|M5'])}
      />,
    )

    // Status tidak diketahui (null) maupun dipantau tidak boleh memunculkan
    // tombol "aktifkan" — itu akan menuduh pengguna mematikan sesuatu,
    // padahal radar berjalan normal.
    expect(screen.queryByRole('button', { name: /aktifkan pemantauan/i })).not.toBeInTheDocument()
  })

  it('status belum diketahui tidak pernah menuduh slot dimatikan', () => {
    const aktif = sinyal({ signal_id: 'status-XAUUSD-H4-M15' })

    render(
      <TimeframeRadarMatrix signals={[aktif]} strategy="ila-2.0" symbol="XAUUSD" timeframeAktif={null} />,
    )

    // `null` berarti server belum mengirim matriks saklar. Menampilkan tombol
    // "aktifkan" di sini akan menyesatkan pengguna.
    expect(screen.queryByRole('button', { name: /aktifkan pemantauan/i })).not.toBeInTheDocument()
  })

  it('menampilkan ajakan mengaktifkan saat seluruh timeframe dimatikan', () => {
    render(
      <TimeframeRadarMatrix
        signals={[]}
        strategy="ila-2.0"
        symbol="XAUUSD"
        timeframeAktif={new Set()}
      />,
    )

    expect(screen.getByText(/belum dipantau/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /aktifkan pemantauan/i })).toBeInTheDocument()
  })
})
