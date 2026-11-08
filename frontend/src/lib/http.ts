import { ApiError } from './api-error'
import type { ApiResponse, PaginationMeta } from '@/types/api'

const DEFAULT_TIMEOUT_MS = 15_000

/**
 * Sesi disimpan backend di cookie HttpOnly sehingga tidak dapat dibaca
 * JavaScript. Karena itu setiap permintaan cukup mengirim `credentials`,
 * dan permintaan pengubah keadaan menyertakan header CSRF dari cookie
 * yang memang sengaja dapat dibaca.
 */

export const CSRF_COOKIE = 'ila_csrf'
export const CSRF_HEADER = 'X-CSRF-Token'

/** Membaca token CSRF dari cookie; kosong berarti sesi belum siap. */
export function readCsrfToken(): string {
  const match = document.cookie
    .split(';')
    .map((part) => part.trim())
    .find((part) => part.startsWith(`${CSRF_COOKIE}=`))

  if (!match) return ''
  return decodeURIComponent(match.slice(CSRF_COOKIE.length + 1))
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  query?: Record<string, string | number | boolean | undefined>
  signal?: AbortSignal
  timeoutMs?: number
}

/**
 * Dipanggil HANYA saat sesi tidak sah (401). 403 tidak boleh masuk sini:
 * mencampur keduanya membuat admin biasa ter-logout hanya karena membuka
 * menu yang bukan haknya (plans/23 §6).
 */
let unauthorizedHandler: (() => void) | null = null

export function onUnauthorized(handler: (() => void) | null): void {
  unauthorizedHandler = handler
}

/**
 * selamaPemulihan menandai bahwa 401 yang muncul bukan tanda sesi berakhir.
 *
 * Dipakai saat memulihkan sesi: pengunjung yang belum pernah masuk memang
 * menerima 401 dari /me, dan itu bukan alasan menampilkan "sesi Anda habis".
 * Penghitung dipakai, bukan boolean, agar dua permintaan yang bertumpuk
 * tidak saling membatalkan penanda.
 */
let penandaPemulihan = 0

export async function selamaPemulihan<T>(fn: () => Promise<T>): Promise<T> {
  penandaPemulihan++
  try {
    return await fn()
  } finally {
    penandaPemulihan--
  }
}

const SAFE_METHODS = new Set(['GET', 'HEAD', 'OPTIONS'])

/**
 * request menangani satu panggilan API: header, CSRF, timeout, parsing balasan
 * seragam, dan penerjemahan kegagalan menjadi ApiError.
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const payload = await requestEnvelope<T>(path, options)
  return payload.data as T
}

/**
 * requestList mengembalikan data beserta metadata paginasi. Daftar yang
 * panjang butuh `total` dan `has_more` untuk tombol halaman berikutnya.
 */
export async function requestList<T>(
  path: string,
  options: RequestOptions = {},
): Promise<{ data: T[]; meta?: PaginationMeta }> {
  const payload = await requestEnvelope<T[]>(path, options)
  return { data: payload.data ?? [], meta: payload.meta }
}

/** requestEnvelope adalah inti semua panggilan: header, CSRF, timeout, parsing. */
async function requestEnvelope<T>(
  path: string,
  options: RequestOptions = {},
): Promise<ApiResponse<T>> {
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

  // Double-submit: header harus sama dengan cookie agar backend menerima.
  if (!SAFE_METHODS.has(method)) {
    const csrf = readCsrfToken()
    if (csrf) headers.set(CSRF_HEADER, csrf)
  }

  try {
    const response = await fetch(buildUrl(path, query), {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
      credentials: 'same-origin',
    })

    const payload = await parseJson<T>(response)

    if (!response.ok || !payload.success) {
      // Sesi habis hanya bermakna bila endpoint memang butuh sesi.
      // 403 berarti sesi sah tetapi tidak berwenang: pengguna tetap masuk.
      if (response.status === 401 && penandaPemulihan === 0) {
        unauthorizedHandler?.()
      }

      throw new ApiError(
        messageForStatus(response.status, payload.message),
        response.status,
        payload.error,
        payload.details,
      )
    }

    return payload
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

/**
 * downloadFile mengambil berkas (CSV) lalu menyimpannya lewat tautan sementara.
 * Nama berkas diambil dari header agar server tetap yang menentukan.
 */
export async function downloadFile(
  path: string,
  query?: Record<string, string | undefined>,
): Promise<string> {
  const response = await fetch(buildUrl(path, query), {
    method: 'GET',
    headers: { Accept: 'text/csv' },
    credentials: 'same-origin',
  })

  if (!response.ok) {
    if (response.status === 401 && penandaPemulihan === 0) unauthorizedHandler?.()
    throw new ApiError(messageForStatus(response.status), response.status, 'DOWNLOAD_FAILED')
  }

  const disposition = response.headers.get('Content-Disposition') ?? ''
  const match = /filename="([^"]+)"/.exec(disposition)
  const filename = match?.[1] ?? 'unduhan.csv'

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  // Objek URL harus dilepas agar memori tab tidak menumpuk.
  URL.revokeObjectURL(url)

  return filename
}

/**
 * messageForStatus memberi pesan yang bisa ditindaklanjuti. Pesan mentah
 * backend dipakai bila ada; status khusus punya penjelasan sendiri.
 */
function messageForStatus(status: number, fallback?: string): string {
  switch (status) {
    case 403:
      return 'Anda tidak punya hak akses untuk tindakan ini.'
    case 423:
      return 'Sistem sedang dalam pemeliharaan. Silakan coba beberapa saat lagi.'
    case 429:
      return 'Terlalu banyak percobaan. Silakan tunggu sebentar lalu coba lagi.'
    default:
      return fallback || 'Permintaan gagal diproses.'
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
