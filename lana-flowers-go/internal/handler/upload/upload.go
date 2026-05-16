// Package upload — простая загрузка фото на диск + раздача статикой.
// Файлы лежат в /var/lib/lana-flowers/uploads и раздаются Caddy через /uploads/<name>.
package upload

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

const (
	maxFileSize = 5 << 20 // 5 MB
	uploadDir   = "/var/lib/lana-flowers/uploads"
	publicBase  = "/uploads"
)

func RegisterRoutes(r *gin.Engine, _ *sql.DB) {
	r.POST("/upload", Upload)
}

func Upload(c *gin.Context) {
	if _, ok := c.Get("user_id"); !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
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

	ext := allowedExt(header)
	if ext == "" {
		response.Err(c, http.StatusBadRequest, "BAD_TYPE", "only jpg/png/webp allowed")
		return
	}

	name, err := randomName(ext)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "RAND", err.Error())
		return
	}

	dst, err := os.Create(filepath.Join(uploadDir, name))
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "FS_ERROR", err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		response.Err(c, http.StatusInternalServerError, "WRITE", err.Error())
		return
	}

	url := publicBase + "/" + name
	response.OK(c, gin.H{"url": url})
}

func allowedExt(h *multipart.FileHeader) string {
	ct := h.Header.Get("Content-Type")
	switch ct {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	}
	// fallback по расширению имени
	switch strings.ToLower(filepath.Ext(h.Filename)) {
	case ".jpg", ".jpeg":
		return ".jpg"
	case ".png":
		return ".png"
	case ".webp":
		return ".webp"
	}
	return ""
}

func randomName(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", hex.EncodeToString(b), ext), nil
}
