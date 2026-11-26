import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { JournalAIPreferences, JournalReview } from './JournalReview'
import { journalApi } from './api'
import { journalKey, reviewPollInterval, useJournalMutation } from './hooks'
import { wibDay } from './model'
import type { JournalReview as Review } from './types'

const today = wibDay(new Date().toISOString())
const ready: Review = {
  account_id: 'a1',
  date: today,
  revision: 'r1',
  status: 'ready',
  source: 'rules',
  text: 'Tetap jaga batas risiko.',
  updated_at: new Date().toISOString(),
}

function mount(component: React.ReactNode = <JournalReview accountId="a1" />) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return {
    client,
    ...render(<QueryClientProvider client={client}>{component}</QueryClientProvider>),
  }
}

beforeEach(() => {
  vi.restoreAllMocks()
  vi.spyOn(journalApi, 'review').mockResolvedValue(ready)
  vi.spyOn(journalApi, 'reviewSettings').mockImplementation(async (account) => ({
    account_id: account ?? '',
    ai_enabled: true,
    effective_ai_enabled: false,
  }))
  vi.spyOn(journalApi, 'saveReviewSettings').mockResolvedValue({
    account_id: 'a1',
    ai_enabled: false,
    effective_ai_enabled: false,
  })
})

afterEach(() => vi.useRealTimers())

describe('manual journal review', () => {
  it('polls outstanding work quickly and refreshes ready cache once a minute for WIB rollover', () => {
    expect(reviewPollInterval({ status: 'pending' })).toBe(5000)
    expect(reviewPollInterval({ status: 'running' })).toBe(5000)
    expect(reviewPollInterval({ status: 'ready' })).toBe(60000)
    expect(reviewPollInterval()).toBe(false)
  })

  it('renders a compact 1-line insight bar and refreshes across midnight WIB', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-11-26T16:59:30Z'))
    vi.mocked(journalApi.review)
      .mockResolvedValueOnce({ ...ready, date: '2026-11-26', text: 'Insight hari pertama.' })
      .mockResolvedValue({
        ...ready,
        date: '2026-11-27',
        revision: 'next-day',
        text: 'Insight hari kedua.',
      })

    const view = mount()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
    })
    expect(screen.getByText('Insight hari pertama.')).toBeInTheDocument()
    expect(screen.getByText('Aturan')).toBeInTheDocument()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(60001)
    })
    expect(journalApi.review).toHaveBeenCalledTimes(2)
    expect(screen.getByText('Insight hari kedua.')).toBeInTheDocument()
    view.unmount()
  })

  it('shows pending fallback, source badge and hover tooltip metadata', async () => {
    vi.mocked(journalApi.review).mockResolvedValue({
      ...ready,
      status: 'running',
      text: '',
      source: '',
    })
    const onOpenSettings = vi.fn()
    const view = mount(<JournalReview accountId="a1" onOpenSettings={onOpenSettings} />)
    expect(await screen.findByText(/Menyiapkan insight harian/)).toBeInTheDocument()
    expect(screen.getByText(/Menunggu/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Info insight harian' }))
    expect(screen.getByRole('tooltip')).toHaveTextContent(/WIB/)
    fireEvent.click(screen.getByRole('button', { name: 'Atur' }))
    expect(onOpenSettings).toHaveBeenCalledOnce()
    view.unmount()
  })

  it('lets users toggle global and journal AI preferences inside settings modal', async () => {
    mount(<JournalAIPreferences accountId="a1" />)
    const accountToggle = await screen.findByLabelText(/Aktifkan AI untuk jurnal ini/)
    await waitFor(() => expect(accountToggle).toBeEnabled())
    fireEvent.click(accountToggle)
    await waitFor(() => expect(journalApi.saveReviewSettings).toHaveBeenCalledWith(false, 'a1'))

    const globalToggle = screen.getByLabelText(/Aktifkan insight AI/)
    await waitFor(() => expect(globalToggle).toBeEnabled())
    fireEvent.click(globalToggle)
    await waitFor(() =>
      expect(journalApi.saveReviewSettings).toHaveBeenCalledWith(false, undefined),
    )
    expect(screen.getByText(/Hanya aturan/)).toBeInTheDocument()
    expect(screen.getByText(/tetap privat/)).toBeInTheDocument()
  })

  it('does not allow preference writes when settings failed to load', async () => {
    vi.mocked(journalApi.reviewSettings).mockRejectedValue(new Error('Pengaturan tidak tersedia'))
    mount(<JournalAIPreferences accountId="a1" />)
    expect(await screen.findByText('Pengaturan tidak tersedia')).toBeInTheDocument()
    expect(screen.getByLabelText(/Aktifkan AI untuk jurnal ini/)).toBeDisabled()
  })

  it('invalidates cached review through the existing journal mutation key', async () => {
    function Mutation() {
      const mutation = useJournalMutation(async () => undefined)
      return <button onClick={() => mutation.mutate()}>Simpan jurnal</button>
    }
    const view = mount(<Mutation />)
    view.client.setQueryData([...journalKey, 'review', 'a1'], ready)
    fireEvent.click(screen.getByText('Simpan jurnal'))
    await waitFor(() =>
      expect(view.client.getQueryState([...journalKey, 'review', 'a1'])?.isInvalidated).toBe(true),
    )
  })
})
