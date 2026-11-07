/** Batas nilai yang mengikuti kontrak API backend. */

export type Tier = 'free' | 'basic' | 'plus'
export type Role = 'user' | 'admin' | 'support'

/** Bentuk balasan seragam dari backend (pkg/httpx). */
export type ApiResponse<T> = {
  success: boolean
  message: string
  data?: T
  meta?: PaginationMeta
  error?: string
  details?: unknown
}

export type PaginationMeta = {
  page: number
  per_page: number
  total: number
  total_pages: number
}

export type User = {
  id: string
  name: string
  email: string
  role: Role
  tier: Tier
  email_verified_at: string | null
  created_at: string
  updated_at: string
}

export type LoginPayload = {
  email: string
  password: string
}

export type RegisterPayload = {
  name: string
  email: string
  password: string
}

export type LoginResult = {
  token: string
  expires_at: string
  user: User
}
