import { useCallback, useEffect, useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowDown,
  ArrowUp,
  Check,
  History,
  MessageSquareText,
  RefreshCw,
  Square,
  AlertCircle,
} from 'lucide-react'
import { Alert, Badge, Button, Skeleton } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { queryKeys } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { assistantApi, streamAssistantChat } from '@/features/assistant/api'
import { MarkdownContent } from '@/features/assistant/MarkdownContent'
import { ThinkingIndicator } from '@/features/assistant/ThinkingIndicator'
import type { AssistantMessage, AssistantSession, UsedToolMeta } from '@/features/assistant/types'

const STARTER_PROMPTS = [
  'Analisa sinyal XAUUSD H4 terakhir',
  'Kenapa posisi terakhir saya kena SL?',
  'Cek daftar pair pantauan & kuota saya',
  'Jelaskan syarat valid C2 closure IRL',
] as const

const MAX_MESSAGE_CHARS = 2000
const HISTORY_PAGE_SIZE = 20
const SESSION_QUERY_KEY = ['assistant', 'session'] as const

export default function AssistantPage() {
  const queryClient = useQueryClient()

  const sessionQuery = useQuery({
    queryKey: SESSION_QUERY_KEY,
    queryFn: () => assistantApi.getSession({ limit: HISTORY_PAGE_SIZE }),
    staleTime: 60_000,
  })

  const session = sessionQuery.data
  const enabled = session?.enabled ?? true
  const messages = session?.messages ?? []
  const hasMore = Boolean(session?.has_more)
  const compactedBeforeSeq = session?.compacted_before_seq ?? 0
  const activeCount = session?.active_message_count ?? 0
  const hasSummary = Boolean(session?.has_summary)

  const [loadingOlder, setLoadingOlder] = useState(false)
  const [compactingManual, setCompactingManual] = useState(false)

  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [thinkingLabel, setThinkingLabel] = useState('Sedang berpikir...')
  const [activeToolLabel, setActiveToolLabel] = useState<string | null>(null)
  const [streamTools, setStreamTools] = useState<UsedToolMeta[]>([])
  const [streamActions, setStreamActions] = useState<string[]>([])
  const [streamContent, setStreamContent] = useState('')
  const [chatError, setChatError] = useState<string | null>(null)
  const [showJumpBottom, setShowJumpBottom] = useState(false)

  const scrollBoxRef = useRef<HTMLDivElement | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const stickBottomRef = useRef(true)
  const tempIdRef = useRef(0)

  useEffect(() => {
    return () => {
      abortRef.current?.abort()
    }
  }, [])

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'smooth') => {
    const el = scrollBoxRef.current
    if (!el) return
    el.scrollTo({ top: el.scrollHeight, behavior })
    stickBottomRef.current = true
    setShowJumpBottom(false)
  }, [])

  useEffect(() => {
    if (stickBottomRef.current) {
      scrollToBottom('auto')
    }
  }, [messages.length, streamContent, activeToolLabel, thinkingLabel, scrollToBottom])

  const handleScroll = () => {
    const el = scrollBoxRef.current
    if (!el) return
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    const nearBottom = distanceFromBottom < 96
    stickBottomRef.current = nearBottom
    setShowJumpBottom(!nearBottom)
  }

  const updateSessionCache = useCallback(
    (updater: (prev: AssistantSession) => AssistantSession) => {
      queryClient.setQueryData<AssistantSession>(SESSION_QUERY_KEY, (prev) => {
        const base: AssistantSession = prev ?? {
          enabled: true,
          compacted_before_seq: 0,
          active_message_count: 0,
          has_summary: false,
          has_more: false,
          messages: [],
        }
        return updater(base)
      })
    },
    [queryClient],
  )

  const handleLoadOlder = async () => {
    const firstMsg = messages[0]
    if (loadingOlder || !hasMore || !firstMsg) return
    const el = scrollBoxRef.current
    const prevHeight = el?.scrollHeight ?? 0

    // Lepas pin ke bawah sebelum menambah pesan lama agar layar tidak melompat
    // balik ke pesan terbaru ketika daftar pesan bertambah panjang.
    stickBottomRef.current = false
    setLoadingOlder(true)
    try {
      const older = await assistantApi.getSession({
        beforeSeq: firstMsg.seq,
        limit: HISTORY_PAGE_SIZE,
      })
      updateSessionCache((prev) => {
        const seen = new Set(prev.messages.map((m) => m.seq))
        const uniqueOlder = (older.messages ?? []).filter((m) => !seen.has(m.seq))
        return {
          ...prev,
          has_more: Boolean(older.has_more),
          messages: [...uniqueOlder, ...prev.messages],
        }
      })
      requestAnimationFrame(() => {
        if (el) {
          el.scrollTop = el.scrollHeight - prevHeight
        }
      })
    } catch (err) {
      setChatError(err instanceof ApiError ? err.message : 'Gagal memuat riwayat sebelumnya.')
    } finally {
      setLoadingOlder(false)
    }
  }

  const handleCompactManual = async () => {
    if (isStreaming || compactingManual || activeCount === 0) return
    setCompactingManual(true)
    setChatError(null)
    try {
      const fresh = await assistantApi.compactSession()
      queryClient.setQueryData(SESSION_QUERY_KEY, fresh)
    } catch (err) {
      setChatError(err instanceof ApiError ? err.message : 'Gagal meringkas topik percakapan.')
    } finally {
      setCompactingManual(false)
    }
  }

  const resizeTextarea = (el: HTMLTextAreaElement | null) => {
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 136)}px`
  }

  const sendMessage = async (rawText?: string) => {
    const text = (rawText ?? input).trim()
    if (!text || isStreaming || !enabled || sessionQuery.isLoading) return
    if (text.length > MAX_MESSAGE_CHARS) {
      setChatError(`Pesan maksimal ${MAX_MESSAGE_CHARS} karakter.`)
      return
    }

    setChatError(null)
    if (rawText === undefined) {
      setInput('')
      requestAnimationFrame(() => resizeTextarea(textareaRef.current))
    }

    tempIdRef.current += 1
    const tempSeq = (messages[messages.length - 1]?.seq ?? 0) + 1
    const tempUserMsg: AssistantMessage = {
      id: `temp-user-${tempIdRef.current}`,
      seq: tempSeq,
      role: 'user',
      content: text,
      created_at: '',
    }

    updateSessionCache((prev) => ({
      ...prev,
      messages: [...prev.messages, tempUserMsg],
    }))

    setIsStreaming(true)
    setThinkingLabel('Sedang berpikir...')
    setActiveToolLabel(null)
    setStreamTools([])
    setStreamActions([])
    setStreamContent('')
    stickBottomRef.current = true

    const controller = new AbortController()
    abortRef.current = controller

    let accText = ''
    let accTools: UsedToolMeta[] = []

    try {
      await streamAssistantChat({
        message: text,
        signal: controller.signal,
        onEvent: (ev) => {
          switch (ev.type) {
            case 'status':
              setThinkingLabel(ev.label || 'Sedang berpikir...')
              break
            case 'tool_start':
              setActiveToolLabel(ev.label || 'Memeriksa data...')
              break
            case 'tool_done': {
              setActiveToolLabel(null)
              const toolEntry: UsedToolMeta = {
                name: ev.name,
                label: ev.summary || ev.name,
                ok: ev.ok,
                summary: ev.summary,
              }
              accTools = [...accTools, toolEntry]
              setStreamTools(accTools)
              break
            }
            case 'action_applied':
              void queryClient.invalidateQueries({ queryKey: queryKeys.subscriptions })
              void queryClient.invalidateQueries({ queryKey: queryKeys.strategies })
              setStreamActions((prev) => [
                ...prev,
                ev.kind === 'subscription_added'
                  ? `${ev.symbol} ditambahkan ke Watchlist Pair`
                  : `${ev.symbol} dihapus dari Watchlist Pair`,
              ])
              break
            case 'delta':
              accText += ev.text
              setStreamContent(accText)
              break
            case 'done': {
              tempIdRef.current += 1
              const assistantMsg: AssistantMessage = {
                id: ev.message_id || `assistant-${tempIdRef.current}`,
                seq: ev.seq || tempSeq + 1,
                role: 'assistant',
                content: accText,
                tools_used: accTools,
                created_at: '',
              }
              updateSessionCache((prev) => ({
                ...prev,
                has_summary: prev.has_summary || ev.compacted,
                compacted_before_seq: ev.compacted
                  ? Math.max(prev.compacted_before_seq, tempSeq - 1)
                  : prev.compacted_before_seq,
                active_message_count: ev.compacted ? 2 : prev.active_message_count + 2,
                messages: [...prev.messages, assistantMsg],
              }))
              setStreamContent('')
              setStreamTools([])
              break
            }
            case 'error':
              setChatError(ev.message || 'Terjadi gangguan pada ILA AI.')
              break
          }
        },
      })
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') {
        if (accText.trim()) {
          tempIdRef.current += 1
          const stoppedMsg: AssistantMessage = {
            id: `aborted-${tempIdRef.current}`,
            seq: tempSeq + 1,
            role: 'assistant',
            content: `${accText}\n\n*(Respons dihentikan)*`,
            tools_used: accTools,
            created_at: '',
          }
          updateSessionCache((prev) => ({
            ...prev,
            messages: [...prev.messages, stoppedMsg],
          }))
        }
      } else {
        setChatError(err instanceof ApiError ? err.message : 'Gagal terhubung ke ILA AI.')
      }
    } finally {
      abortRef.current = null
      setIsStreaming(false)
      setActiveToolLabel(null)
      setStreamContent('')
    }
  }

  const handleStop = () => {
    abortRef.current?.abort()
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void sendMessage()
    }
  }

  const firstActiveIdx = messages.findIndex((m) => m.seq > compactedBeforeSeq)
  const loadErrorMsg = sessionQuery.isError
    ? sessionQuery.error instanceof ApiError
      ? sessionQuery.error.message
      : 'Gagal memuat sesi ILA AI.'
    : null

  return (
    <div className="mx-auto flex h-[calc(100dvh-9.75rem-env(safe-area-inset-bottom))] min-h-0 w-full max-w-3xl flex-col sm:h-[calc(100dvh-10rem-env(safe-area-inset-bottom))] lg:h-[calc(100dvh-7.5rem)]">
      {/* Header tipis ruang kerja */}
      <header className="flex shrink-0 items-center justify-between gap-2 border-b border-border pb-2.5 sm:pb-3">
        <div className="flex min-w-0 flex-1 items-center gap-2.5">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border bg-secondary/60">
            <MessageSquareText className="h-4 w-4 text-foreground" aria-hidden="true" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5 sm:gap-2">
              <h1 className="truncate text-[15px] font-semibold tracking-tight text-foreground">
                ILA AI
              </h1>
              <Badge variant="muted" className="shrink-0 text-[10.5px]">
                New ILA 2.0
              </Badge>
            </div>
            <p className="truncate text-[11.5px] text-muted-foreground sm:text-[12px]">
              {hasSummary
                ? 'Sesi aktif, ringkasan konteks tersimpan'
                : 'Terkunci pada aturan New ILA 2.0 dan data akun Anda'}
            </p>
          </div>
        </div>

        <Button
          variant="outline"
          size="sm"
          className="press shrink-0 gap-1.5 px-2.5 text-[12px] sm:px-3"
          disabled={isStreaming || compactingManual || activeCount === 0 || !enabled}
          onClick={() => void handleCompactManual()}
          title="Ringkas percakapan aktif ke memori agar konteks tetap ringan"
        >
          <RefreshCw
            className={cn('h-3.5 w-3.5', compactingManual && 'animate-spin')}
            aria-hidden="true"
          />
          <span className="sm:hidden">{compactingManual ? 'Meringkas...' : 'Topik baru'}</span>
          <span className="hidden sm:inline">
            {compactingManual ? 'Meringkas...' : 'Mulai topik baru'}
          </span>
        </Button>
      </header>

      {/* Area daftar pesan */}
      <div className="relative min-h-0 min-w-0 flex-1">
        <div
          ref={scrollBoxRef}
          onScroll={handleScroll}
          className="scrollbar-thin flex h-full min-w-0 flex-col overflow-y-auto py-3 pr-1 sm:py-4"
        >
          {sessionQuery.isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-12 w-3/4 rounded-xl" />
              <Skeleton className="ml-auto h-10 w-1/2 rounded-xl" />
              <Skeleton className="h-24 w-5/6 rounded-xl" />
            </div>
          ) : loadErrorMsg ? (
            <Alert variant="destructive" className="my-4">
              <div className="flex w-full flex-wrap items-center justify-between gap-2">
                <span>{loadErrorMsg}</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="press"
                  onClick={() => void sessionQuery.refetch()}
                >
                  Coba lagi
                </Button>
              </div>
            </Alert>
          ) : !enabled ? (
            <Alert variant="info" className="my-4">
              <span>
                ILA AI belum aktif di server saat ini. Hubungi admin apabila fitur asisten perlu
                dinyalakan.
              </span>
            </Alert>
          ) : messages.length === 0 && !isStreaming ? (
            <div className="my-auto flex flex-col items-start gap-3 py-3 sm:gap-4 sm:py-6">
              <div className="space-y-1.5">
                <h2 className="text-[15px] font-semibold tracking-tight text-foreground sm:text-[16px]">
                  Tanyakan setup, posisi, atau kelola watchlist Anda
                </h2>
                <p className="max-w-xl text-[13px] leading-relaxed text-muted-foreground">
                  Jawaban mengacu langsung pada dokumen baku New ILA 2.0 serta data sinyal,
                  posisi, jurnal, dan candle pasar dari akun Anda.
                </p>
              </div>

              <div className="grid w-full gap-2 sm:grid-cols-2">
                {STARTER_PROMPTS.map((prompt) => (
                  <button
                    key={prompt}
                    type="button"
                    onClick={() => void sendMessage(prompt)}
                    className="press hover-lift min-w-0 rounded-xl border border-border bg-card px-3.5 py-2.5 text-left text-[13px] text-foreground transition-colors hover:bg-secondary/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {prompt}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div className="space-y-5">
              {hasMore && (
                <div className="flex justify-center pb-1">
                  <Button
                    variant="outline"
                    size="sm"
                    className="press gap-1.5 rounded-full text-[12px] text-muted-foreground hover:text-foreground"
                    disabled={loadingOlder}
                    onClick={() => void handleLoadOlder()}
                  >
                    <History className="h-3.5 w-3.5" aria-hidden="true" />
                    <span>{loadingOlder ? 'Memuat riwayat...' : 'Muat riwayat sebelumnya'}</span>
                  </Button>
                </div>
              )}

              {messages.map((msg, idx) => {
                const showCompactionDivider =
                  compactedBeforeSeq > 0 && idx === firstActiveIdx && idx > 0
                return (
                  <div key={msg.id} className="space-y-4">
                    {showCompactionDivider && (
                      <div className="flex items-center gap-3 py-1 text-[11.5px] text-muted-foreground">
                        <div className="h-px flex-1 bg-border" />
                        <span>Konteks percakapan sebelumnya diringkas otomatis</span>
                        <div className="h-px flex-1 bg-border" />
                      </div>
                    )}

                    {msg.role === 'user' ? (
                      <div className="flex justify-end">
                        <div className="max-w-[88%] rounded-2xl rounded-br-sm bg-primary px-3.5 py-2.5 text-[13.5px] leading-relaxed text-primary-foreground sm:max-w-[80%]">
                          <p className="whitespace-pre-wrap break-words">{msg.content}</p>
                        </div>
                      </div>
                    ) : (
                      <div className="min-w-0 space-y-2 pr-1 sm:pr-2">
                        {msg.tools_used && msg.tools_used.length > 0 && (
                          <div className="flex flex-wrap gap-1.5">
                            {msg.tools_used.map((tool, tIdx) => (
                              <span
                                key={`${tool.name}-${tIdx}`}
                                className="inline-flex items-center gap-1 rounded-md border border-border bg-secondary/50 px-2 py-0.5 text-[11px] text-muted-foreground"
                              >
                                {tool.ok ? (
                                  <Check className="h-3 w-3 shrink-0 text-foreground" aria-hidden="true" />
                                ) : (
                                  <AlertCircle className="h-3 w-3 shrink-0 text-muted-foreground" aria-hidden="true" />
                                )}
                                <span>{tool.summary || tool.label}</span>
                              </span>
                            ))}
                          </div>
                        )}
                        <MarkdownContent content={msg.content} />
                      </div>
                    )}
                  </div>
                )
              })}

              {/* Status streaming & thinking saat respons sedang berlangsung */}
              {isStreaming && (
                <div className="min-w-0 space-y-2.5 pr-1 sm:pr-2">
                  {streamActions.length > 0 && (
                    <div className="flex flex-wrap gap-1.5">
                      {streamActions.map((act, idx) => (
                        <Badge key={idx} variant="muted" className="gap-1 text-[11px]">
                          <Check className="h-3 w-3" aria-hidden="true" />
                          <span>{act}</span>
                        </Badge>
                      ))}
                    </div>
                  )}

                  {streamContent ? (
                    <>
                      {streamTools.length > 0 && (
                        <div className="flex flex-wrap gap-1.5">
                          {streamTools.map((tool, idx) => (
                            <span
                              key={`${tool.name}-${idx}`}
                              className="inline-flex items-center gap-1 rounded-md border border-border bg-secondary/50 px-2 py-0.5 text-[11px] text-muted-foreground"
                            >
                              <Check className="h-3 w-3 shrink-0 text-foreground" aria-hidden="true" />
                              <span>{tool.summary || tool.label}</span>
                            </span>
                          ))}
                        </div>
                      )}
                      <MarkdownContent content={streamContent} />
                    </>
                  ) : (
                    <ThinkingIndicator
                      label={thinkingLabel}
                      tools={streamTools}
                      activeToolLabel={activeToolLabel}
                    />
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Tombol lompat ke bawah saat pengguna scroll ke atas */}
        {showJumpBottom && (
          <button
            type="button"
            onClick={() => scrollToBottom('smooth')}
            aria-label="Gulir ke pesan terbaru"
            className="press absolute bottom-3 right-3 flex h-9 w-9 items-center justify-center rounded-full border border-border bg-card text-foreground shadow-sm transition-[opacity,transform] duration-150 ease-out hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:h-8 sm:w-8"
          >
            <ArrowDown className="h-4 w-4" aria-hidden="true" />
          </button>
        )}
      </div>

      {/* Error inline saat chat */}
      {chatError && (
        <div className="shrink-0 pb-2">
          <Alert variant="destructive">{chatError}</Alert>
        </div>
      )}

      {/* Composer sticky di bawah */}
      <form
        onSubmit={(e) => {
          e.preventDefault()
          void sendMessage()
        }}
        className="shrink-0 space-y-1.5 border-t border-border pt-3"
      >
        <div className="flex items-end gap-2 rounded-2xl border border-input bg-card px-3 py-2 transition-colors focus-within:border-foreground/35">
          <textarea
            ref={textareaRef}
            rows={1}
            value={input}
            disabled={!enabled || sessionQuery.isLoading}
            onChange={(e) => {
              setInput(e.target.value)
              resizeTextarea(e.target)
            }}
            onKeyDown={handleKeyDown}
            placeholder={
              enabled
                ? 'Tanya analisa ILA 2.0, kenapa kena SL, atau minta pantau pair...'
                : 'ILA AI belum aktif'
            }
            aria-label="Pesan untuk ILA AI"
            className="max-h-36 min-h-[24px] flex-1 resize-none bg-transparent py-1 text-[16px] leading-relaxed text-foreground outline-none placeholder:text-muted-foreground focus:outline-none focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-60 sm:text-[13.5px]"
          />

          {isStreaming ? (
            <Button
              type="button"
              size="icon"
              variant="secondary"
              onClick={handleStop}
              aria-label="Hentikan respons"
              title="Hentikan respons"
              className="press h-9 w-9 shrink-0 rounded-xl sm:h-8 sm:w-8"
            >
              <Square className="h-3.5 w-3.5 fill-current" aria-hidden="true" />
            </Button>
          ) : (
            <Button
              type="submit"
              size="icon"
              disabled={!enabled || sessionQuery.isLoading || input.trim().length === 0}
              aria-label="Kirim pesan"
              title="Kirim pesan"
              className="press h-9 w-9 shrink-0 rounded-xl sm:h-8 sm:w-8"
            >
              <ArrowUp className="h-4 w-4" aria-hidden="true" />
            </Button>
          )}
        </div>

        <div className="flex items-center justify-between gap-2 px-1 text-[11px] leading-tight text-muted-foreground">
          <span>Mengacu New ILA 2.0 & data akun Anda. Bukan saran keuangan.</span>
          {input.length > MAX_MESSAGE_CHARS - 250 && (
            <span className="tabular shrink-0">
              {input.length}/{MAX_MESSAGE_CHARS}
            </span>
          )}
        </div>
      </form>
    </div>
  )
}
