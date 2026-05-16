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
	OnboardingCompleted bool   `json:"onboarding_completed"`
	IsRegistered        bool   `json:"is_registered"`
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
		       phone_number, display_name, avatar_url, onboarding_completed
		FROM users WHERE user_id = $1
	`, userID).Scan(&u.UserID, &u.FirstName, &u.LastName, &u.Username, &u.PhotoURL, &u.IsPremium, &u.LanguageCode,
		&u.PhoneNumber, &u.DisplayName, &u.AvatarURL, &u.OnboardingCompleted)
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
	OnboardingCompleted *bool   `json:"onboarding_completed,omitempty"`
}

// UpdateMe — частичное обновление. Принимаем только то, что юзер вправе менять про себя.
//   PATCH /users/me   { display_name?, avatar_url?, onboarding_completed? }
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
		if len(name) > 64 {
			name = name[:64]
		}
		sets = append(sets, "display_name = $"+strconv.Itoa(idx))
		args = append(args, name)
		idx++
	}
	if req.AvatarURL != nil {
		sets = append(sets, "avatar_url = $"+strconv.Itoa(idx))
		args = append(args, *req.AvatarURL)
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
