import { describe, expect, it } from 'vitest'
import { loginSchema, registerSchema } from '@/features/auth/schemas'

describe('loginSchema', () => {
  it('menerima email dan password yang sah', () => {
    const result = loginSchema.safeParse({ identifier: 'budi@example.com', password: 'apa-saja' })
    expect(result.success).toBe(true)
  })

  // Satu kolom masuk harus menerima username juga, bukan hanya email.
  it('menerima username pada kolom yang sama', () => {
    const result = loginSchema.safeParse({ identifier: 'budi_santoso', password: 'apa-saja' })
    expect(result.success).toBe(true)
  })

  it('menolak username yang memuat spasi', () => {
    const result = loginSchema.safeParse({ identifier: 'budi santoso', password: 'apa-saja' })
    expect(result.success).toBe(false)
  })

  it('menolak password kosong', () => {
    const result = loginSchema.safeParse({ identifier: 'budi@example.com', password: '' })
    expect(result.success).toBe(false)
  })
})

/** Data dasar yang sah; test di bawah mengubah satu bagian saja. */
const dataSah = {
  name: 'Budi Santoso',
  username: 'budi.santoso',
  email: 'budi@example.com',
  phone: '',
  password: 'rahasia-panjang-123',
  acceptTerms: true,
}

describe('registerSchema', () => {
  it('menolak password kurang dari 10 karakter', () => {
    const result = registerSchema.safeParse({ ...dataSah, password: 'pendek' })
    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues[0]?.message).toContain('minimal 10')
    }
  })

  it('menolak nama kurang dari 3 karakter', () => {
    const result = registerSchema.safeParse({ ...dataSah, name: 'Bu' })
    expect(result.success).toBe(false)
  })

  it('menerima data yang sah', () => {
    const result = registerSchema.safeParse(dataSah)
    expect(result.success).toBe(true)
  })

  // Username adalah identitas masuk, jadi bentuknya dibatasi sejak awal.
  it('menolak username dengan spasi atau simbol bebas', () => {
    for (const username of ['budi santoso', 'budi!', 'bu']) {
      const result = registerSchema.safeParse({ ...dataSah, username })
      expect(result.success, `username ${username} seharusnya ditolak`).toBe(false)
    }
  })

  it('menerima nomor telepon kosong', () => {
    const result = registerSchema.safeParse({ ...dataSah, phone: '' })
    expect(result.success).toBe(true)
  })

  it('menolak nomor telepon yang jelas bukan nomor', () => {
    const result = registerSchema.safeParse({ ...dataSah, phone: 'bukan nomor' })
    expect(result.success).toBe(false)
  })

  // Backend menolak pendaftaran tanpa persetujuan, jadi formulir harus
  // menolaknya lebih dulu agar pengguna tidak melihat galat server.
  it('menolak pendaftaran tanpa persetujuan', () => {
    const result = registerSchema.safeParse({ ...dataSah, acceptTerms: false })
    expect(result.success).toBe(false)
  })
})
