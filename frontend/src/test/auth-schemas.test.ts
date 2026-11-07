import { describe, expect, it } from 'vitest'
import { loginSchema, registerSchema } from '@/features/auth/schemas'

describe('loginSchema', () => {
  it('menerima email dan password yang sah', () => {
    const result = loginSchema.safeParse({ email: 'budi@example.com', password: 'apa-saja' })
    expect(result.success).toBe(true)
  })

  it('menolak email dengan format salah', () => {
    const result = loginSchema.safeParse({ email: 'bukan-email', password: 'apa-saja' })
    expect(result.success).toBe(false)
  })

  it('menolak password kosong', () => {
    const result = loginSchema.safeParse({ email: 'budi@example.com', password: '' })
    expect(result.success).toBe(false)
  })
})

describe('registerSchema', () => {
  it('menolak password kurang dari 10 karakter', () => {
    const result = registerSchema.safeParse({
      name: 'Budi',
      email: 'budi@example.com',
      password: 'pendek',
    })
    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues[0]?.message).toContain('minimal 10')
    }
  })

  it('menolak nama kurang dari 3 karakter', () => {
    const result = registerSchema.safeParse({
      name: 'Bu',
      email: 'budi@example.com',
      password: 'rahasia-panjang-123',
    })
    expect(result.success).toBe(false)
  })

  it('menerima data yang sah', () => {
    const result = registerSchema.safeParse({
      name: 'Budi Santoso',
      email: 'budi@example.com',
      password: 'rahasia-panjang-123',
    })
    expect(result.success).toBe(true)
  })
})
