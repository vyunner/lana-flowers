package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	imgproc "lana-flowers-go/internal/image"
)

// runThumbsCmd — одноразовый сканер для генерации thumb'ов для старых
// фоток (загруженных ДО появления авто-генерации в upload.go).
//
// Использование: ./app thumbs /var/lib/lana-flowers/uploads
//
// Идёт рекурсивно по userId-папкам, для каждого .jpg/.png/.webp
// без существующего *_thumb.jpg — генерит thumb. Существующие пропускает
// (идемпотентно, можно запускать сколько угодно).
func runThumbsCmd(args []string) {
	root := "/var/lib/lana-flowers/uploads"
	if len(args) > 0 {
		root = args[0]
	}

	var processed, skipped, failed int
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		// Пропускаем уже сгенерированные thumb'ы
		if strings.Contains(info.Name(), "_thumb.") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			return nil
		}
		thumb := imgproc.ThumbPath(path)
		if _, err := os.Stat(thumb); err == nil {
			skipped++
			return nil
		}
		if err := imgproc.GenerateThumb(path, thumb); err != nil {
			log.Printf("thumb fail %s: %v", path, err)
			failed++
			return nil
		}
		processed++
		if processed%50 == 0 {
			fmt.Printf("processed %d (skipped %d, failed %d)...\n", processed, skipped, failed)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("walk: %v", err)
	}
	fmt.Printf("done. processed=%d skipped=%d failed=%d\n", processed, skipped, failed)
}
