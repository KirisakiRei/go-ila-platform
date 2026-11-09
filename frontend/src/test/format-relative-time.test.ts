import { describe, expect, it } from 'vitest'
import { formatRelativeTime } from '@/lib/format'

/**
 * Jarak waktu untuk kartu halaman publik.
 *
 * Yang dijaga: batas antar satuan konsisten (detik → menit → jam → hari), waktu
 * yang belum lewat tidak pernah tampil sebagai tanggal masa depan, dan nilai
 * kosong/tidak sah menjadi tanda pisah, bukan "Invalid Date".
 */
describe('formatRelativeTime', () => {
  const acuan = new Date('2026-11-09T10:00:00Z')
  const lalu = (detik: number) => new Date(acuan.getTime() - detik * 1000).toISOString()

  it('menulis "baru saja" untuk waktu di bawah satu menit', () => {
    expect(formatRelativeTime(lalu(0), acuan)).toBe('baru saja')
    expect(formatRelativeTime(lalu(59), acuan)).toBe('baru saja')
  })

  it('berpindah ke menit tepat pada 60 detik', () => {
    expect(formatRelativeTime(lalu(60), acuan)).toBe('1 menit lalu')
    expect(formatRelativeTime(lalu(59 * 60), acuan)).toBe('59 menit lalu')
  })

  it('berpindah ke jam tepat pada 60 menit', () => {
    expect(formatRelativeTime(lalu(60 * 60), acuan)).toBe('1 jam lalu')
    expect(formatRelativeTime(lalu(23 * 3600), acuan)).toBe('23 jam lalu')
  })

  it('berpindah ke hari tepat pada 24 jam', () => {
    expect(formatRelativeTime(lalu(24 * 3600), acuan)).toBe('1 hari lalu')
    expect(formatRelativeTime(lalu(6 * 24 * 3600), acuan)).toBe('6 hari lalu')
  })

  it('memakai bulan dan tahun untuk rentang panjang', () => {
    expect(formatRelativeTime(lalu(30 * 24 * 3600), acuan)).toBe('1 bulan lalu')
    expect(formatRelativeTime(lalu(400 * 24 * 3600), acuan)).toBe('1 tahun lalu')
  })

  // Jam server yang sedikit di depan tidak boleh menghasilkan "−1 menit lalu".
  it('memperlakukan waktu di masa depan sebagai "baru saja"', () => {
    expect(formatRelativeTime(lalu(-120), acuan)).toBe('baru saja')
  })

  it('mengembalikan tanda pisah untuk nilai kosong atau tidak sah', () => {
    expect(formatRelativeTime(null, acuan)).toBe('—')
    expect(formatRelativeTime(undefined, acuan)).toBe('—')
    expect(formatRelativeTime('bukan tanggal', acuan)).toBe('—')
  })
})
