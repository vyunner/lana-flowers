// Client-side resize фото перед upload'ом. Айфоны делают снимки
// 8-12 MB, бэк-лимит 5 MB → юзер недоволен «фото отклонено». Здесь
// масштабируем до MAX_DIM-пикселей по большей стороне и жмём в JPEG
// с качеством 0.85 — обычно уходит в 200-800 KB при визуальном
// неотличии от оригинала на мини-апп-разрешении.

const MAX_DIM = 1920
const JPEG_QUALITY = 0.85
const TARGET_MIME = 'image/jpeg'

/**
 * Сжать картинку. Принимает File/Blob, возвращает Promise<File>.
 *
 * Если что-то пошло не так (старый браузер без OffscreenCanvas, кривой
 * формат) — возвращаем оригинал, не блокируем upload.
 */
export async function resizeImage(file) {
  if (!file || typeof file.type !== 'string' || !file.type.startsWith('image/')) {
    return file
  }
  try {
    const bitmap = await createImageBitmap(file)
    let { width, height } = bitmap
    if (width <= MAX_DIM && height <= MAX_DIM && file.size < 1.5 * 1024 * 1024) {
      // Маленькое и так — не трогаем (избегаем потери качества от
      // лишнего пережатия).
      bitmap.close()
      return file
    }
    if (width > MAX_DIM || height > MAX_DIM) {
      const scale = Math.min(MAX_DIM / width, MAX_DIM / height)
      width = Math.round(width * scale)
      height = Math.round(height * scale)
    }

    const canvas =
      typeof OffscreenCanvas !== 'undefined'
        ? new OffscreenCanvas(width, height)
        : Object.assign(document.createElement('canvas'), { width, height })
    const ctx = canvas.getContext('2d')
    ctx.drawImage(bitmap, 0, 0, width, height)
    bitmap.close()

    let blob
    if (canvas.convertToBlob) {
      blob = await canvas.convertToBlob({ type: TARGET_MIME, quality: JPEG_QUALITY })
    } else {
      blob = await new Promise((resolve) =>
        canvas.toBlob(resolve, TARGET_MIME, JPEG_QUALITY),
      )
    }
    if (!blob) return file

    // Если ужали в БОЛЬШИЙ файл (бывает на скриншотах) — оставляем оригинал.
    if (blob.size >= file.size) return file

    return new File([blob], renameToJpg(file.name), { type: TARGET_MIME })
  } catch {
    return file
  }
}

function renameToJpg(name) {
  if (!name) return 'photo.jpg'
  const i = name.lastIndexOf('.')
  return (i > 0 ? name.slice(0, i) : name) + '.jpg'
}
