export interface CompressOptions {
  /** Sisi terpanjang maksimum (px). */
  maxSize?: number
  /** Kualitas WebP 0–1. */
  quality?: number
  /** File di bawah ukuran ini dan dimensinya sudah kecil tidak dikompres. */
  skipBelowBytes?: number
}

function loadImage(file: Blob): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const img = new Image()
    img.onload = () => {
      URL.revokeObjectURL(url)
      resolve(img)
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('Gambar tidak dapat dibaca'))
    }
    img.src = url
  })
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, quality))
}

/**
 * Kompres gambar di browser sebelum upload: canvas → WebP (fallback JPEG),
 * sisi terpanjang maks. 1920px, kualitas 0.82. File kecil / GIF / SVG dilewati.
 */
export async function compressImage(file: File, opts: CompressOptions = {}): Promise<File> {
  const maxSize = opts.maxSize ?? 1920
  const quality = opts.quality ?? 0.82
  const skipBelow = opts.skipBelowBytes ?? 300 * 1024

  if (!file.type.startsWith('image/') || file.type === 'image/gif' || file.type === 'image/svg+xml') return file

  let img: HTMLImageElement
  try {
    img = await loadImage(file)
  } catch {
    return file
  }

  const w = img.naturalWidth
  const h = img.naturalHeight
  const long = Math.max(w, h)
  if (!long) return file
  if (long <= maxSize && file.size <= skipBelow) return file

  const scale = Math.min(1, maxSize / long)
  const cw = Math.round(w * scale)
  const ch = Math.round(h * scale)
  const canvas = document.createElement('canvas')
  canvas.width = cw
  canvas.height = ch
  const ctx = canvas.getContext('2d')
  if (!ctx) return file
  ctx.drawImage(img, 0, 0, cw, ch)

  let blob = await canvasToBlob(canvas, 'image/webp', quality)
  if (!blob || blob.type !== 'image/webp') {
    // Browser tanpa encoder WebP → JPEG dengan latar putih.
    ctx.globalCompositeOperation = 'destination-over'
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, cw, ch)
    blob = await canvasToBlob(canvas, 'image/jpeg', quality)
  }
  if (!blob) return file
  if (blob.size >= file.size && scale === 1) return file

  const ext = blob.type === 'image/webp' ? 'webp' : 'jpg'
  const base = file.name.replace(/\.[^.]+$/, '') || 'gambar'
  return new File([blob], `${base}.${ext}`, { type: blob.type })
}
