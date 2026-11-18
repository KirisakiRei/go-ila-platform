/* Service Worker khusus Web Push ILA Platform.
 * Sengaja TIDAK mendaftarkan handler `fetch` agar navigasi dan cache HTTP
 * antara situs publik (/) dan panel pelanggan (/app/*) tetap murni dikelola browser.
 */

self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
})

self.addEventListener('push', (event) => {
  let payload = {
    title: 'ILA Platform',
    body: 'Pembaruan aktivitas trading tersedia.',
    url: '/app/dashboard',
    tag: 'ila-notification',
    kind: 'entry',
    require_interaction: false,
  }

  if (event.data) {
    try {
      const parsed = event.data.json()
      if (parsed && typeof parsed === 'object') {
        payload = { ...payload, ...parsed }
      }
    } catch {
      const rawText = event.data.text()
      if (rawText) {
        payload.body = rawText
      }
    }
  }

  const targetUrl =
    typeof payload.url === 'string' && payload.url.startsWith('/')
      ? payload.url
      : '/app/dashboard'

  const showTask = self.registration.showNotification(payload.title || 'ILA Platform', {
    body: payload.body || '',
    icon: '/brand/pwa-192.png',
    badge: '/brand/favicon-64.png',
    tag: payload.tag || 'ila-notification',
    requireInteraction: Boolean(payload.require_interaction),
    data: {
      url: targetUrl,
      kind: payload.kind || 'entry',
    },
  })

  const notifyTabsTask = self.clients
    .matchAll({ type: 'window', includeUncontrolled: true })
    .then((windowClients) => {
      for (const client of windowClients) {
        client.postMessage({
          type: 'ILA_PUSH_EVENT',
          kind: payload.kind || 'entry',
          url: targetUrl,
        })
      }
    })
    .catch(() => undefined)

  event.waitUntil(Promise.all([showTask, notifyTabsTask]))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()

  const rawUrl = event.notification?.data?.url
  const targetPath =
    typeof rawUrl === 'string' && rawUrl.startsWith('/') ? rawUrl : '/app/dashboard'
  const absoluteTarget = new URL(targetPath, self.location.origin).href

  event.waitUntil(
    self.clients
      .matchAll({ type: 'window', includeUncontrolled: true })
      .then((windowClients) => {
        for (const client of windowClients) {
          if (client.url.startsWith(self.location.origin + '/app') && 'focus' in client) {
            return client
              .navigate(absoluteTarget)
              .catch(() => client)
              .then((c) => (c && 'focus' in c ? c.focus() : undefined))
          }
        }
        if (self.clients.openWindow) {
          return self.clients.openWindow(absoluteTarget)
        }
        return undefined
      }),
  )
})
