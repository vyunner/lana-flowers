// Package image — генерация уменьшенных копий фотографий для каталога.
// Чисто stdlib + golang.org/x/image/draw, без CGO/libwebp.
//
// На каждое загруженное фото создаём <name>_thumb.jpg с длинной стороной
// THUMB_MAX_DIM (400px) и JPEG-качеством THUMB_QUALITY (70). Каталог на
// 50 букетов с 800KB-фотками = 40MB; с thumb'ами = ~4MB. Экономия 10x.
//
// WebP пока не делаем — нужен CGO + libwebp-dev. JPEG q=70 даёт 80%
// эффекта без головной боли с зависимостями.
package image

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png" // регистрация декодера PNG
	"io"
	"os"

	"golang.org/x/image/draw"
)

const (
	thumbMaxDim   = 400
	thumbQuality  = 70
	thumbSuffix   = "_thumb.jpg"
)

// ThumbPath возвращает путь к thumb'у для исходного файла.
// "/var/lib/.../uploads/123/abc.jpg" → "/var/lib/.../uploads/123/abc_thumb.jpg"
func ThumbPath(src string) string {
	// Срезаем расширение, добавляем _thumb.jpg
	dot := -1
	for i := len(src) - 1; i >= 0; i-- {
		if src[i] == '.' {
			dot = i
			break
		}
		if src[i] == '/' {
			break
		}
	}
	if dot < 0 {
		return src + thumbSuffix
	}
	return src[:dot] + thumbSuffix
}

// GenerateThumb читает исходное изображение, ресайзит до thumbMaxDim
// (по длинной стороне, сохраняя пропорции) и сохраняет JPEG'ом.
// Если src уже меньше — копируем как есть в JPEG (унификация формата).
//
// Ошибки декодирования (юзер прислал что-то странное) пробрасываются
// caller'у — он решает фолбэчить ли на отсутствие thumb'а.
func GenerateThumb(srcPath, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// EXIF orientation читаем ДО декода (iPhone JPEG-фотки приходят с
	// тегом «rotate 90° CW»). image.Decode сырые пиксели как есть, без
	// поворота — поэтому пост-фактум применяем applyOrientation, иначе
	// thumb приедет повёрнутым.
	orient := readEXIFOrientation(bytes.NewReader(data))

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	src = applyOrientation(src, orient)

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	var nw, nh int
	if w <= thumbMaxDim && h <= thumbMaxDim {
		nw, nh = w, h
	} else if w > h {
		nw = thumbMaxDim
		nh = h * thumbMaxDim / w
	} else {
		nh = thumbMaxDim
		nw = w * thumbMaxDim / h
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	// CatmullRom — высокое качество ресэмплинга, медленнее чем
	// ApproxBiLinear, но фотки маленькие → разница в мс.
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, dst, &jpeg.Options{Quality: thumbQuality})
}

// GenerateThumbStream — вариант для in-memory случаев. Не используется
// сейчас, но пригодится если перейдём на S3 (нет os.File, есть io.Reader).
func GenerateThumbStream(src io.Reader, dst io.Writer) error {
	// Читаем весь поток в память — нужно дважды (EXIF + Decode), а io.Reader
	// одноразовый. Для типичных фоток ≤10MB это норм.
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	orient := readEXIFOrientation(bytes.NewReader(data))
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	img = applyOrientation(img, orient)
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	var nw, nh int
	if w <= thumbMaxDim && h <= thumbMaxDim {
		nw, nh = w, h
	} else if w > h {
		nw = thumbMaxDim
		nh = h * thumbMaxDim / w
	} else {
		nh = thumbMaxDim
		nw = w * thumbMaxDim / h
	}
	out := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(out, out.Bounds(), img, bounds, draw.Over, nil)
	return jpeg.Encode(dst, out, &jpeg.Options{Quality: thumbQuality})
}
