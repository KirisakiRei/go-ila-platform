import { describe, expect, it, vi, afterEach } from 'vitest'
import { ApiError } from '@/lib/api-error'
import { CSRF_HEADER, readCsrfToken, request } from '@/lib/http'

function mockFetch(status: number, body: unknown) {
  const fn = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

function setCookie(value: string) {
  document.cookie = value
}

describe('request', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    // Bersihkan cookie antar-tes dengan mengosongkan nilai dan masa berlaku.
    for (const name of ['ila_csrf', 'ila_session']) {
      document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`
    }
  })

  it('mengembalikan data saat balasan sukses', async () => {
    mockFetch(200, { success: true, message: 'ok', data: { id: '1' } })

    const result = await request<{ id: string }>('/ping')
    expect(result).toEqual({ id: '1' })
  })

  it('tidak pernah mengirim header Authorization', async () => {
    const fetchMock = mockFetch(200, { success: true, message: 'ok', data: null })

    await request('/ping')

    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers
    expect(headers.get('Authorization')).toBeNull()
    // Sesi dikirim lewat cookie yang tidak terjangkau JavaScript.
    expect(fetchMock.mock.calls[0]?.[1]?.credentials).toBe('same-origin')
  })

  it('menyertakan header CSRF pada permintaan tulis', async () => {
    setCookie('ila_csrf=abc123')
    const fetchMock = mockFetch(200, { success: true, message: 'ok', data: null })

    await request('/trial', { method: 'POST' })

    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers
    expect(headers.get(CSRF_HEADER)).toBe('abc123')
  })

  it('tidak menyertakan header CSRF pada permintaan baca', async () => {
    setCookie('ila_csrf=abc123')
    const fetchMock = mockFetch(200, { success: true, message: 'ok', data: null })

    await request('/me')

    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Headers
    expect(headers.get(CSRF_HEADER)).toBeNull()
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

describe('readCsrfToken', () => {
  afterEach(() => {
    document.cookie = 'ila_csrf=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
    document.cookie = 'ila_admin_csrf=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
  })

  it('mengembalikan string kosong bila cookie tidak ada', () => {
    expect(readCsrfToken()).toBe('')
  })

  it('membaca nilai cookie di antara cookie lain', () => {
    document.cookie = 'lain=x'
    document.cookie = 'ila_csrf=token-uji'

    expect(readCsrfToken()).toBe('token-uji')
  })

  it('membaca nilai cookie admin untuk path admin', () => {
    document.cookie = 'ila_csrf=token-user'
    document.cookie = 'ila_admin_csrf=token-admin'

    expect(readCsrfToken('/admin/users')).toBe('token-admin')
    expect(readCsrfToken('/auth/me')).toBe('token-user')
  })
})
