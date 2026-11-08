import { request } from './http'
import type { LoginPayload, LoginResult, OnboardingState, RegisterPayload, SubscriptionBanner, User } from '@/types/api'

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

  /** Server selalu membalas sukses agar alamat email tidak bisa dipetakan. */
  forgotPassword(email: string) {
    return request<void>('/auth/forgot-password', { method: 'POST', body: { email } })
  },

  /** Reset menukar token sekali pakai; seluruh perangkat ikut keluar. */
  resetPassword(token: string, newPassword: string) {
    return request<void>('/auth/reset-password', {
      method: 'POST',
      body: { token, new_password: newPassword },
    })
  },

  verifyEmail(token: string) {
    return request<void>('/auth/verify-email', { method: 'POST', body: { token } })
  },

  resendVerification() {
    return request<void>('/me/verify-email/resend', { method: 'POST', body: {} })
  },
}

/** Alur akun: onboarding dan banner langganan. */
export const accountFlowApi = {
  onboarding: () => request<OnboardingState>('/me/onboarding'),
  saveOnboarding: (step: number, done = false) =>
    request<OnboardingState>('/me/onboarding', { method: 'PUT', body: { step, done } }),
  banner: () => request<SubscriptionBanner>('/me/subscription-banner'),
}
