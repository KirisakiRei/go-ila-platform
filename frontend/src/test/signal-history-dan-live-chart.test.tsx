import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { TimeframeRadarMatrix } from '@/features/signals/components/TimeframeRadarMatrix'
import { TradingRadarChart } from '@/features/charts/TradingRadarChart'
import { dedupeSetupSignals, describeSignalOutcome } from '@/features/signals/lib/outcome'
import { marketApi } from '@/lib/api'
import type { Signal } from '@/types/api'

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries: vi.fn(() => ({
      applyOptions: vi.fn(),
      setData: vi.fn(),
      update: vi.fn(),
      createPriceLine: vi.fn(() => ({})),
      removePriceLine: vi.fn(),
      attachPrimitive: vi.fn(),
    })),
    applyOptions: vi.fn(),
    removeSeries: vi.fn(),
    timeScale: () => ({
      fitContent: vi.fn(),
      getVisibleLogicalRange: () => null,
      setVisibleLogicalRange: vi.fn(),
    }),
    remove: vi.fn(),
  })),
  createSeriesMarkers: vi.fn(() => ({ setMarkers: vi.fn() })),
  CandlestickSeries: 'Candlestick',
  LineSeries: 'Line',
  LineStyle: { Solid: 0, Dashed: 1, Dotted: 2 },
}))

class MockEventSource {
  static instances: MockEventSource[] = []
  url: string
  closed = false
  onerror: ((ev: Event) => void) | null = null
  private listeners = new Map<string, Set<EventListener>>()

  constructor(url: string) {
    this.url = url
    MockEventSource.instances.push(this)
  }

  addEventListener(type: string, listener: EventListener) {
    const set = this.listeners.get(type) ?? new Set<EventListener>()
    set.add(listener)
    this.listeners.set(type, set)
  }

  removeEventListener(type: string, listener: EventListener) {
    this.listeners.get(type)?.delete(listener)
  }

  emit(type: string, data?: unknown) {
    const ev = new MessageEvent(type, {
      data: data !== undefined ? JSON.stringify(data) : '',
    })
    for (const fn of this.listeners.get(type) ?? []) {
      fn(ev)
    }
  }

  close() {
    this.closed = true
  }
}

function makeSignal(over: Partial<Signal> & Pick<Signal, 'signal_id'>): Signal {
  return {
    symbol: 'USTEC',
    strategy_version: 'ila-2.0',
    engine_version: '2.0.0',
    htf: 'H4',
    ltf: 'M15',
    direction: 'SELL',
    entry_price: 27173.4,
    stop_loss: 27288.9,
    take_profit: 26942.4,
    risk_reward: 2.0,
    status: 'EXPIRED',
    summary: 'Liquidity sweep H4',
    next_action: '',
    reject_reason: '',
    published_at: '2026-11-17T08:45:05Z',
    expires_at: '2026-11-17T09:00:05Z',
    can_confirm: false,
    delayed: false,
    ...over,
  }
}

describe('Deduplikasi Histori Setup & Badge TP/SL', () => {
  it('menyatukan terbitan ulang setup identik dan mewariskan status posisi nyata', () => {
    const s1 = makeSignal({
      signal_id: 'sig-1',
      published_at: '2026-11-18T05:35:05Z',
      is_executed: true,
      position_status: 'CLOSED_TP',
      outcome_r: 1.92,
      profit_loss: 207.9,
      pnl_known: true,
    })
    const s2 = makeSignal({
      signal_id: 'sig-2',
      published_at: '2026-11-18T06:00:05Z',
      status: 'EXPIRED',
    })
    const s3 = makeSignal({
      signal_id: 'sig-3',
      published_at: '2026-11-18T06:25:05Z',
      status: 'EXPIRED',
    })

    const deduped = dedupeSetupSignals([s3, s2, s1])
    expect(deduped).toHaveLength(1)
    expect(deduped[0]?.is_executed).toBe(true)

    const badge = describeSignalOutcome(deduped[0]!)
    expect(badge.label).toBe('Hit TP (+1.92R)')
    expect(badge.tone).toBe('tp')
    expect(badge.executedBadge).toBe(true)
  })

  it('menampilkan Hit TP, Hit SL, dan Sedang Berjalan yang seragam untuk sinyal tidak dieksekusi', () => {
    const tpSig = makeSignal({ signal_id: 'tp-1', eval_status: 'TP', eval_r_lower: 2.5 })
    const slSig = makeSignal({ signal_id: 'sl-1', eval_status: 'SL', eval_r_lower: -1 })
    const runSig = makeSignal({ signal_id: 'run-1', eval_status: 'RUNNING' })

    expect(describeSignalOutcome(tpSig).label).toBe('Hit TP (+2.50R)')
    expect(describeSignalOutcome(slSig).label).toBe('Hit SL (-1.00R)')
    expect(describeSignalOutcome(runSig).label).toBe('Sedang Berjalan')
  })
})

describe('Sinkronisasi Timeframe Radar ↔ Chart & Live Stream SSE', () => {
  const origEventSource = window.EventSource

  beforeEach(() => {
    MockEventSource.instances = []
    Object.defineProperty(window, 'EventSource', {
      writable: true,
      configurable: true,
      value: MockEventSource,
    })
    vi.spyOn(marketApi, 'candles').mockImplementation(async (symbol, tf) => ({
      symbol,
      feed: 'FOREXCOM',
      timeframe: tf,
      candles: [{ time: 1_760_000_000, open: 27100, high: 27150, low: 27080, close: 27120, volume: 10 }],
    }))
  })

  afterEach(() => {
    Object.defineProperty(window, 'EventSource', {
      writable: true,
      configurable: true,
      value: origEventSource,
    })
    vi.restoreAllMocks()
  })

  it('mengganti tab timeframe chart saat slot radar dipilih dan mematikan stream pada mode potongan histori', async () => {
    const histSig = makeSignal({
      signal_id: 'sig-hist-1',
      htf: 'H4',
      ltf: 'M15',
      eval_status: 'TP',
      eval_r_lower: 2.0,
      eval_at: '2026-11-17T10:15:00Z',
    })

    function Harness() {
      const [slotKey, setSlotKey] = useState<string | null>('H4-M15')
      const [snap, setSnap] = useState<Signal | null>(null)
      return (
        <div>
          <TimeframeRadarMatrix
            signals={[]}
            strategy="ila-2.0"
            symbol="USTEC"
            activeSlotKey={slotKey}
            onSelectSlot={(k) => {
              setSlotKey(k)
              setSnap(null)
            }}
          />
          <button type="button" onClick={() => setSnap(histSig)}>
            Buka Potongan Histori
          </button>
          <TradingRadarChart
            symbol="USTEC"
            signals={[]}
            strategy="ila-2.0"
            selectedSlotKey={slotKey}
            snapshotSignal={snap}
            onClearSnapshot={() => setSnap(null)}
          />
        </div>
      )
    }

    const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
    render(
      <QueryClientProvider client={qc}>
        <Harness />
      </QueryClientProvider>,
    )

    // Awalnya memilih H4-M15 -> tab chart harus H4 Setup & M15 Trigger.
    expect(screen.getByRole('button', { name: 'H4 Setup' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'M15 Trigger' })).toBeInTheDocument()
    expect(MockEventSource.instances.at(-1)?.url).toContain('tf=H4')

    act(() => {
      MockEventSource.instances.at(-1)?.emit('ready', { symbol: 'USTEC', feed: 'FOREXCOM', timeframe: 'H4' })
    })
    expect(screen.getByText('Live')).toBeInTheDocument()

    // Ganti klik slot H1 · M5 di Radar Matrix -> chart langsung ikut berganti ke H1 Setup & M5 Trigger!
    fireEvent.click(screen.getByRole('button', { name: /H1 · M5/i }))
    expect(screen.getByRole('button', { name: 'H1 Setup' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'M5 Trigger' })).toBeInTheDocument()
    expect(MockEventSource.instances.at(-1)?.url).toContain('tf=H1')

    // Buka potongan histori -> koneksi SSE ditutup, muncul label Potongan Histori & Statis.
    const prevStream = MockEventSource.instances.at(-1)
    fireEvent.click(screen.getByRole('button', { name: 'Buka Potongan Histori' }))
    expect(prevStream?.closed).toBe(true)
    expect(screen.getByText(/Potongan Histori · SELL H4\/M15/)).toBeInTheDocument()
    expect(screen.getByText('Hit TP (+2.00R)')).toBeInTheDocument()
    expect(screen.getByText('Statis')).toBeInTheDocument()
  })
})
