import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import AssistantPage from '@/pages/AssistantPage'
import { MarkdownContent } from '@/features/assistant/MarkdownContent'
import { AppShell } from '@/components/layout/AppShell'
import { AuthProvider } from '@/features/auth/AuthProvider'
import { authApi } from '@/lib/auth-api'
import { queryKeys } from '@/lib/queries'
import * as assistantApiModule from '@/features/assistant/api'
import type { User } from '@/types/api'

const mockUser: User = {
  id: 'u1',
  name: 'Ryan',
  username: 'ryan',
  phone: '+628123456789',
  email: 'ryan@example.com',
  role: 'user',
  tier: 'plus',
  idle_warning: false,
  email_verified_at: '2026-01-01T00:00:00Z',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('Halaman ILA AI (/asisten)', () => {
  beforeEach(() => {
    Element.prototype.scrollTo = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('menampilkan menu ILA AI di sidebar AppShell', () => {
    vi.spyOn(authApi, 'me').mockResolvedValue(mockUser)
    vi.spyOn(authApi, 'csrf').mockResolvedValue({ csrf_token: 'test' })
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={['/dashboard']}>
          <AuthProvider>
            <AppShell user={mockUser}>
              <div>Konten</div>
            </AppShell>
          </AuthProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    )

    const links = screen.getAllByRole('link', { name: /ila ai/i })
    expect(links.length).toBeGreaterThanOrEqual(1)
    for (const link of links) {
      expect(link).toHaveAttribute('href', '/asisten')
    }
  })

  it('menjalankan alur thinking, tool call, streaming, dan invalidate subscriptions', async () => {
    vi.spyOn(assistantApiModule.assistantApi, 'getSession').mockResolvedValue({
      enabled: true,
      session_id: 'sess-1',
      compacted_before_seq: 0,
      active_message_count: 0,
      has_summary: false,
      has_more: false,
      messages: [],
    })

    const streamSpy = vi
      .spyOn(assistantApiModule, 'streamAssistantChat')
      .mockImplementation(async ({ onEvent }) => {
        onEvent({ type: 'status', stage: 'thinking', label: 'Sedang berpikir...' })
        onEvent({ type: 'tool_start', name: 'subscribe_pair', label: 'Menambahkan USDJPY ke pantauan...' })
        onEvent({
          type: 'tool_done',
          name: 'subscribe_pair',
          ok: true,
          summary: 'USDJPY ditambahkan ke pantauan',
        })
        onEvent({ type: 'action_applied', kind: 'subscription_added', symbol: 'USDJPY' })
        onEvent({ type: 'status', stage: 'streaming', label: 'Menyusun jawaban...' })
        onEvent({ type: 'delta', text: 'Pair **USDJPY** sudah masuk watchlist Anda.' })
        onEvent({ type: 'done', message_id: 'msg-2', seq: 2, compacted: false })
      })

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

    render(
      <QueryClientProvider client={queryClient}>
        <AssistantPage />
      </QueryClientProvider>,
    )

    const starter = await screen.findByRole('button', {
      name: /analisa sinyal xauusd h4 terakhir/i,
    })
    expect(starter).toBeInTheDocument()

    const textarea = screen.getByLabelText(/pesan untuk ila ai/i)
    fireEvent.change(textarea, { target: { value: 'Pantau pair USDJPY' } })
    fireEvent.click(screen.getByRole('button', { name: /kirim pesan/i }))

    await waitFor(() => {
      expect(streamSpy).toHaveBeenCalledTimes(1)
    })

    expect(await screen.findByText(/sudah masuk watchlist Anda/i)).toBeInTheDocument()
    expect(screen.getByText(/USDJPY ditambahkan ke pantauan/i)).toBeInTheDocument()
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: queryKeys.subscriptions })
  })

  it('merender markdown h4, hashtag, dan garis pemisah tanpa infinite loop atau OOM', () => {
    render(
      <MarkdownContent
        content={[
          '#### Ringkasan Setup',
          '#W525654572960402 posisi SELL',
          '#1 batas harian',
          '| Pair | Arah | tanpa garis pemisah',
          '####### tujuh pagar',
        ].join('\n')}
      />,
    )

    expect(screen.getByText('Ringkasan Setup')).toBeInTheDocument()
    expect(screen.getByText(/#W525654572960402 posisi SELL/)).toBeInTheDocument()
    expect(screen.getByText(/#1 batas harian/)).toBeInTheDocument()
    expect(screen.getByText(/####### tujuh pagar/)).toBeInTheDocument()
  })

  it('memuat riwayat chat sebelumnya per halaman dengan before_seq tanpa duplikasi', async () => {
    const getSessionSpy = vi
      .spyOn(assistantApiModule.assistantApi, 'getSession')
      .mockResolvedValueOnce({
        enabled: true,
        session_id: 'sess-1',
        compacted_before_seq: 0,
        active_message_count: 25,
        has_summary: false,
        has_more: true,
        messages: [
          {
            id: 'm21',
            seq: 21,
            role: 'user',
            content: 'Pesan terbaru halaman 1',
            created_at: '2026-11-23T12:00:00Z',
          },
        ],
      })
      .mockResolvedValueOnce({
        enabled: true,
        session_id: 'sess-1',
        compacted_before_seq: 0,
        active_message_count: 25,
        has_summary: false,
        has_more: false,
        messages: [
          {
            id: 'm10',
            seq: 10,
            role: 'user',
            content: 'Pesan lama halaman 2',
            created_at: '2026-11-23T11:00:00Z',
          },
          {
            id: 'm21-dup',
            seq: 21,
            role: 'user',
            content: 'Pesan terbaru halaman 1',
            created_at: '2026-11-23T12:00:00Z',
          },
        ],
      })

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
    render(
      <QueryClientProvider client={queryClient}>
        <AssistantPage />
      </QueryClientProvider>,
    )

    expect(await screen.findByText('Pesan terbaru halaman 1')).toBeInTheDocument()
    const loadOlderBtn = screen.getByRole('button', { name: /muat riwayat sebelumnya/i })
    fireEvent.click(loadOlderBtn)

    expect(await screen.findByText('Pesan lama halaman 2')).toBeInTheDocument()
    expect(screen.getAllByText('Pesan terbaru halaman 1')).toHaveLength(1)
    expect(getSessionSpy).toHaveBeenNthCalledWith(1, { limit: 20 })
    expect(getSessionSpy).toHaveBeenNthCalledWith(2, { beforeSeq: 21, limit: 20 })
  })

  it('merender placeholder streaming dan diagram SVG skematik untuk blok ila-diagram', () => {
    const streaming = '```ila-diagram\n{"title": "C2 Closure"'
    const { rerender } = render(<MarkdownContent content={streaming} />)
    expect(screen.getByText(/menyiapkan ilustrasi candlestick/i)).toBeInTheDocument()

    const completed = [
      '```ila-diagram',
      JSON.stringify({
        title: 'Contoh C2 Closure ERL',
        subtitle: 'HTF H4',
        candles: [
          { o: 100, h: 104, l: 99, c: 103, label: 'C1' },
          { o: 103, h: 106, l: 101, c: 102, marker: 'C2 Sweep' },
          { o: 102, h: 103, l: 97, c: 98, label: 'C3' },
        ],
        levels: [
          { price: 104, label: 'Swing High ERL', kind: 'sweep' },
          { price: 98, label: 'Entry', kind: 'entry' },
        ],
        zones: [{ from_idx: 0, to_idx: 2, top: 101, bottom: 100, label: 'FVG' }],
      }),
      '```',
    ].join('\n')

    rerender(<MarkdownContent content={completed} />)
    expect(screen.getByRole('img', { name: /contoh c2 closure erl/i })).toBeInTheDocument()
    expect(screen.getByText('Swing High ERL')).toBeInTheDocument()
    expect(screen.getByText('C2 Sweep')).toBeInTheDocument()

    rerender(<MarkdownContent content={'```ila-diagram\n{json rusak}\n```'} />)
    expect(screen.getByText(/ilustrasi tidak dapat ditampilkan/i)).toBeInTheDocument()
  })

  it('menahan pemuatan canvas chart TradingView pada pesan lama sampai diklik pengguna', () => {
    const chartMd = [
      '```ila-chart',
      JSON.stringify({
        title: 'XAUUSD (M15) — Setup SELL',
        symbol: 'XAUUSD',
        timeframe: 'M15',
        lines: [{ price: 3615.03, label: 'Entry', kind: 'entry' }],
      }),
      '```',
    ].join('\n')

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
    render(
      <QueryClientProvider client={queryClient}>
        <MarkdownContent content={chartMd} autoLoadChart={false} />
      </QueryClientProvider>,
    )

    expect(screen.getByText('XAUUSD (M15) — Setup SELL')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /tampilkan chart/i })).toBeInTheDocument()
  })
})
