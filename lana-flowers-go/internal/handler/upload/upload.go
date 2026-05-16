// Package upload — простая загрузка фото на диск + раздача статикой.
// Файлы лежат в /var/lib/lana-flowers/uploads/<userID>/<random>.<ext>
// и раздаются Caddy через /uploads/<userID>/<file>.
package upload

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	imgproc "lana-flowers-go/internal/image"
	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

const (
	maxFileSize = 5 << 20 // 5 MB
	uploadDir   = "/var/lib/lana-flowers/uploads"
	publicBase  = "/uploads"
	sniffBytes  = 512 // достаточно для DetectContentType
)

func RegisterRoutes(r *gin.Engine, _ *sql.DB) {
	r.POST("/upload", Upload)
}

func Upload(c *gin.Context) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)
	if uid == "" {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "empty user id")
		return
	}

	// Каждый юзер пишет в свою подпапку — изоляция, проще лимиты/квоты в будущем
	// + по URL видно автора (приватные данные не утекают, это только TG user_id).
	userDir := filepath.Join(uploadDir, sanitizeUID(uid))
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		response.Err(c, http.StatusInternalServerError, "FS_ERROR", err.Error())
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize+1024)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Err(c, http.StatusBadRequest, "BAD_FILE", err.Error())
		return
	}
	defer file.Close()

	if header.Size > maxFileSize {
		response.Err(c, http.StatusRequestEntityTooLarge, "TOO_BIG", "max 5MB")
		return
	}

	// Sniff'аем реальный content-type по первым 512 байтам — НЕ верим
	// заголовку Content-Type от клиента (он может прислать exe с заголовком
	// image/jpeg). DetectContentType из stdlib — это та же логика что в браузерах.
	ext, err := detectImageExt(file)
	if err != nil {
		if errors.Is(err, errUnsupportedType) {
			response.Err(c, http.StatusBadRequest, "BAD_TYPE", "only jpg/png/webp allowed")
		} else {
			response.Err(c, http.StatusBadRequest, "BAD_FILE", err.Error())
		}
		return
	}

	name, err := randomName(ext)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "RAND", err.Error())
		return
	}

	dst, err := os.Create(filepath.Join(userDir, name))
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "FS_ERROR", err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		response.Err(c, http.StatusInternalServerError, "WRITE", err.Error())
		return
	}

	// Закрываем dst перед чтением для thumb-генерации, иначе на некоторых
	// файловых системах буферы ещё не сброшены и Decode упадёт.
	_ = dst.Close()

	// Генерируем уменьшенную копию для каталога (400px JPEG q=70).
	// Если упало — НЕ валим upload: фронт умеет фолбэчить на оригинал
	// если thumb недоступен. Лучше «без оптимизации» чем «не загрузил».
	srcPath := filepath.Join(userDir, name)
	thumbPath := imgproc.ThumbPath(srcPath)
	if err := imgproc.GenerateThumb(srcPath, thumbPath); err != nil {
		log.Printf("upload: thumb generation failed for %s: %v", srcPath, err)
	}

	url := publicBase + "/" + sanitizeUID(uid) + "/" + name
	response.OK(c, gin.H{"url": url})
}

var errUnsupportedType = errors.New("unsupported image type")

// detectImageExt считывает первые 512 байт файла, определяет реальный тип
// и возвращает расширение. Курсор файла сбрасывается в начало, чтобы потом
// io.Copy записал ВЕСЬ файл, а не остаток.
func detectImageExt(f io.ReadSeeker) (string, error) {
	head := make([]byte, sniffBytes)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	ct := http.DetectContentType(head[:n])
	switch {
	case bytes.HasPrefix([]byte(ct), []byte("image/jpeg")):
		return ".jpg", nil
	case bytes.HasPrefix([]byte(ct), []byte("image/png")):
		return ".png", nil
	case bytes.HasPrefix([]byte(ct), []byte("image/webp")):
		return ".webp", nil
	}
	return "", errUnsupportedType
}

// sanitizeUID — оставляем только цифры. Telegram user_id это всегда целое
// число (int64), но на всякий случай защищаемся от path-traversal.
func sanitizeUID(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "_anon"
	}
	return string(out)
}

func randomName(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", hex.EncodeToString(b), ext), nil
}
