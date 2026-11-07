import { ApiError } from './api-error'
import type { ApiResponse } from '@/types/api'

const DEFAULT_TIMEOUT_MS = 15_000

const AUTH_TOKEN_KEY = 'ila.token'

/**
 * Token disimpan di localStorage agar sesi bertahan saat halaman dimuat ulang.
 * Untuk produksi jangka panjang, pertimbangkan pindah ke cookie HttpOnly
 * agar tidak terjangkau JavaScript.
 */
export const tokenStorage = {
  get(): string | null {
    return localStorage.getItem(AUTH_TOKEN_KEY)
  },
  set(token: string): void {
    localStorage.setItem(AUTH_TOKEN_KEY, token)
  },
  clear(): void {
    localStorage.removeItem(AUTH_TOKEN_KEY)
  },
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  query?: Record<string, string | number | boolean | undefined>
  signal?: AbortSignal
  timeoutMs?: number
}

/** Dipanggil saat backend menolak token; memutus sesi di sisi UI. */
let unauthorizedHandler: (() => void) | null = null

export function onUnauthorized(handler: () => void): void {
  unauthorizedHandler = handler
}

/**
 * request menangani satu panggilan API: header, timeout, parsing balasan
 * seragam, dan penerjemahan kegagalan menjadi ApiError.
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, signal, timeoutMs = DEFAULT_TIMEOUT_MS } = options

  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), timeoutMs)

  // Abort dari pemanggil juga harus membatalkan permintaan.
  if (signal) {
    signal.addEventListener('abort', () => controller.abort(), { once: true })
  }

  const headers = new Headers({ Accept: 'application/json' })
  if (body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }

  const token = tokenStorage.get()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  try {
    const response = await fetch(buildUrl(path, query), {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
      credentials: 'same-origin',
    })

    if (response.status === 401) {
      unauthorizedHandler?.()
    }

    const payload = await parseJson<T>(response)

    if (!response.ok || !payload.success) {
      throw new ApiError(
        payload.message || 'Permintaan gagal diproses.',
        response.status,
        payload.error,
        payload.details,
      )
    }

    return payload.data as T
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new ApiError('Permintaan melebihi batas waktu. Silakan coba lagi.', 0, 'TIMEOUT')
    }
    throw new ApiError('Tidak dapat menghubungi server. Periksa koneksi Anda.', 0, 'NETWORK')
  } finally {
    clearTimeout(timer)
  }
}

async function parseJson<T>(response: Response): Promise<ApiResponse<T>> {
  try {
    return (await response.json()) as ApiResponse<T>
  } catch {
    throw new ApiError('Balasan server tidak dapat dibaca.', response.status, 'INVALID_RESPONSE')
  }
}

/** buildUrl menyusun URL lengkap; parameter kosong dilewati. */
function buildUrl(
  path: string,
  query?: Record<string, string | number | boolean | undefined>,
): string {
  if (!query) return `/api/v1${path}`

  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }

  const search = params.toString()
  return search ? `/api/v1${path}?${search}` : `/api/v1${path}`
}
