import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Bell,
  BellOff,
  CheckCircle2,
  Loader2,
  Send,
  Share,
  Smartphone,
  Trash2,
} from 'lucide-react'
import { Alert, Badge, Button, Skeleton } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { formatDateTime } from '@/lib/format'
import {
  detectPushEnvironment,
  getCurrentPushEndpoint,
  pushApi,
  subscribeCurrentBrowser,
  unsubscribeCurrentBrowser,
  type PushSubscriptionItem,
} from '@/lib/push-notifications'
import { cn } from '@/lib/utils'

export const PUSH_STATUS_QUERY_KEY = ['me', 'push', 'status'] as const

export function PushNotificationPanel() {
  const queryClient = useQueryClient()
  const [env, setEnv] = useState(() => detectPushEnvironment())
  const [localEndpoint, setLocalEndpoint] = useState<string | null>(null)
  const [notice, setNotice] = useState<{ tone: 'success' | 'error'; text: string } | null>(null)

  const statusQuery = useQuery({
    queryKey: PUSH_STATUS_QUERY_KEY,
    queryFn: pushApi.status,
  })

  useEffect(() => {
    let active = true
    void getCurrentPushEndpoint().then((ep) => {
      if (active) setLocalEndpoint(ep)
    })
    return () => {
      active = false
    }
  }, [statusQuery.dataUpdatedAt])

  const subscriptions = statusQuery.data?.subscriptions ?? []
  const vapidPublicKey = statusQuery.data?.vapid_public_key ?? ''

  // Perangkat ini dianggap aktif bila browser memiliki subscription dan ada
  // minimal 1 perangkat terdaftar di akun pengguna.
  const isThisDeviceActive = Boolean(localEndpoint) && subscriptions.length > 0

  const subscribeMutation = useMutation({
    mutationFn: async () => {
      if (!vapidPublicKey) {
        throw new Error('Kunci publik VAPID belum tersedia dari server.')
      }
      const browserSub = await subscribeCurrentBrowser(vapidPublicKey)
      setLocalEndpoint(browserSub.endpoint)
      setEnv(detectPushEnvironment())
      return pushApi.subscribe({
        endpoint: browserSub.endpoint,
        keys: browserSub.keys,
        notify_entry: true,
        notify_exit: true,
      })
    },
    onSuccess: () => {
      setNotice({
        tone: 'success',
        text: 'Notifikasi Web Push aktif di perangkat ini.',
      })
      void queryClient.invalidateQueries({ queryKey: PUSH_STATUS_QUERY_KEY })
    },
    onError: (err) => {
      setEnv(detectPushEnvironment())
      setNotice({
        tone: 'error',
        text:
          err instanceof ApiError || err instanceof Error
            ? err.message
            : 'Gagal mengaktifkan notifikasi di perangkat ini.',
      })
    },
  })

  const unsubscribeMutation = useMutation({
    mutationFn: async () => {
      const endpoint = await unsubscribeCurrentBrowser()
      if (endpoint) {
        await pushApi.unsubscribeEndpoint(endpoint)
      }
      setLocalEndpoint(null)
    },
    onSuccess: () => {
      setNotice({
        tone: 'success',
        text: 'Notifikasi Web Push pada perangkat ini telah dinonaktifkan.',
      })
      void queryClient.invalidateQueries({ queryKey: PUSH_STATUS_QUERY_KEY })
    },
    onError: (err) => {
      setNotice({
        tone: 'error',
        text:
          err instanceof ApiError || err instanceof Error
            ? err.message
            : 'Gagal menonaktifkan notifikasi perangkat ini.',
      })
    },
  })

  const updatePrefMutation = useMutation({
    mutationFn: ({
      id,
      notify_entry,
      notify_exit,
    }: {
      id: string
      notify_entry?: boolean
      notify_exit?: boolean
    }) => pushApi.updatePreferences(id, { notify_entry, notify_exit }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: PUSH_STATUS_QUERY_KEY })
    },
    onError: (err) => {
      setNotice({
        tone: 'error',
        text: err instanceof Error ? err.message : 'Gagal menyimpan preferensi notifikasi.',
      })
    },
  })

  const removeDeviceMutation = useMutation({
    mutationFn: (id: string) => pushApi.remove(id),
    onSuccess: () => {
      setNotice({ tone: 'success', text: 'Perangkat dihapus dari daftar notifikasi push.' })
      void queryClient.invalidateQueries({ queryKey: PUSH_STATUS_QUERY_KEY })
    },
    onError: (err) => {
      setNotice({
        tone: 'error',
        text: err instanceof Error ? err.message : 'Gagal menghapus perangkat.',
      })
    },
  })

  const testMutation = useMutation({
    mutationFn: () => pushApi.sendTest(),
    onSuccess: (res) => {
      if (res.delivered > 0) {
        setNotice({
          tone: 'success',
          text: `Notifikasi uji berhasil dikirim ke ${res.delivered} perangkat.`,
        })
      } else {
        setNotice({
          tone: 'error',
          text: 'Tidak ada perangkat yang menerima notifikasi uji. Coba aktifkan ulang perangkat ini.',
        })
      }
      void queryClient.invalidateQueries({ queryKey: PUSH_STATUS_QUERY_KEY })
    },
    onError: (err) => {
      setNotice({
        tone: 'error',
        text:
          err instanceof ApiError || err instanceof Error
            ? err.message
            : 'Gagal mengirim tes notifikasi.',
      })
    },
  })

  if (statusQuery.isLoading) {
    return <Skeleton className="h-28 w-full rounded-xl" />
  }

  return (
    <div className="space-y-3.5 text-xs">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="font-medium text-muted-foreground">Status Perangkat Ini:</span>
          {isThisDeviceActive ? (
            <span className="inline-flex items-center gap-1 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:text-emerald-400">
              <CheckCircle2 className="h-3 w-3" />
              Aktif
            </span>
          ) : env.needsIOSHomeScreen ? (
            <Badge variant="outline" className="text-amber-700 dark:text-amber-400">
              Perlu Tambah ke Layar Utama
            </Badge>
          ) : (
            <Badge variant="outline" className="text-muted-foreground">
              Belum Aktif
            </Badge>
          )}
        </div>

        {subscriptions.length > 0 && (
          <span className="text-[11px] text-muted-foreground">
            {subscriptions.length} perangkat terdaftar
          </span>
        )}
      </div>

      <p className="leading-relaxed text-muted-foreground">
        Notifikasi push browser/aplikasi dikirim secara hemat <strong>hanya</strong> saat{' '}
        <strong className="text-foreground">sinyal siap entry manual</strong>,{' '}
        <strong className="text-foreground">posisi entry baru dibuka</strong>, atau{' '}
        <strong className="text-foreground">posisi ditutup (Take Profit, Stop Loss, / Manual)</strong>.
        Tahap pantauan radar awal tidak pernah membunyikan push.
      </p>

      {/* Panduan khusus iPhone / iPad Safari yang belum dipasang sebagai PWA */}
      {env.needsIOSHomeScreen && (
        <div className="space-y-2 rounded-xl border border-amber-500/30 bg-amber-500/10 p-3 text-[11.5px]">
          <p className="font-semibold text-foreground">
            Cara Mengaktifkan Web Push di iPhone / iPad (iOS 16.4+):
          </p>
          <ol className="list-decimal space-y-1 pl-4 leading-relaxed text-muted-foreground">
            <li>
              Ketuk tombol <strong>Bagikan (Share)</strong>{' '}
              <Share className="inline h-3 w-3 text-foreground" /> di bilah bawah/atas Safari.
            </li>
            <li>
              Pilih menu <strong>"Tambahkan ke Layar Utama" (Add to Home Screen)</strong>.
            </li>
            <li>
              Buka ikon <strong>ILA Platform</strong> dari Layar Utama Anda, lalu kembali ke menu
              ini dan ketuk tombol <strong>Aktifkan di Perangkat Ini</strong>.
            </li>
          </ol>
        </div>
      )}

      {!env.needsIOSHomeScreen && !env.supported && (
        <Alert variant="warning">
          Browser ini belum mendukung standar Web Push / Service Worker. Gunakan Chrome, Edge,
          Firefox, atau Safari versi terbaru (atau gunakan Bot Telegram).
        </Alert>
      )}

      {env.permission === 'denied' && (
        <Alert variant="warning">
          Izin notifikasi sedang diblokir di browser ini. Buka pengaturan izin situs (ikon gembok
          di bilah alamat), ubah <strong>Notifikasi</strong> menjadi <strong>Izinkan</strong>, lalu
          coba lagi.
        </Alert>
      )}

      {notice && (
        <Alert variant={notice.tone === 'success' ? 'success' : 'destructive'}>{notice.text}</Alert>
      )}

      {/* Tombol Aksi Utama Perangkat Ini */}
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        {env.supported && !env.needsIOSHomeScreen && (
          <>
            {!isThisDeviceActive ? (
              <Button
                size="sm"
                className="w-full gap-1.5 sm:w-auto"
                disabled={subscribeMutation.isPending || env.permission === 'denied'}
                isLoading={subscribeMutation.isPending}
                onClick={() => {
                  setNotice(null)
                  subscribeMutation.mutate()
                }}
              >
                <Bell className="h-3.5 w-3.5" />
                Aktifkan di Perangkat Ini
              </Button>
            ) : (
              <Button
                size="sm"
                variant="outline"
                className="w-full gap-1.5 sm:w-auto"
                disabled={unsubscribeMutation.isPending}
                isLoading={unsubscribeMutation.isPending}
                onClick={() => {
                  setNotice(null)
                  unsubscribeMutation.mutate()
                }}
              >
                <BellOff className="h-3.5 w-3.5" />
                Matikan di Perangkat Ini
              </Button>
            )}
          </>
        )}

        {subscriptions.length > 0 && (
          <Button
            size="sm"
            variant="secondary"
            className="w-full gap-1.5 sm:w-auto"
            disabled={testMutation.isPending}
            isLoading={testMutation.isPending}
            onClick={() => {
              setNotice(null)
              testMutation.mutate()
            }}
          >
            <Send className="h-3.5 w-3.5" />
            Kirim Tes Notifikasi
          </Button>
        )}
      </div>

      {/* Daftar Perangkat Terdaftar & Saklar Entry / Exit */}
      {subscriptions.length > 0 && (
        <div className="space-y-2 border-t border-border pt-3">
          <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            Perangkat & Preferensi Pemicu
          </p>
          <div className="divide-y divide-border rounded-xl border border-border bg-card">
            {subscriptions.map((sub) => (
              <DeviceSubscriptionRow
                key={sub.id}
                item={sub}
                busy={
                  (updatePrefMutation.isPending && updatePrefMutation.variables?.id === sub.id) ||
                  (removeDeviceMutation.isPending && removeDeviceMutation.variables === sub.id)
                }
                onToggleEntry={(val) =>
                  updatePrefMutation.mutate({ id: sub.id, notify_entry: val })
                }
                onToggleExit={(val) =>
                  updatePrefMutation.mutate({ id: sub.id, notify_exit: val })
                }
                onRemove={() => removeDeviceMutation.mutate(sub.id)}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function DeviceSubscriptionRow({
  item,
  busy,
  onToggleEntry,
  onToggleExit,
  onRemove,
}: {
  item: PushSubscriptionItem
  busy: boolean
  onToggleEntry: (next: boolean) => void
  onToggleExit: (next: boolean) => void
  onRemove: () => void
}) {
  return (
    <div className="space-y-2.5 p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 flex-1 items-start gap-2.5">
          <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-border bg-secondary text-foreground">
            <Smartphone className="h-3.5 w-3.5" />
          </span>
          <div className="min-w-0 flex-1">
            <p className="truncate text-xs font-semibold text-foreground">{item.device_label}</p>
            <p className="truncate text-[10.5px] text-muted-foreground">
              {item.last_sent_at
                ? `Terakhir terkirim ${formatDateTime(item.last_sent_at)}`
                : `Terdaftar ${formatDateTime(item.created_at)}`}
            </p>
          </div>
        </div>

        <button
          type="button"
          disabled={busy}
          onClick={onRemove}
          aria-label={`Hapus perangkat ${item.device_label}`}
          className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="grid grid-cols-1 gap-2 rounded-lg border border-border/70 bg-secondary/20 p-2.5 sm:grid-cols-2">
        <div className="flex items-center justify-between gap-2">
          <span className="text-[11px] font-medium text-foreground">
            Sinyal Siap & Entry Dibuka
          </span>
          <PushSwitch checked={item.notify_entry} busy={busy} onChange={onToggleEntry} />
        </div>
        <div className="flex items-center justify-between gap-2">
          <span className="text-[11px] font-medium text-foreground">
            Posisi Tutup (TP / SL / Manual)
          </span>
          <PushSwitch checked={item.notify_exit} busy={busy} onChange={onToggleExit} />
        </div>
      </div>
    </div>
  )
}

function PushSwitch({
  checked,
  busy,
  onChange,
}: {
  checked: boolean
  busy: boolean
  onChange: (next: boolean) => void
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={busy}
      onClick={() => onChange(!checked)}
      className={cn(
        'inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border p-0.5 transition-colors',
        checked ? 'border-primary bg-primary' : 'border-border bg-secondary hover:bg-secondary/80',
        busy && 'cursor-wait opacity-60',
      )}
    >
      <span
        className={cn(
          'pointer-events-none flex h-5 w-5 items-center justify-center rounded-full bg-background shadow-xs transition-transform duration-200',
          checked ? 'translate-x-5' : 'translate-x-0',
        )}
      >
        {busy && <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />}
      </span>
    </button>
  )
}
