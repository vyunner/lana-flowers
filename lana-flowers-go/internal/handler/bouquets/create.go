package bouquets

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"lana-flowers-go/internal/adminbot"
	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

const (
	maxTitleLen       = 120
	maxDescriptionLen = 2000
	maxPhotos         = 5
)

// allowedCategories — белый список. Категория сверяется с этим списком,
// иначе можно создать букет с category="zalupa" и засрать БД мусором.
// Должен совпадать с CATEGORIES в SellSheet.vue. Категория "all" =
// «другое/смешанное» (sane default из фронта).
var allowedCategories = map[string]bool{
	"roses": true, "peonies": true, "wild": true,
	"composition": true, "dried": true, "all": true,
}

type createReq struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	Price       int64    `json:"price" binding:"required"`
	City        string   `json:"city" binding:"required"`
	Photos      []string `json:"photos"`
	Category    string   `json:"category"`
}

func Create(c *gin.Context, db *sql.DB) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)

	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.City = strings.TrimSpace(req.City)

	if req.Title == "" || req.City == "" {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", "title and city are required")
		return
	}
	if len([]rune(req.Title)) > maxTitleLen {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", "title too long")
		return
	}
	if len([]rune(req.Description)) > maxDescriptionLen {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", "description too long")
		return
	}
	if req.Price <= 0 {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", "price must be > 0")
		return
	}

	if req.Category == "" {
		req.Category = "all"
	}
	if !allowedCategories[req.Category] {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", "unknown category")
		return
	}

	// Photos: только URL'ы на наш собственный /uploads/<userID>/...
	// Без этой проверки можно прислать «photos: [evil.com/csam.jpg]» и
	// фронт послушно вставит в каталог — юридический риск для C2C.
	if req.Photos == nil {
		req.Photos = []string{}
	}
	if len(req.Photos) > maxPhotos {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", "too many photos")
		return
	}
	for _, p := range req.Photos {
		if !isOwnUploadURL(p) {
			response.Err(c, http.StatusBadRequest, "BAD_REQUEST",
				"photos must come from /upload (внешние URL не разрешены)")
			return
		}
	}

	var id int64
	err := db.QueryRowContext(c.Request.Context(), `
		INSERT INTO bouquets (seller_id, title, description, price, city, photos, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, uid, req.Title, req.Description, req.Price, req.City, pq.Array(req.Photos), req.Category).Scan(&id)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	sellerName, sellerUsername, sellerPhone := adminbot.LookupUser(db, uid)
	descPreview := req.Description
	if len(descPreview) > 120 {
		descPreview = descPreview[:120] + "…"
	}
	go adminbot.Record(db, adminbot.EventBouquetCreated,
		map[string]any{
			"bouquet_id":  id,
			"seller_id":   uid,
			"title":       req.Title,
			"price":       req.Price,
			"city":        req.City,
			"category":    req.Category,
			"photos":      len(req.Photos),
			"description": req.Description,
		},
		fmt.Sprintf(
			"🌸 <b>Новое объявление</b>\n\n"+
				"<b>%s</b>\n"+
				"Цена: <b>%s ₸</b>\n"+
				"Город: %s · %s\n"+
				"Фото: %d шт%s\n\n"+
				"Продавец: %s%s",
			adminbot.EscapeHTML(req.Title),
			adminbot.FormatPrice(req.Price),
			adminbot.EscapeHTML(req.City),
			req.Category,
			len(req.Photos),
			descSuffix(descPreview),
			adminbot.FormatUser(uid, sellerName, sellerUsername),
			phoneSuffix(sellerPhone),
		),
	)

	response.OK(c, gin.H{"id": id})
}

func descSuffix(d string) string {
	if d == "" {
		return ""
	}
	return "\n«" + adminbot.EscapeHTML(d) + "»"
}

func phoneSuffix(p string) string {
	if p == "" {
		return ""
	}
	return "\n📱 <code>" + p + "</code>"
}

// isOwnUploadURL — фото должно быть из нашего /upload-эндпоинта:
// абсолютный с нашим origin'ом ИЛИ относительный /uploads/<userID>/<file>.
// Сейчас upload.go возвращает «/uploads/<uid>/<rand>.<ext>» как URL.
func isOwnUploadURL(p string) bool {
	if p == "" {
		return false
	}
	// /uploads/... (наш formfat) или https://64-...nip.io/uploads/... либо
	// наш собственный домен. Достаточно проверить что path начинается с
	// /uploads/ — origin белый список проверяется через CORS.
	if strings.HasPrefix(p, "/uploads/") {
		return true
	}
	// Абсолютный URL: оставляем место для нашего домена в проде.
	if strings.HasPrefix(p, "https://64-226-107-161.nip.io/uploads/") {
		return true
	}
	return false
}
