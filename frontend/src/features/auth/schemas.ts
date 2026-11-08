import { z } from 'zod'

/**
 * Skema validasi mengikuti aturan backend agar pengguna mendapat pesan
 * sebelum permintaan dikirim, bukan sesudah.
 */
export const loginSchema = z.object({
  email: z.string().min(1, 'Email wajib diisi').email('Format email tidak sah'),
  password: z.string().min(1, 'Password wajib diisi'),
})

export const registerSchema = z.object({
  name: z.string().min(3, 'Nama minimal 3 karakter').max(100, 'Nama maksimal 100 karakter'),
  email: z.string().min(1, 'Email wajib diisi').email('Format email tidak sah'),
  password: z
    .string()
    .min(10, 'Password minimal 10 karakter')
    .max(72, 'Password maksimal 72 karakter'),
  // Backend mewajibkan persetujuan Syarat Layanan, Kebijakan Privasi, dan
  // Kebijakan Penggunaan Wajar sebelum akun dibuat.
  acceptTerms: z.boolean().refine((value) => value, {
    message: 'Anda harus menyetujui syarat dan kebijakan',
  }),
})

export type LoginFormValues = z.infer<typeof loginSchema>
export type RegisterFormValues = z.infer<typeof registerSchema>
