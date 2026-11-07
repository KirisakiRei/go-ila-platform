import { request } from './http'
import type { LoginPayload, LoginResult, RegisterPayload, User } from '@/types/api'

/** Semua panggilan autentikasi terkumpul di satu modul. */
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
}
