export type UsedToolMeta = {
  name: string
  label: string
  ok: boolean
  summary?: string
}

export type AssistantMessage = {
  id: string
  seq: number
  role: 'user' | 'assistant'
  content: string
  tools_used?: UsedToolMeta[]
  created_at: string
}

export type AssistantSession = {
  enabled: boolean
  session_id?: string
  last_message_at?: string | null
  last_compacted_at?: string | null
  compacted_before_seq: number
  active_message_count: number
  has_summary: boolean
  has_more: boolean
  messages: AssistantMessage[]
}

export type AssistantStreamEvent =
  | { type: 'status'; stage: 'compacting' | 'thinking' | 'streaming'; label: string }
  | { type: 'tool_start'; name: string; label: string }
  | { type: 'tool_done'; name: string; ok: boolean; summary?: string }
  | { type: 'action_applied'; kind: 'subscription_added' | 'subscription_removed'; symbol: string }
  | { type: 'delta'; text: string }
  | { type: 'done'; message_id: string; seq: number; compacted: boolean }
  | { type: 'error'; code?: string; message: string }
