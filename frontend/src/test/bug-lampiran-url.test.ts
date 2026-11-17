import { describe, expect, it } from 'vitest'
import { bugApi } from '@/lib/bug-api'

describe('Tautan lampiran laporan bug', () => {
  // Regresi: tautan ini dipakai langsung di <a href> dan tidak melewati
  // buildUrl, jadi tanpa prefix /api/v1 browser membuka halaman aplikasi dan
  // pengguna melihat 404 saat menekan lampiran.
  it('memakai prefix /api/v1 yang sama dengan panggilan API lain', () => {
    const url = bugApi.attachmentUrl('rep-1', 'att-2')
    expect(url).toBe('/api/v1/bug-reports/rep-1/attachments/att-2')
  })
})
