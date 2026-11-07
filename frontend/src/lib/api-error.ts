/**
 * ApiError membawa status HTTP dan kode error dari backend
 * sehingga UI bisa membedakan salah kredensial, validasi, dan gangguan server.
 */
export class ApiError extends Error {
  readonly status: number
  readonly code?: string
  readonly details?: unknown

  constructor(message: string, status: number, code?: string, details?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }

  /** Benar bila permintaan gagal karena sesi tidak sah atau kedaluwarsa. */
  get isUnauthorized(): boolean {
    return this.status === 401
  }

  /** Benar bila kegagalan terjadi di konektivitas, bukan pada data. */
  get isNetworkProblem(): boolean {
    return this.status === 0
  }
}
