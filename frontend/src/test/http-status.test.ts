import { describe, expect, it, vi, afterEach } from 'vitest'
import { onUnauthorized, request } from '@/lib/http'

function mockFetch(status: number, body: unknown) {
  const fn = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

describe('pembedaan status sesi', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  // 401 berarti sesi tidak sah: penangan global harus dipanggil.
  it('memanggil handler saat 401', async () => {
    const handler = vi.fn()
    onUnauthorized(handler)
    mockFetch(401, { success: false, message: 'unauthorized', error: 'UNAUTHORIZED' })

    await expect(request('/me')).rejects.toThrow()
    expect(handler).toHaveBeenCalledTimes(1)
  })

  // 403 berarti sesi sah tetapi tidak berwenang. Melakukan logout di sini
  // membuat pengguna keluar hanya karena membuka menu yang bukan haknya.
  it('tidak memanggil handler saat 403', async () => {
    const handler = vi.fn()
    onUnauthorized(handler)
    mockFetch(403, { success: false, message: 'forbidden', error: 'FORBIDDEN' })

    await expect(request('/admin/users')).rejects.toThrow(
      'Anda tidak punya hak akses untuk tindakan ini.',
    )
    expect(handler).not.toHaveBeenCalled()
  })

  // 429 harus menjelaskan bahwa pengguna perlu menunggu.
  it('memberi pesan jelas saat 429', async () => {
    mockFetch(429, { success: false, message: 'too many', error: 'TOO_MANY_REQUESTS' })

    await expect(request('/auth/login', { method: 'POST', body: {} })).rejects.toThrow(
      'Terlalu banyak percobaan',
    )
  })

  // 423 dipakai saat pemeliharaan: pengguna tidak diarahkan keluar.
  it('memberi pesan pemeliharaan saat 423', async () => {
    mockFetch(423, { success: false, message: 'maintenance', error: 'LOCKED' })

    await expect(request('/me')).rejects.toThrow('pemeliharaan')
  })
})
