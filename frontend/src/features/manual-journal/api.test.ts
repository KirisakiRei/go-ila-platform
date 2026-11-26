import { beforeEach, describe, expect, it, vi } from 'vitest'
import { request, downloadFile } from '@/lib/http'
import { journalApi } from './api'
import { emptyTrade } from './model'
import type { JournalFilters } from './types'

vi.mock('@/lib/http', () => ({
  request: vi.fn(),
  downloadFile: vi.fn(),
  readCsrfToken: () => 'csrf',
  CSRF_HEADER: 'X-CSRF-Token',
}))
const filters: JournalFilters = {
  account_id: 'a1',
  from: '2026-11-01',
  to: '2026-11-30',
  status: '',
  asset: '',
  setup: '',
  tag: '',
  emotion: 'Tenang',
  mistake: 'Late entry',
}
beforeEach(() => vi.resetAllMocks())

describe('manual journal API contract', () => {
  it('loads bounded pages, preserves filters and normalizes nullable lists', async () => {
    const row = { ...emptyTrade('a1'), id: 't1', tags: null }
    vi.mocked(request)
      .mockResolvedValueOnce(Array.from({ length: 100 }, () => row))
      .mockResolvedValueOnce([row])
    const result = await journalApi.entries(filters)
    expect(result.data).toHaveLength(101)
    expect(result.data[0]?.tags).toEqual([])
    expect(request).toHaveBeenNthCalledWith(2, '/manual-journal/entries', {
      query: { ...filters, page: 2, per_page: 100 },
      signal: undefined,
    })
  })
  it('uses full PUT payload with decimal strings rather than inferred pnl', async () => {
    const input = { ...emptyTrade('a1'), entry_price: '2600.12345678', net_pnl: '0.0000' }
    vi.mocked(request).mockResolvedValue({ ...input, id: 't1' })
    await journalApi.saveEntry(input, 't1')
    expect(request).toHaveBeenCalledWith('/manual-journal/entries/t1', {
      method: 'PUT',
      body: input,
    })
    expect(input.entry_price).toBe('2600.12345678')
  })
  it('previews and confirms identical CSV and exports identical filters', async () => {
    vi.mocked(request).mockResolvedValue({ rows: [], count: 0 })
    await journalApi.preview('csv text')
    await journalApi.import({ csv: 'csv text', token: 'preview-token' })
    await journalApi.export(filters)
    expect(request).toHaveBeenCalledWith('/manual-journal/import/preview', {
      method: 'POST',
      body: { csv: 'csv text' },
    })
    expect(request).toHaveBeenCalledWith('/manual-journal/import/confirm', {
      method: 'POST',
      body: { csv: 'csv text', token: 'preview-token' },
    })
    expect(downloadFile).toHaveBeenCalledWith('/manual-journal/export', filters)
  })
  it('rejects invalid screenshots before uploading', async () => {
    await expect(
      journalApi.uploadScreenshot('t1', new File(['bad'], 'bad.svg', { type: 'image/svg+xml' })),
    ).rejects.toThrow(/PNG/)
  })
})
