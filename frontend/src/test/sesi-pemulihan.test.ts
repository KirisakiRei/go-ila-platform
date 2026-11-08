import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { onUnauthorized, request, selamaPemulihan } from '@/lib/http'

/**
 * Pengunjung yang belum pernah masuk menerima 401 dari /me saat halaman
 * dimuat. Itu bukan sesi berakhir, jadi handler unauthorized tidak boleh
 * dipanggil dan halaman masuk tidak boleh menampilkan "Sesi Anda berakhir".
 */

const fetchAsli = globalThis.fetch

/** balas401 membuat fetch palsu yang selalu membalas status 401. */
function balas401() {
  globalThis.fetch = vi.fn().mockImplementation(() =>
    Promise.resolve(
      new Response(JSON.stringify({ success: false, message: 'unauthorized access' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    ),
  ) as unknown as typeof fetch
}

describe('selamaPemulihan', () => {
  beforeEach(() => {
    balas401()
  })

  afterEach(() => {
    globalThis.fetch = fetchAsli
    onUnauthorized(null)
  })

  it('menahan laporan 401 selama pemulihan sesi', async () => {
    const dipanggil = vi.fn()
    onUnauthorized(dipanggil)

    await expect(selamaPemulihan(() => request('/me'))).rejects.toThrow()

    expect(dipanggil, '401 saat pemulihan tidak boleh dianggap sesi habis').not.toHaveBeenCalled()
  })

  it('kembali melaporkan 401 setelah pemulihan selesai', async () => {
    const dipanggil = vi.fn()
    onUnauthorized(dipanggil)

    await selamaPemulihan(() => request('/me')).catch(() => undefined)
    await request('/me').catch(() => undefined)

    expect(dipanggil).toHaveBeenCalledTimes(1)
  })

  it('penanda tetap aktif walau permintaan gagal', async () => {
    const dipanggil = vi.fn()
    onUnauthorized(dipanggil)

    await selamaPemulihan(async () => {
      await request('/me').catch(() => undefined)
      // Permintaan kedua di dalam penanda yang sama juga tidak boleh lapor.
      await request('/me').catch(() => undefined)
    })

    expect(dipanggil).not.toHaveBeenCalled()

    // Setelah blok selesai, laporan kembali normal.
    await request('/me').catch(() => undefined)
    expect(dipanggil).toHaveBeenCalledTimes(1)
  })
})
