import { ApiError } from '@/lib/api-error'
import { CSRF_HEADER, readCsrfToken, request } from '@/lib/http'
import type { AssistantSession, AssistantStreamEvent } from './types'

export const assistantApi = {
  getSession: (params?: { beforeSeq?: number; limit?: number }) =>
    request<AssistantSession>('/assistant/session', {
      query: {
        before_seq: params?.beforeSeq,
        limit: params?.limit,
      },
    }),

  compactSession: () =>
    request<AssistantSession>('/assistant/compact', {
      method: 'POST',
      timeoutMs: 45_000,
    }),
}

type StreamChatOptions = {
  message: string
  signal?: AbortSignal
  onEvent: (event: AssistantStreamEvent) => void
}

/**
 * Mengirim pertanyaan ke POST /api/v1/assistant/chat dan membaca aliran SSE
 * memakai fetch + ReadableStream agar cookie sesi HttpOnly dan header CSRF
 * tetap terkirim (EventSource standar hanya mendukung GET).
 */
export async function streamAssistantChat({
  message,
  signal,
  onEvent,
}: StreamChatOptions): Promise<void> {
  const headers = new Headers({
    Accept: 'text/event-stream',
    'Content-Type': 'application/json',
  })
  const csrf = readCsrfToken('/assistant/chat')
  if (csrf) {
    headers.set(CSRF_HEADER, csrf)
  }

  const response = await fetch('/api/v1/assistant/chat', {
    method: 'POST',
    headers,
    body: JSON.stringify({ message }),
    credentials: 'same-origin',
    signal,
  })

  if (!response.ok) {
    let msg = 'Gagal menghubungi ILA AI.'
    let code = 'ASSISTANT_ERROR'
    try {
      const payload = (await response.json()) as { message?: string; error?: string }
      if (payload?.message) msg = payload.message
      if (payload?.error) code = payload.error
    } catch {
      // Abaikan kegagalan parsing JSON error non-standar.
    }
    throw new ApiError(msg, response.status, code)
  }

  if (!response.body) {
    throw new ApiError('Aliran respons tidak tersedia.', response.status, 'STREAM_UNAVAILABLE')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  const dispatchBlock = (block: string) => {
    const lines = block.split(/\r?\n/)
    let eventName = ''
    const dataLines: string[] = []

    for (const line of lines) {
      if (!line || line.startsWith(':')) continue
      if (line.startsWith('event:')) {
        eventName = line.slice(6).trim()
      } else if (line.startsWith('data:')) {
        dataLines.push(line.slice(5).trim())
      }
    }
    if (dataLines.length === 0) return

    const rawData = dataLines.join('\n')
    try {
      const parsed = JSON.parse(rawData) as AssistantStreamEvent
      if (!parsed.type && eventName) {
        ;(parsed as { type: string }).type = eventName
      }
      onEvent(parsed)
    } catch {
      // Abaikan frame JSON tak lengkap.
    }
  }

  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const blocks = buffer.split(/\r?\n\r?\n/)
      buffer = blocks.pop() ?? ''

      for (const block of blocks) {
        if (block.trim()) dispatchBlock(block)
      }
    }
    if (buffer.trim()) {
      dispatchBlock(buffer)
    }
  } finally {
    reader.releaseLock()
  }
}
