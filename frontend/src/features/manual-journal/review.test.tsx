import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { JournalReview } from './JournalReview'
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
function mount(component = <JournalReview accountId="a1" />) {
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
  it('refreshes a mounted ready review across midnight WIB and shows the next daily banner', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-11-26T16:59:30Z'))
    vi.mocked(journalApi.review)
      .mockResolvedValueOnce({ ...ready, date: '2026-11-26' })
      .mockResolvedValue({ ...ready, date: '2026-11-27', revision: 'next-day' })
    const view = mount()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
    })
    expect(screen.getByRole('complementary')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Tutup pengingat review hari ini' }))
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60001)
    })
    expect(journalApi.review).toHaveBeenCalledTimes(2)
    expect(screen.getByRole('complementary')).toBeInTheDocument()
    expect(screen.getByText(/Tanggal review 2026-11-27/)).toBeInTheDocument()
    view.unmount()
  })
  it('shows pending fallback, source and WIB timestamp without creating raw AI requests', async () => {
    vi.mocked(journalApi.review).mockResolvedValue({
      ...ready,
      status: 'running',
      text: '',
      source: '',
    })
    const view = mount()
    expect(await screen.findByText(/Review sedang disiapkan/)).toBeInTheDocument()
    expect(screen.getByText(/Sumber: menunggu/)).toBeInTheDocument()
    expect(screen.getByText(/Tanggal review .* WIB · Diperbarui/)).toBeInTheDocument()
    expect(journalApi.review).toHaveBeenCalledWith('a1', expect.any(AbortSignal))
    view.unmount()
  })
  it('persists daily dismissal per account and WIB date, not revision', async () => {
    const view = mount()
    expect(
      await screen.findByRole('complementary', { name: 'Pengingat review hari ini' }),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Tutup pengingat review hari ini' }))
    expect(localStorage.getItem(`ila:manual-journal:daily-review:a1:${today}`)).toBe('dismissed')
    view.unmount()
    vi.mocked(journalApi.review).mockResolvedValue({ ...ready, revision: 'r2' })
    const next = mount()
    await screen.findByText('Sumber: aturan')
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
    next.unmount()
    vi.mocked(journalApi.review).mockResolvedValue({ ...ready, account_id: 'a2' })
    mount(<JournalReview accountId="a2" />)
    expect(await screen.findByRole('complementary')).toBeInTheDocument()
  })
  it('lets users disable global and account AI independently and explains effective fallback', async () => {
    mount()
    const accountToggle = await screen.findByLabelText('Izinkan review AI untuk akun ini')
    await waitFor(() => expect(accountToggle).toBeEnabled())
    fireEvent.click(accountToggle)
    await waitFor(() => expect(journalApi.saveReviewSettings).toHaveBeenCalledWith(false, 'a1'))
    await waitFor(() =>
      expect(screen.getByLabelText('Izinkan review AI untuk seluruh jurnal saya')).toBeEnabled(),
    )
    fireEvent.click(screen.getByLabelText('Izinkan review AI untuk seluruh jurnal saya'))
    await waitFor(() =>
      expect(journalApi.saveReviewSettings).toHaveBeenCalledWith(false, undefined),
    )
    expect(screen.getByText(/AI efektif tidak aktif/)).toBeInTheDocument()
    expect(screen.getByText(/isi jurnal mentah tidak dikirim/)).toBeInTheDocument()
  })
  it('does not allow preference writes when settings failed to load', async () => {
    vi.mocked(journalApi.reviewSettings).mockRejectedValue(new Error('Pengaturan tidak tersedia'))
    mount()
    expect(await screen.findByText('Pengaturan tidak tersedia')).toBeInTheDocument()
    expect(screen.getByLabelText('Izinkan review AI untuk akun ini')).toBeDisabled()
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
