package bouquets

import (
	"database/sql"
	"net/http"
	"strings"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

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
	req.City = strings.TrimSpace(req.City)
	if req.Title == "" || req.City == "" {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", "title and city are required")
		return
	}
	if req.Price <= 0 {
		response.Err(c, http.StatusBadRequest, "BAD_REQUEST", "price must be > 0")
		return
	}
	if req.Category == "" {
		req.Category = "all"
	}
	if req.Photos == nil {
		req.Photos = []string{}
	}

	var id int64
	err := db.QueryRow(`
		INSERT INTO bouquets (seller_id, title, description, price, city, photos, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, uid, req.Title, req.Description, req.Price, req.City, pq.Array(req.Photos), req.Category).Scan(&id)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	response.OK(c, gin.H{"id": id})
}
