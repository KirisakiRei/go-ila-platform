const MAX_RAW_BYTES = 30 * 1024 * 1024
export const MAX_UPLOAD_BYTES = 5 * 1024 * 1024
const TARGET_BYTES = Math.round(1.2 * 1024 * 1024)
const BACKEND_MIMES = new Set(['image/png', 'image/jpeg', 'image/webp', 'image/gif'])
const ACCEPTED_MIMES = new Set([
  'image/png',
  'image/jpeg',
  'image/jpg',
  'image/webp',
  'image/gif',
  'image/bmp',
  'image/avif',
])
const ACCEPTED_EXT = /\.(png|jpe?g|webp|gif|bmp|avif)$/i

export function isSupportedChartImage(file: File): boolean {
  const mime = file.type.toLowerCase()
  if (mime) return ACCEPTED_MIMES.has(mime)
  return ACCEPTED_EXT.test(file.name)
}

export function validateChartImageInput(file: File): string | null {
  if (!isSupportedChartImage(file)) {
    return 'Pilih gambar PNG, JPEG, WebP, atau GIF.'
  }
  if (file.size > MAX_RAW_BYTES) {
    return 'Ukuran gambar terlalu besar (maksimal 30 MiB sebelum kompresi).'
  }
  return null
}

function normalizeMime(file: File): string {
  const mime = file.type.toLowerCase()
  if (mime === 'image/jpg') return 'image/jpeg'
  if (mime) return mime
  const name = file.name.toLowerCase()
  if (name.endsWith('.png')) return 'image/png'
  if (name.endsWith('.jpg') || name.endsWith('.jpeg')) return 'image/jpeg'
  if (name.endsWith('.webp')) return 'image/webp'
  if (name.endsWith('.gif')) return 'image/gif'
  return ''
}

function renameWithExt(name: string, mime: string): string {
  const ext = mime === 'image/webp' ? '.webp' : mime === 'image/jpeg' ? '.jpg' : '.png'
  const base = name.replace(/\.[^.]+$/, '') || 'chart'
  return `${base}${ext}`
}

type DecodedImage = {
  width: number
  height: number
  draw: (ctx: CanvasRenderingContext2D, w: number, h: number) => void
  close: () => void
}

async function decodeImage(file: File): Promise<DecodedImage | null> {
  if (typeof document === 'undefined') return null

  // Skip decoding entirely when Canvas 2D is unavailable (e.g. headless DOM / jsdom).
  const probe = document.createElement('canvas')
  try {
    if (!probe.getContext?.('2d')) return null
  } catch {
    return null
  }

  if (typeof createImageBitmap === 'function') {
    try {
      const bitmap = await createImageBitmap(file)
      return {
        width: bitmap.width,
        height: bitmap.height,
        draw: (ctx, w, h) => ctx.drawImage(bitmap, 0, 0, w, h),
        close: () => bitmap.close(),
      }
    } catch {
      // Fallback to HTMLImageElement below when createImageBitmap rejects or is unavailable.
    }
  }

  if (typeof Image === 'undefined' || typeof URL?.createObjectURL !== 'function') return null

  const url = URL.createObjectURL(file)
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image()
      const timer = setTimeout(() => reject(new Error('decode timeout')), 4000)
      el.onload = () => {
        clearTimeout(timer)
        resolve(el)
      }
      el.onerror = () => {
        clearTimeout(timer)
        reject(new Error('decode failed'))
      }
      el.src = url
    })
    const width = img.naturalWidth || img.width
    const height = img.naturalHeight || img.height
    if (!width || !height) {
      URL.revokeObjectURL(url)
      return null
    }
    return {
      width,
      height,
      draw: (ctx, w, h) => ctx.drawImage(img, 0, 0, w, h),
      close: () => URL.revokeObjectURL(url),
    }
  } catch {
    URL.revokeObjectURL(url)
    return null
  }
}

function encodeCanvas(canvas: HTMLCanvasElement, mime: string, quality: number): Promise<Blob | null> {
  return new Promise((resolve) => {
    try {
      canvas.toBlob((blob) => resolve(blob), mime, quality)
    } catch {
      resolve(null)
    }
  })
}

async function renderCompressed(
  image: DecodedImage,
  maxSide: number,
  qualities: number[],
): Promise<Blob | null> {
  const scale = Math.min(1, maxSide / Math.max(image.width, image.height))
  const width = Math.max(1, Math.round(image.width * scale))
  const height = Math.max(1, Math.round(image.height * scale))

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) return null

  ctx.imageSmoothingEnabled = true
  ctx.imageSmoothingQuality = 'high'

  // Try WebP first (best quality-to-size for trading charts), fallback to JPEG.
  ctx.clearRect(0, 0, width, height)
  image.draw(ctx, width, height)

  let best: Blob | null = null
  const firstWebp = await encodeCanvas(canvas, 'image/webp', qualities[0]!)
  const supportsWebp = Boolean(firstWebp && firstWebp.type === 'image/webp')

  if (supportsWebp && firstWebp) {
    best = firstWebp
    if (best.size <= TARGET_BYTES) return best
    for (const q of qualities.slice(1)) {
      const candidate = await encodeCanvas(canvas, 'image/webp', q)
      if (candidate && candidate.type === 'image/webp' && candidate.size < best.size) {
        best = candidate
        if (best.size <= TARGET_BYTES) return best
      }
    }
    return best
  }

  // JPEG has no alpha channel; paint white backdrop to avoid black transparent regions.
  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, width, height)
  image.draw(ctx, width, height)

  for (const q of qualities) {
    const candidate = await encodeCanvas(canvas, 'image/jpeg', q)
    if (candidate && candidate.type === 'image/jpeg') {
      if (!best || candidate.size < best.size) best = candidate
      if (best.size <= TARGET_BYTES) return best
    }
  }

  return best
}

export async function compressChartImage(file: File): Promise<File> {
  const inputError = validateChartImageInput(file)
  if (inputError) throw new Error(inputError)

  const mime = normalizeMime(file)

  // Preserve animated GIFs when already within the backend upload limit.
  if (mime === 'image/gif' && file.size <= MAX_UPLOAD_BYTES) {
    return file
  }

  const decoded = await decodeImage(file)
  if (!decoded) {
    if (!BACKEND_MIMES.has(mime) || file.size > MAX_UPLOAD_BYTES) {
      throw new Error('Pilih PNG/JPEG/WebP/GIF maksimal 5 MiB.')
    }
    if (mime !== file.type) {
      return new File([file], file.name, { type: mime, lastModified: file.lastModified })
    }
    return file
  }

  try {
    const passes = [
      { maxSide: 1920, qualities: [0.84, 0.72, 0.58] },
      { maxSide: 1440, qualities: [0.72, 0.56] },
      { maxSide: 1080, qualities: [0.62, 0.48] },
    ]

    let best: Blob | null = null
    for (const pass of passes) {
      const candidate = await renderCompressed(decoded, pass.maxSide, pass.qualities)
      if (candidate && (!best || candidate.size < best.size)) {
        best = candidate
      }
      if (best && best.size <= TARGET_BYTES) break
    }

    // Keep original untouched if it is already compact, valid for backend, and smaller than re-encoded output.
    if (
      BACKEND_MIMES.has(mime) &&
      file.size <= TARGET_BYTES &&
      (!best || file.size <= best.size)
    ) {
      if (mime !== file.type) {
        return new File([file], file.name, { type: mime, lastModified: file.lastModified })
      }
      return file
    }

    if (best && best.size <= MAX_UPLOAD_BYTES && BACKEND_MIMES.has(best.type)) {
      return new File([best], renameWithExt(file.name, best.type), {
        type: best.type,
        lastModified: file.lastModified,
      })
    }

    if (BACKEND_MIMES.has(mime) && file.size <= MAX_UPLOAD_BYTES) {
      return file
    }

    throw new Error(
      'Ukuran gambar masih melebihi 5 MiB setelah dikompres. Coba crop atau kecilkan resolusi.',
    )
  } finally {
    decoded.close()
  }
}
