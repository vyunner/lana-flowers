package users

import (
	"database/sql"
	"net/http"
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

type userView struct {
	UserID              string `json:"user_id"`
	FirstName           string `json:"first_name"`     // из Telegram
	LastName            string `json:"last_name"`      // из Telegram
	Username            string `json:"username"`       // из Telegram
	PhotoURL            string `json:"photo_url"`      // из Telegram
	IsPremium           bool   `json:"is_premium"`
	LanguageCode        string `json:"language_code"`
	PhoneNumber         string `json:"phone_number"`
	DisplayName         string `json:"display_name"`         // кастомное имя в приложении
	AvatarURL           string `json:"avatar_url"`           // загруженное фото-аватар
	OnboardingCompleted bool   `json:"onboarding_completed"`
	IsRegistered        bool   `json:"is_registered"`
}

func Me(c *gin.Context, db *sql.DB) {
	uid, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user in context")
		return
	}
	loadAndReturn(c, db, uid.(string))
}

func Get(c *gin.Context, db *sql.DB) {
	id := c.Param("id")
	loadAndReturn(c, db, id)
}

func loadAndReturn(c *gin.Context, db *sql.DB, userID string) {
	var u userView
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
	AvatarURL         *string `json:"avatar_url,omitempty"`
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
		sets = append(sets, "display_name = $"+itoa(idx))
		args = append(args, name)
		idx++
	}
	if req.AvatarURL != nil {
		sets = append(sets, "avatar_url = $"+itoa(idx))
		args = append(args, *req.AvatarURL)
		idx++
	}
	if req.OnboardingCompleted != nil {
		sets = append(sets, "onboarding_completed = $"+itoa(idx))
		args = append(args, *req.OnboardingCompleted)
		idx++
	}

	if len(sets) == 0 {
		loadAndReturn(c, db, uid)
		return
	}

	q := "UPDATE users SET " + strings.Join(sets, ", ") + " WHERE user_id = $" + itoa(idx)
	args = append(args, uid)

	if _, err := db.Exec(q, args...); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	loadAndReturn(c, db, uid)
}

func itoa(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	buf := [16]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}
