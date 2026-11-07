import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { ApiError } from '@/lib/api-error'
import { request, tokenStorage } from '@/lib/http'

function mockFetch(status: number, body: unknown) {
  const fn = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

describe('request', () => {
  beforeEach(() => {
    tokenStorage.clear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('mengembalikan data saat balasan sukses', async () => {
    mockFetch(200, { success: true, message: 'ok', data: { id: '1' } })

    const result = await request<{ id: string }>('/ping')
    expect(result).toEqual({ id: '1' })
  })

  it('menyertakan token bila tersedia', async () => {
    tokenStorage.set('token-uji')
    const fetchMock = mockFetch(200, { success: true, message: 'ok', data: null })

    await request('/ping')

    const init = fetchMock.mock.calls[0]?.[1]
    const headers = init?.headers as Headers
    expect(headers.get('Authorization')).toBe('Bearer token-uji')
  })

  it('melempar ApiError saat balasan gagal', async () => {
    mockFetch(401, {
      success: false,
      message: 'Sesi tidak sah',
      error: 'UNAUTHORIZED',
    })

    await expect(request('/me')).rejects.toBeInstanceOf(ApiError)
  })

  it('menandai kegagalan jaringan sebagai ApiError dengan status 0', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('gagal konek')))

    const error = await request('/ping').catch((e: unknown) => e)

    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).isNetworkProblem).toBe(true)
  })
})
