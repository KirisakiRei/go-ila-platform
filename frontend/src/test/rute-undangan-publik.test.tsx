import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PublicApp } from '@/app/PublicApp'

/**
 * Uji ini mengunci satu cacat nyata: tautan undangan yang dibuat admin berbentuk
 * `/daftar?invite=KODE`, tetapi entri publik tidak punya rute itu sehingga
 * pengunjung menerima "Halaman tidak ditemukan" padahal kodenya sah.
 *
 * Berbeda dari `undangan.test.tsx` yang memasang `<Route path="/daftar">` sendiri,
 * di sini yang diuji adalah tabel rute SUNGGUHAN milik aplikasi. Daftar rute
 * buatan tidak bisa membuktikan rute terpasang.
 */

/** Jawaban fetch tiruan: status publik DAN pemeriksaan undangan. */
function jawabFetch(valid: boolean) {
  return vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input)

    if (url.includes('/status')) {
      return new Response(
        JSON.stringify({
          success: true,
          data: {
            mode: 'normal',
            message: '',
            server_time: '2026-11-08T12:00:00Z',
            time_pair: { wib: '19:00 WIB', ny: '07:00 NY', utc: '12:00 UTC' },
            market: {
              open: true,
              label: 'Buka',
              next_change: '2026-11-08T22:00:00Z',
              next_label: 'Tutup',
              next_change_pair: { wib: '05:00 WIB', ny: '17:00 NY', utc: '22:00 UTC' },
            },
            invite_only: true,
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      )
    }

    // Pemeriksaan kode undangan.
    if (url.includes('/invites/')) {
      if (!valid) {
        return new Response(
          JSON.stringify({ success: false, error: { message: 'Undangan tidak berlaku.' } }),
          { status: 403, headers: { 'Content-Type': 'application/json' } },
        )
      }
      return new Response(
        JSON.stringify({
          success: true,
          data: {
            valid: true,
            tier: 'plus',
            expires_at: '2026-11-20T12:00:00Z',
            sisa_kuota: 3,
            email_terkunci: false,
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      )
    }

    return new Response(JSON.stringify({ success: true, data: null }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  })
}

/** Membuka aplikasi publik pada satu alamat, sebagaimana pengunjung membukanya. */
function bukaAlamat(alamat: string) {
  window.history.pushState({}, '', alamat)
  return render(<PublicApp />)
}

describe('Alamat tautan undangan dilayani entri publik', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    window.history.pushState({}, '', '/')
  })

  it('/daftar?invite=KODE menampilkan formulir, bukan halaman tidak ditemukan', async () => {
    vi.stubGlobal('fetch', jawabFetch(true))

    bukaAlamat('/daftar?invite=KODE-UJI-123')

    // Inilah yang tadinya gagal: pengunjung menerima "Halaman tidak ditemukan".
    expect(await screen.findByRole('heading', { name: 'Daftar' })).toBeInTheDocument()
    expect(screen.queryByText('Halaman tidak ditemukan')).not.toBeInTheDocument()
  })

  it('/daftar?invite=KODE yang tidak sah menolak dengan pesan undangan', async () => {
    vi.stubGlobal('fetch', jawabFetch(false))

    bukaAlamat('/daftar?invite=KODE-PALSU')

    expect(await screen.findByText('Undangan tidak berlaku')).toBeInTheDocument()
    expect(screen.queryByText('Halaman tidak ditemukan')).not.toBeInTheDocument()
  })

  it('/daftar tanpa kode undangan menjelaskan akses lewat undangan', async () => {
    vi.stubGlobal('fetch', jawabFetch(true))

    bukaAlamat('/daftar')

    expect(await screen.findByText('Akses lewat undangan')).toBeInTheDocument()
  })
})
