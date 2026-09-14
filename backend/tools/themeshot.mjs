// Alat tema: thumbnail katalog & screenshot pembanding (butuh Google Chrome).
//
//   node tools/themeshot.mjs thumb <slug> [themedevBase=http://localhost:8090] [cdpPort=9401]
//       → themes/<slug>/assets/thumb.webp (cover 390×844, WebP)
//   node tools/themeshot.mjs strip <slug> <out.jpg> [themedevBase] [cdpPort]
//       → 1 gambar: cover + 3 layar pertama setelah "Buka Undangan" (untuk membandingkan kemiripan)
import { spawn } from 'node:child_process'
import { writeFileSync, mkdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { tmpdir } from 'node:os'

const [mode, slug, a3, a4, a5] = process.argv.slice(2)
const out = mode === 'strip' ? a3 : null
const base = (mode === 'strip' ? a4 : a3) || 'http://localhost:8090'
const port = +((mode === 'strip' ? a5 : a4) || 9401)
const W = 390, H = 844
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

try { await fetch(`http://127.0.0.1:${port}/json/version`) } catch {
  spawn('C:/Program Files/Google/Chrome/Application/chrome.exe', ['--headless=new', '--disable-gpu', '--hide-scrollbars',
    `--remote-debugging-port=${port}`, `--user-data-dir=${tmpdir()}/themeshot-${port}`, 'about:blank'], { stdio: 'ignore', detached: true }).unref()
}
let page
for (let k = 0; k < 100 && !page; k++) { try { page = await (await fetch(`http://127.0.0.1:${port}/json/new?about:blank`, { method: 'PUT' })).json() } catch { await sleep(300) } }
const ws = new WebSocket(page.webSocketDebuggerUrl)
await new Promise((r) => ws.addEventListener('open', r))
let id = 0; const pending = new Map()
ws.addEventListener('message', (e) => { const m = JSON.parse(e.data); if (m.id && pending.has(m.id)) { pending.get(m.id)(m.result); pending.delete(m.id) } })
const send = (method, params = {}) => new Promise((res) => { const i = ++id; pending.set(i, res); ws.send(JSON.stringify({ id: i, method, params })) })
const ev = async (expression) => (await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true }))?.result?.value
const shot = async (format = 'png') => Buffer.from((await send('Page.captureScreenshot', { format, quality: 82 })).data, 'base64')

await send('Page.enable')
await send('Network.enable')
await send('Network.setCacheDisabled', { cacheDisabled: true }) // CSS tema selalu versi terbaru
await send('Emulation.setDeviceMetricsOverride', { width: W, height: H, deviceScaleFactor: 1, mobile: true })
await send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-reduced-motion', value: 'reduce' }] })
await send('Page.navigate', { url: `${base}/${slug}?to=Bapak+Joko+Santoso` })
await sleep(800)
await ev('document.fonts.ready.then(() => true)') // tunggu web font
await sleep(1500) // animasi cover

if (mode === 'thumb') {
  const file = resolve(`themes/${slug}/assets/thumb.webp`)
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(file, await shot('webp'))
  console.log('thumbnail:', file)
} else {
  const frames = [await shot('jpeg')]
  await ev(`(document.querySelector('[data-open],.cover button,.btn--open,button') || {click(){}}).click()`)
  await sleep(1200)
  for (let i = 1; i <= 3; i++) { await ev(`window.scrollTo(0, ${i * H})`); await sleep(900); frames.push(await shot('jpeg')) }
  const html = `<body style="margin:0;background:#111;display:flex;gap:6px;padding:6px">${frames.map((b) => `<img width="${W}" src="data:image/jpeg;base64,${b.toString('base64')}">`).join('')}</body>`
  await send('Emulation.setDeviceMetricsOverride', { width: W * 4 + 30, height: H + 12, deviceScaleFactor: 1, mobile: false })
  await send('Page.navigate', { url: 'data:text/html;base64,' + Buffer.from(html).toString('base64') })
  await sleep(1200)
  writeFileSync(out, Buffer.from((await send('Page.captureScreenshot', { format: 'jpeg', quality: 70 })).data, 'base64'))
  console.log('strip:', out)
}
await send('Page.close').catch(() => {})
process.exit(0)
