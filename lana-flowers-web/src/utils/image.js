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
    // imageOrientation: 'from-image' — критично: iPhone снимает фото в
    // физически landscape-пикселях с EXIF-тегом поворота. Без этого флага
    // createImageBitmap возвращает сырые пиксели, EXIF теряется, canvas →
    // JPEG записывает повёрнутую картинку как «правильную». Браузеры до
    // ~2020 имели разный дефолт, явный from-image гарантирует ориентацию
    // на всех платформах.
    const bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' })
    let { width, height } = bitmap
    // ВНИМАНИЕ: НЕ возвращаем оригинал даже если файл маленький — EXIF в
    // нём может быть, а бэк-thumb генератор (Go image/jpeg) ориентацию не
    // применяет → thumb приедет повёрнутым. Лучше один лишний прогон
    // через canvas (потеря качества при q=0.85 минимальна) чем
    // непредсказуемый поворот в каталоге.
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

// THUMB_CACHE_BUSTER — версия thumb'ов. Инкрементируется когда бэк делает
// массовую перегенерацию всех _thumb.jpg (логика поменялась, например
// EXIF rotation). Telegram WebView и браузеры кешируют картинки по URL
// очень агрессивно (Cache-Control headers они часто игнорируют), и без
// смены URL юзер видит старую закешированную версию.
//
// При обычной upload-операции каждое фото имеет уникальное имя файла
// (random hash), так что коллизии нет — это нужно только для регенов.
const THUMB_CACHE_BUSTER = 2

/**
 * thumbUrl — для каталога/списков, где фото показывается мелко.
 * Бэк при upload кладёт <name>_thumb.jpg рядом с оригиналом (400px
 * JPEG q=70, ~30-80KB вместо ~500KB оригинала).
 *
 * Если URL'а нет или формат странный — возвращаем оригинал (fallback).
 * Если thumb физически не сгенерирован (старые фотки до фичи) — сервер
 * вернёт 404 и браузер покажет broken-image. Фронт защищается через
 * <img onerror>, который переключает на оригинал.
 */
export function thumbUrl(originalUrl) {
  if (!originalUrl) return ''
  const dot = originalUrl.lastIndexOf('.')
  const slash = originalUrl.lastIndexOf('/')
  if (dot < 0 || dot < slash) return originalUrl
  return originalUrl.slice(0, dot) + '_thumb.jpg?v=' + THUMB_CACHE_BUSTER
}
