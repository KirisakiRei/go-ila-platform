import { request } from './http'
import type { LoginPayload, LoginResult, RegisterPayload, User } from '@/types/api'

/**
 * Semua panggilan autentikasi terkumpul di satu modul.
 * Token sesi tidak pernah menyentuh JavaScript; backend menaruhnya di
 * cookie HttpOnly sehingga hanya respons yang membawa data akun.
 */
export const authApi = {
  login(payload: LoginPayload) {
    return request<LoginResult>('/auth/login', { method: 'POST', body: payload })
  },

  register(payload: RegisterPayload) {
    return request<LoginResult>('/auth/register', { method: 'POST', body: payload })
  },

  me() {
    return request<User>('/me')
  },

  logout() {
    return request<{ logged_out: boolean }>('/auth/logout', { method: 'POST' })
  },

  /** Memastikan cookie CSRF tersedia setelah halaman dimuat ulang. */
  csrf() {
    return request<{ csrf_token: string }>('/auth/csrf')
  },
}
