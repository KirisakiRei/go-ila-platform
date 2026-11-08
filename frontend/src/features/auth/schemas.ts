import { z } from 'zod'

/**
 * Skema validasi mengikuti aturan backend agar pengguna mendapat pesan
 * sebelum permintaan dikirim, bukan sesudah.
 */

/**
 * Satu kolom untuk email atau username. Pembedanya sederhana: yang memuat
 * `@` dianggap email, sisanya username. Backend mencocokkan keduanya.
 */
const identifierField = z
  .string()
  .trim()
  .min(3, 'Minimal 3 karakter')
  .max(150, 'Maksimal 150 karakter')
  .refine((value) => !value.includes(' ') || value.includes('@'), {
    message: 'Username tidak boleh memuat spasi',
  })

export const loginSchema = z.object({
  identifier: identifierField,
  password: z.string().min(1, 'Password wajib diisi'),
})

/** Username dipakai untuk masuk, jadi bentuknya dibatasi dan mudah diingat. */
const usernameField = z
  .string()
  .trim()
  .min(3, 'Username minimal 3 karakter')
  .max(32, 'Username maksimal 32 karakter')
  .regex(/^[a-zA-Z0-9._-]+$/, 'Hanya huruf, angka, titik, garis bawah, dan strip')

/**
 * Nomor telepon opsional. Yang diperiksa hanya bentuk kasar karena nomor
 * tidak dipakai untuk verifikasi; backend menyisakan angka dan tanda +.
 */
const phoneField = z
  .string()
  .trim()
  .refine((value) => value === '' || /^\+?[0-9\s-]{7,20}$/.test(value), {
    message: 'Nomor telepon tidak sah',
  })

export const registerSchema = z.object({
  name: z.string().trim().min(3, 'Nama minimal 3 karakter').max(100, 'Nama maksimal 100 karakter'),
  username: usernameField,
  email: z.string().trim().min(1, 'Email wajib diisi').email('Format email tidak sah'),
  phone: phoneField,
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

/** Langkah pendaftaran dipisah menjadi 2 tahap agar ringkas dan cepat. */
export const REGISTER_STEPS = ['Akun', 'Keamanan'] as const
export type RegisterStep = (typeof REGISTER_STEPS)[number]

/**
 * Field yang diperiksa pada tiap langkah. Pengguna tidak bisa lanjut
 * sebelum bagiannya sah, tetapi validasi akhir tetap dilakukan saat kirim.
 */
export const REGISTER_STEP_FIELDS: Record<RegisterStep, (keyof RegisterFormValues)[]> = {
  Akun: ['name', 'username', 'email', 'phone'],
  Keamanan: ['password', 'acceptTerms'],
}
