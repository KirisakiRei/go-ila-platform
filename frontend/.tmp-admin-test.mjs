import { spawn } from 'node:child_process'
import fs from 'node:fs'

const CHROME = 'C:/Program Files/Google/Chrome/Application/chrome.exe'
const PORT = 9340
const proc = spawn(CHROME, [
  '--headless=new',
  '--remote-debugging-port=' + PORT,
  '--disable-gpu',
  '--hide-scrollbars',
  '--window-size=1440,1000',
  '--user-data-dir=C:/tmp/ila-admin-test',
  'about:blank',
])
const wait = (ms) => new Promise((r) => setTimeout(r, ms))

async function target() {
  for (let i = 0; i < 40; i++) {
    try {
      const list = await (await fetch('http://127.0.0.1:' + PORT + '/json')).json()
      const p = list.find((t) => t.type === 'page')
      if (p) return p.webSocketDebuggerUrl
    } catch {}
    await wait(300)
  }
  throw new Error('chrome tidak siap')
}

const ws = new WebSocket(await target())
await new Promise((r) => (ws.onopen = r))
let id = 0
const pending = new Map()
ws.onmessage = (e) => {
  const m = JSON.parse(e.data)
  if (m.id && pending.has(m.id)) {
    pending.get(m.id)(m.result)
    pending.delete(m.id)
  }
}
const send = (method, params = {}) =>
  new Promise((r) => {
    const i = ++id
    pending.set(i, r)
    ws.send(JSON.stringify({ id: i, method, params }))
  })

const evalJs = async (expression) => {
  const r = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })
  return r.result?.value
}

await send('Page.enable')
await send('Runtime.enable')
await send('Network.enable')
await send('Network.setCacheDisabled', { cacheDisabled: true })

fs.mkdirSync('.tmp-admin', { recursive: true })

// 1. Panel admin tanpa sesi harus mendarat di pintu admin.
await send('Page.navigate', { url: 'https://rekavia.com/app/admin' })
await wait(3500)
console.log('1) buka /app/admin tanpa sesi ->', await evalJs('location.pathname'))
console.log('   judul halaman           ->', await evalJs('document.querySelector("h1")?.innerText'))

// 2. Isi formulir dan masuk.
await evalJs(`(() => {
  const set = (el, v) => {
    const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set
    setter.call(el, v)
    el.dispatchEvent(new Event('input', { bubbles: true }))
  }
  set(document.querySelector('#admin-identifier'), 'superadmin@rekavia.com')
  set(document.querySelector('#admin-password'), 'password@2026')
  return true
})()`)
await wait(400)
await evalJs('document.querySelector("form").requestSubmit()')
await wait(4000)

console.log('2) setelah login            ->', await evalJs('location.pathname'))
const shot = await send('Page.captureScreenshot', { format: 'png' })
fs.writeFileSync('.tmp-admin/panel.png', Buffer.from(shot.data, 'base64'))

console.log('3) menu admin terlihat      ->', await evalJs(
  'Array.from(document.querySelectorAll("nav a")).slice(0,6).map(a=>a.innerText.trim()).join(" | ")',
))

// 4. Buka pengelolaan konten halaman.
await send('Page.navigate', { url: 'https://rekavia.com/app/admin/konten/halaman' })
await wait(3500)
console.log('4) halaman konten           ->', await evalJs('location.pathname'))
console.log('   judul                    ->', await evalJs('document.querySelector("h1")?.innerText'))
const shot2 = await send('Page.captureScreenshot', { format: 'png' })
fs.writeFileSync('.tmp-admin/konten.png', Buffer.from(shot2.data, 'base64'))

ws.close()
proc.kill()
process.exit(0)
