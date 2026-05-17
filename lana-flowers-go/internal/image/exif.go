package image

import (
	"image"
	"io"
)

// readEXIFOrientation парсит JPEG-заголовок и возвращает значение
// EXIF-тега Orientation (1-8). Если тега нет / файл не JPEG / парс
// упал — возвращает 1 (no-op, изображение в правильной ориентации).
//
// Без внешних либ — ручной парс APP1 → TIFF → IFD0. Хватает 64KB
// первых байт файла (EXIF в JPEG лежит в начале, до данных самой
// картинки), так что читаем через io.LimitReader без аллокации
// всего файла.
//
// Значения orientation per EXIF spec:
//
//	1 normal · 2 flipH · 3 rot180 · 4 flipV
//	5 transpose · 6 rot90CW · 7 transverse · 8 rot90CCW
func readEXIFOrientation(r io.Reader) int {
	const def = 1
	data, err := io.ReadAll(io.LimitReader(r, 64*1024))
	if err != nil || len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return def
	}
	// Идём по JPEG-маркерам после SOI (FFD8). Ищем APP1 (FFE1).
	i := 2
	for i+4 < len(data) {
		if data[i] != 0xFF {
			return def
		}
		marker := data[i+1]
		// SOS (FFDA) — дальше payload изображения, EXIF не будет.
		if marker == 0xDA {
			return def
		}
		segLen := int(data[i+2])<<8 | int(data[i+3])
		if segLen < 2 || i+2+segLen > len(data) {
			return def
		}
		if marker == 0xE1 && segLen >= 8 && string(data[i+4:i+10]) == "Exif\x00\x00" {
			return parseEXIFOrientation(data[i+10 : i+2+segLen])
		}
		i += 2 + segLen
	}
	return def
}

func parseEXIFOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var be bool
	switch string(tiff[0:2]) {
	case "II":
		be = false
	case "MM":
		be = true
	default:
		return 1
	}
	u16 := func(off int) int {
		if off+2 > len(tiff) {
			return 0
		}
		if be {
			return int(tiff[off])<<8 | int(tiff[off+1])
		}
		return int(tiff[off+1])<<8 | int(tiff[off])
	}
	u32 := func(off int) int {
		if off+4 > len(tiff) {
			return 0
		}
		if be {
			return int(tiff[off])<<24 | int(tiff[off+1])<<16 | int(tiff[off+2])<<8 | int(tiff[off+3])
		}
		return int(tiff[off+3])<<24 | int(tiff[off+2])<<16 | int(tiff[off+1])<<8 | int(tiff[off])
	}
	if u16(2) != 0x002A {
		return 1
	}
	ifd0 := u32(4)
	if ifd0 < 8 || ifd0+2 > len(tiff) {
		return 1
	}
	n := u16(ifd0)
	for k := 0; k < n; k++ {
		e := ifd0 + 2 + k*12
		if e+12 > len(tiff) {
			return 1
		}
		if u16(e) == 0x0112 { // Orientation tag
			return u16(e + 8) // value-or-offset поле, для SHORT count=1 значение прямо здесь
		}
	}
	return 1
}

// applyOrientation возвращает новое RGBA-изображение, физически повёрнутое
// согласно EXIF Orientation. Для orientation=1 (нормальная ориентация) и
// невалидных значений отдаёт src без копирования.
//
// Реализация попиксельная (img.At + out.Set) — O(W*H). Для 12MP iPhone
// фотки на VPS отрабатывает за ~300мс, для нашего thumb-gen (раз на
// upload) приемлемо.
func applyOrientation(src image.Image, orient int) image.Image {
	if orient <= 1 || orient > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// Для поворотов на 90° (orient 5-8) ширина и высота меняются местами.
	var nw, nh int
	if orient >= 5 {
		nw, nh = h, w
	} else {
		nw, nh = w, h
	}

	out := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orient {
			case 2:
				dx, dy = w-1-x, y // flip H
			case 3:
				dx, dy = w-1-x, h-1-y // rotate 180
			case 4:
				dx, dy = x, h-1-y // flip V
			case 5:
				dx, dy = y, x // transpose
			case 6:
				dx, dy = h-1-y, x // rotate 90 CW
			case 7:
				dx, dy = h-1-y, w-1-x // transverse
			case 8:
				dx, dy = y, w-1-x // rotate 90 CCW
			}
			out.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}
