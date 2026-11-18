import { request } from './http'

export type PushSubscriptionItem = {
  id: string
  device_label: string
  notify_entry: boolean
  notify_exit: boolean
  last_sent_at: string | null
  created_at: string
  updated_at: string
  endpoint_host: string
}

export type PushStatusResponse = {
  supported: boolean
  vapid_public_key: string
  subscriptions: PushSubscriptionItem[]
}

export type PushTestResponse = {
  delivered: number
  expired: number
  failed: number
}

export const pushApi = {
  status: () => request<PushStatusResponse>('/me/push/status?page=1&per_page=10'),
  subscribe: (payload: {
    endpoint: string
    keys: { p256dh: string; auth: string }
    device_label?: string
    notify_entry?: boolean
    notify_exit?: boolean
  }) =>
    request<PushSubscriptionItem>('/me/push/subscriptions', {
      method: 'POST',
      body: payload,
    }),
  updatePreferences: (
    id: string,
    payload: { notify_entry?: boolean; notify_exit?: boolean },
  ) =>
    request<PushSubscriptionItem>(`/me/push/subscriptions/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: payload,
    }),
  remove: (id: string) =>
    request<{ deleted: boolean }>(`/me/push/subscriptions/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  unsubscribeEndpoint: (endpoint: string) =>
    request<{ unsubscribed: boolean }>('/me/push/unsubscribe', {
      method: 'POST',
      body: { endpoint },
    }),
  sendTest: () =>
    request<PushTestResponse>('/me/push/test', {
      method: 'POST',
      body: {},
    }),
}

export type BrowserPushEnvironment = {
  supported: boolean
  isIOS: boolean
  isStandalone: boolean
  needsIOSHomeScreen: boolean
  permission: NotificationPermission | 'unsupported'
}

export function detectPushEnvironment(): BrowserPushEnvironment {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') {
    return {
      supported: false,
      isIOS: false,
      isStandalone: false,
      needsIOSHomeScreen: false,
      permission: 'unsupported',
    }
  }

  const ua = navigator.userAgent || ''
  const isIOS =
    /iPad|iPhone|iPod/i.test(ua) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)

  const navStandalone = Boolean((navigator as Navigator & { standalone?: boolean }).standalone)
  const mediaStandalone =
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(display-mode: standalone)').matches
  const isStandalone = navStandalone || mediaStandalone

  const hasSW = 'serviceWorker' in navigator
  const hasPush = 'PushManager' in window
  const hasNotification = 'Notification' in window
  const supported = hasSW && hasPush && hasNotification

  return {
    supported,
    isIOS,
    isStandalone,
    needsIOSHomeScreen: isIOS && !isStandalone,
    permission: hasNotification ? Notification.permission : 'unsupported',
  }
}

export function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = window.atob(base64)
  const buffer = new ArrayBuffer(rawData.length)
  const outputArray = new Uint8Array(buffer)
  for (let i = 0; i < rawData.length; i += 1) {
    outputArray[i] = rawData.charCodeAt(i)
  }
  return outputArray
}

function arrayBufferToBase64Url(buf: ArrayBuffer | null): string {
  if (!buf) return ''
  const bytes = new Uint8Array(buf)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i += 1) {
    binary += String.fromCharCode(bytes[i] ?? 0)
  }
  return window.btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

export async function ensurePushServiceWorker(): Promise<ServiceWorkerRegistration> {
  const existing = await navigator.serviceWorker.getRegistration('/')
  if (existing) {
    return existing
  }
  const reg = await navigator.serviceWorker.register('/sw.js', { scope: '/' })
  await navigator.serviceWorker.ready
  return reg
}

export async function getCurrentPushEndpoint(): Promise<string | null> {
  if (typeof navigator === 'undefined' || !('serviceWorker' in navigator)) {
    return null
  }
  try {
    const reg = await navigator.serviceWorker.getRegistration('/')
    if (!reg || !reg.pushManager) return null
    const sub = await reg.pushManager.getSubscription()
    return sub?.endpoint ?? null
  } catch {
    return null
  }
}

export async function subscribeCurrentBrowser(vapidPublicKey: string): Promise<{
  endpoint: string
  keys: { p256dh: string; auth: string }
}> {
  const permission = await Notification.requestPermission()
  if (permission !== 'granted') {
    throw new Error(
      permission === 'denied'
        ? 'Izin notifikasi ditolak di pengaturan browser. Aktifkan izin notifikasi pada ikon gembok alamat situs.'
        : 'Izin notifikasi belum diberikan.',
    )
  }

  const reg = await ensurePushServiceWorker()
  let sub = await reg.pushManager.getSubscription()
  if (!sub) {
    sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
    })
  }

  const p256dh = arrayBufferToBase64Url(sub.getKey('p256dh'))
  const auth = arrayBufferToBase64Url(sub.getKey('auth'))
  if (!sub.endpoint || !p256dh || !auth) {
    throw new Error('Browser tidak mengembalikan kunci enkripsi Web Push yang lengkap.')
  }

  return {
    endpoint: sub.endpoint,
    keys: { p256dh, auth },
  }
}

export async function unsubscribeCurrentBrowser(): Promise<string | null> {
  if (typeof navigator === 'undefined' || !('serviceWorker' in navigator)) {
    return null
  }
  const reg = await navigator.serviceWorker.getRegistration('/')
  const sub = await reg?.pushManager?.getSubscription()
  if (!sub) return null
  const endpoint = sub.endpoint
  await sub.unsubscribe().catch(() => false)
  return endpoint
}
