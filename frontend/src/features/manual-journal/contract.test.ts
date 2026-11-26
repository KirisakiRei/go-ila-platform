import { afterEach, describe, expect, it, vi } from 'vitest'
import { journalApi } from './api'
import { onUnauthorized } from '@/lib/http'
import { emptyTrade } from './model'

// Exercise the real shared HTTP transport against backend-shaped envelopes (not mocked API methods).
const account = {
  id: 'a1',
  user_id: 'owner',
  name: 'IDR manual',
  broker: '',
  currency: 'IDR',
  initial_capital: '1000.0000',
  initial_date: '2026-11-01',
  current_realized_balance: '1234.5678',
  balance_complete: true,
  unknown_closed_pnl: 0,
  notes: '',
  archived: true,
  checklist: null,
  created_at: '2026-11-01T00:00:00Z',
}
const response = (data: unknown, status = 200, meta?: unknown) =>
  new Response(
    JSON.stringify({
      success: status < 400,
      message: status < 400 ? 'Manual journal' : 'account is archived',
      data,
      meta,
    }),
    { status, headers: { 'Content-Type': 'application/json' } },
  )
afterEach(() => {
  onUnauthorized(null)
  vi.unstubAllGlobals()
  document.cookie = 'ila_csrf=; Max-Age=0; path=/'
})

describe('manual journal backend envelopes through real HTTP client', () => {
  it('preserves additive account balance strings and explicit incomplete null balances', async () => {
    const incomplete = {
      ...account,
      id: 'a2',
      current_realized_balance: null,
      balance_complete: false,
      unknown_closed_pnl: 2,
    }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response([account, incomplete])))
    const result = await journalApi.accounts()
    expect(result.data[0]).toMatchObject({
      current_realized_balance: '1234.5678',
      balance_complete: true,
      unknown_closed_pnl: 0,
    })
    expect(result.data[1]).toMatchObject({
      current_realized_balance: null,
      balance_complete: false,
      unknown_closed_pnl: 2,
    })
  })
  it('encodes exact emotion and mistake query names and preserves labels containing commas', async () => {
    const fetcher = vi.fn().mockImplementation(async () => response([]))
    vi.stubGlobal('fetch', fetcher)
    const filters = {
      account_id: 'a1',
      from: '',
      to: '',
      status: '' as const,
      asset: '',
      setup: '',
      tag: 'London, NY',
      emotion: 'Calm, focused',
      mistake: 'Late, rushed',
    }
    await journalApi.entries(filters)
    await journalApi.stats(filters)
    await journalApi.insights(filters)
    for (const call of fetcher.mock.calls) {
      const url = new URL(call[0], 'https://ila.test')
      expect(url.searchParams.get('emotion')).toBe('Calm, focused')
      expect(url.searchParams.get('mistake')).toBe('Late, rushed')
      expect(url.searchParams.get('tag')).toBe('London, NY')
      expect(url.searchParams.has('emotions')).toBe(false)
      expect(url.searchParams.has('mistakes')).toBe(false)
    }
  })
  it('reads actual review fields and global/account settings; writes only ai_enabled', async () => {
    const review = {
      account_id: 'a1',
      date: '2026-11-26',
      revision: 'hash',
      status: 'running',
      text: '',
      source: '',
      updated_at: '2026-11-26T01:00:00Z',
    }
    const global = { account_id: '', ai_enabled: true, effective_ai_enabled: false }
    const scoped = { account_id: 'a1', ai_enabled: false, effective_ai_enabled: false }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(response(review))
      .mockResolvedValueOnce(response(global))
      .mockResolvedValueOnce(response(scoped))
      .mockResolvedValueOnce(response(scoped))
      .mockResolvedValueOnce(response({ ...global, ai_enabled: false }))
    vi.stubGlobal('fetch', fetcher)
    expect(await journalApi.review('a1')).toEqual(review)
    expect(await journalApi.reviewSettings()).toEqual(global)
    expect(await journalApi.reviewSettings('a1')).toEqual(scoped)
    await journalApi.saveReviewSettings(false, 'a1')
    await journalApi.saveReviewSettings(false)
    expect(fetcher.mock.calls.map((call) => call[0])).toEqual([
      '/api/v1/manual-journal/review?account_id=a1',
      '/api/v1/manual-journal/review/settings',
      '/api/v1/manual-journal/review/settings?account_id=a1',
      '/api/v1/manual-journal/review/settings?account_id=a1',
      '/api/v1/manual-journal/review/settings',
    ])
    expect(fetcher.mock.calls[0]?.[1].body).toBeUndefined()
    expect(JSON.parse(fetcher.mock.calls[3]?.[1].body)).toEqual({ ai_enabled: false })
    expect(JSON.parse(fetcher.mock.calls[4]?.[1].body)).toEqual({ ai_enabled: false })
  })
  it('reads account checklist null, archives with DELETE and restores using metadata-free PUT', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(
        response([account], 200, {
          current_page: 1,
          per_page: 100,
          total_items: 1,
          total_pages: 1,
        }),
      )
      .mockResolvedValueOnce(response(null))
      .mockResolvedValueOnce(response({ ...account, archived: false, checklist: [] }))
    vi.stubGlobal('fetch', fetcher)
    document.cookie = 'ila_csrf=contract-csrf; path=/'
    const accounts = await journalApi.accounts()
    expect(accounts.data[0]?.checklist).toEqual([])
    await journalApi.archiveAccount('a1')
    await journalApi.saveAccount({ ...account, checklist: [], archived: false }, 'a1')
    expect(fetcher.mock.calls[1]?.[0]).toBe('/api/v1/manual-journal/accounts/a1')
    expect(fetcher.mock.calls[1]?.[1].method).toBe('DELETE')
    const options = fetcher.mock.calls[2]?.[1]
    expect(JSON.parse(options.body)).toEqual({
      name: account.name,
      broker: '',
      currency: 'IDR',
      initial_capital: '1000.0000',
      initial_date: account.initial_date,
      notes: '',
      archived: false,
      checklist: [],
    })
    expect(options.headers.get('X-CSRF-Token')).toBe('contract-csrf')
  })
  it('preserves entry snapshot and decimal strings without leaking response metadata into PUT', async () => {
    const input = {
      ...emptyTrade('a1'),
      checklist: [{ label: 'Old template', checked: true }],
      entry_price: '123.12345678',
    }
    const row = {
      ...input,
      id: 't1',
      user_id: 'owner',
      created_at: '2026-11-01T00:00:00Z',
      updated_at: '2026-11-02T00:00:00Z',
    }
    const fetcher = vi.fn().mockResolvedValue(response(row))
    vi.stubGlobal('fetch', fetcher)
    const saved = await journalApi.saveEntry(row, 't1')
    expect(saved.checklist).toEqual(input.checklist)
    expect(JSON.parse(fetcher.mock.calls[0]?.[1].body)).toEqual(input)
  })
  it('passes persistent preview token with exact CSV and can safely retry the same confirm request', async () => {
    const csv = 'account_id,status\na1,draft'
    const preview = {
      rows: [emptyTrade('a1')],
      count: 1,
      token: 'token-1',
      expires_at: '2099-01-01T00:00:00Z',
    }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(response(preview))
      .mockResolvedValueOnce(response({ count: 1 }))
      .mockResolvedValueOnce(response({ count: 1 }))
    vi.stubGlobal('fetch', fetcher)
    expect(await journalApi.preview(csv)).toEqual(preview)
    await journalApi.import({ csv, token: preview.token })
    await journalApi.import({ csv, token: preview.token })
    expect(fetcher.mock.calls[1]?.[1].body).toBe(fetcher.mock.calls[2]?.[1].body)
    expect(JSON.parse(fetcher.mock.calls[1]?.[1].body)).toEqual({ csv, token: 'token-1' })
  })
  it('reads private screenshot metadata and uploads multipart with CSRF, never public URLs', async () => {
    const shot = {
      id: 's1',
      entry_id: 't1',
      file_name: 'sebelum.png',
      mime_type: 'image/png',
      size_bytes: 3,
    }
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(response([shot]))
      .mockResolvedValueOnce(response(shot, 201))
    vi.stubGlobal('fetch', fetcher)
    document.cookie = 'ila_csrf=upload-csrf; path=/'
    expect(await journalApi.screenshots('t1')).toEqual([shot])
    expect(
      await journalApi.uploadScreenshot(
        't1',
        new File(['png'], 'sebelum.png', { type: 'image/png' }),
      ),
    ).toEqual(shot)
    const options = fetcher.mock.calls[1]?.[1]
    expect(options.body).toBeInstanceOf(FormData)
    expect(options.body.get('file').name).toBe('sebelum.png')
    expect(options.headers.get('X-CSRF-Token')).toBe('upload-csrf')
    expect(options.headers.has('Content-Type')).toBe(false)
    expect(options.credentials).toBe('same-origin')
    expect(options.signal).toBeInstanceOf(AbortSignal)
    expect(journalApi.screenshotUrl('t1', 's1')).toBe(
      '/api/v1/manual-journal/entries/t1/screenshots/s1',
    )
  })
  it('notifies the shared unauthorized handler when screenshot upload returns 401', async () => {
    const unauthorized = vi.fn()
    onUnauthorized(unauthorized)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(null, 401)))
    await expect(
      journalApi.uploadScreenshot('t1', new File(['png'], 'shot.png', { type: 'image/png' })),
    ).rejects.toMatchObject({ status: 401 })
    expect(unauthorized).toHaveBeenCalledOnce()
  })
  it('surfaces archive/mixed currency backend validation instead of fabricating success', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(null, 409)))
    await expect(journalApi.deleteEntry('t1')).rejects.toThrow('account is archived')
  })
})
