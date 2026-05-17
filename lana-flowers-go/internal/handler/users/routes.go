package users

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	g := r.Group("/users")
	{
		g.GET("/me", func(c *gin.Context) { Me(c, db) })
		g.PATCH("/me", func(c *gin.Context) { UpdateMe(c, db) })
		g.GET("/:id", func(c *gin.Context) { Get(c, db) })
	}
}

// privateView — то, что юзер видит про СЕБЯ (GET /users/me). Содержит phone и
// onboarding-флаги — нельзя отдавать чужим.
type privateView struct {
	UserID              string `json:"user_id"`
	FirstName           string `json:"first_name"`
	LastName            string `json:"last_name"`
	Username            string `json:"username"`
	PhotoURL            string `json:"photo_url"`
	IsPremium           bool   `json:"is_premium"`
	LanguageCode        string `json:"language_code"`
	PhoneNumber         string `json:"phone_number"`
	DisplayName         string `json:"display_name"`
	AvatarURL           string `json:"avatar_url"`
	City                string `json:"city"`
	OnboardingCompleted bool   `json:"onboarding_completed"`
	IsRegistered        bool   `json:"is_registered"`
}

// allowedCities — те же города что во фронте (lana-flowers-web/src/data/cities.js).
// Whitelist чтобы юзер не смог через PATCH /users/me записать в city произвольную
// строку (мусор, XSS-payload, ёмкие emoji-комбинации).
// При расширении списка городов — синхронизировать оба места.
var allowedCities = map[string]bool{
	"Алматы":           true,
	"Астана":           true,
	"Шымкент":          true,
	"Караганда":        true,
	"Актобе":           true,
	"Атырау":           true,
	"Тараз":            true,
	"Павлодар":         true,
	"Усть-Каменогорск": true,
	"Семей":            true,
}

// publicView — то, что любой видит про чужого юзера (GET /users/:id).
// БЕЗ phone (приватные данные), без onboarding-флагов (внутренняя кухня).
type publicView struct {
	UserID      string `json:"user_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	PhotoURL    string `json:"photo_url"`
}

func Me(c *gin.Context, db *sql.DB) {
	uid, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user in context")
		return
	}
	loadPrivateAndReturn(c, db, uid.(string))
}

// Get — публичный профайл чужого юзера. Карточка букета показывает имя+аватар
// продавца — для этого endpoint и нужен. Телефон / phone-gate тут НЕ отдаём,
// иначе любой авторизованный мог бы листать все номера в базе.
func Get(c *gin.Context, db *sql.DB) {
	id := c.Param("id")
	var u publicView
	err := db.QueryRow(`
		SELECT user_id, first_name, last_name, display_name, avatar_url, photo_url
		FROM users WHERE user_id = $1
	`, id).Scan(&u.UserID, &u.FirstName, &u.LastName, &u.DisplayName, &u.AvatarURL, &u.PhotoURL)
	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	response.OK(c, u)
}

func loadPrivateAndReturn(c *gin.Context, db *sql.DB, userID string) {
	var u privateView
	err := db.QueryRow(`
		SELECT user_id, first_name, last_name, username, photo_url, is_premium, language_code,
		       phone_number, display_name, avatar_url, city, onboarding_completed
		FROM users WHERE user_id = $1
	`, userID).Scan(&u.UserID, &u.FirstName, &u.LastName, &u.Username, &u.PhotoURL, &u.IsPremium, &u.LanguageCode,
		&u.PhoneNumber, &u.DisplayName, &u.AvatarURL, &u.City, &u.OnboardingCompleted)
	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	u.IsRegistered = u.PhoneNumber != ""
	response.OK(c, u)
}

type patchMeReq struct {
	DisplayName         *string `json:"display_name,omitempty"`
	AvatarURL           *string `json:"avatar_url,omitempty"`
	City                *string `json:"city,omitempty"`
	OnboardingCompleted *bool   `json:"onboarding_completed,omitempty"`
}

// UpdateMe — частичное обновление. Принимаем только то, что юзер вправе менять про себя.
//   PATCH /users/me   { display_name?, avatar_url?, city?, onboarding_completed? }
func UpdateMe(c *gin.Context, db *sql.DB) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)

	var req patchMeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	sets := []string{}
	args := []any{}
	idx := 1

	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		// Считаем по rune'ам, иначе кириллица обрезалась бы по байтам и
		// получили бы битый UTF-8. Лимит 64 символа.
		if rs := []rune(name); len(rs) > 64 {
			name = string(rs[:64])
		}
		// Дополнительная зачистка: схлопываем подряд идущие пробелы/таб
		// в одинарный пробел, чтобы юзер не мог сделать имя «          а».
		name = strings.Join(strings.Fields(name), " ")
		sets = append(sets, "display_name = $"+strconv.Itoa(idx))
		args = append(args, name)
		idx++
	}
	if req.AvatarURL != nil {
		// Аватар принимаем либо пустой (юзер «удалил» фото) либо ссылку
		// только на наш /uploads/... — иначе можно поставить avatar_url
		// на трекинговый pixel или хост порно, и оно будет рендериться
		// в карточках продавца.
		av := strings.TrimSpace(*req.AvatarURL)
		if av != "" && !strings.HasPrefix(av, "/uploads/") &&
			!strings.HasPrefix(av, "https://64-226-107-161.nip.io/uploads/") {
			response.Err(c, http.StatusBadRequest, "BAD_REQUEST",
				"avatar must be from /upload")
			return
		}
		sets = append(sets, "avatar_url = $"+strconv.Itoa(idx))
		args = append(args, av)
		idx++
	}
	if req.City != nil {
		city := strings.TrimSpace(*req.City)
		// Whitelist — иначе можно записать произвольную строку или мусор,
		// который потом сломает фильтрацию каталога / отображение.
		if city != "" && !allowedCities[city] {
			response.Err(c, http.StatusBadRequest, "BAD_REQUEST",
				"unknown city: "+city)
			return
		}
		sets = append(sets, "city = $"+strconv.Itoa(idx))
		args = append(args, city)
		idx++
	}
	if req.OnboardingCompleted != nil {
		sets = append(sets, "onboarding_completed = $"+strconv.Itoa(idx))
		args = append(args, *req.OnboardingCompleted)
		idx++
	}

	if len(sets) == 0 {
		loadPrivateAndReturn(c, db, uid)
		return
	}

	q := "UPDATE users SET " + strings.Join(sets, ", ") + " WHERE user_id = $" + strconv.Itoa(idx)
	args = append(args, uid)

	if _, err := db.Exec(q, args...); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	loadPrivateAndReturn(c, db, uid)
}
