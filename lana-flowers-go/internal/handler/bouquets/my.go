package bouquets

import (
	"database/sql"
	"net/http"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// My — мои объявления (все статусы).
//   GET /bouquets/my
func My(c *gin.Context, db *sql.DB) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)

	rows, err := db.Query(`
		SELECT id, seller_id, title, description, price, city, photos, category, status, created_at
		FROM bouquets
		WHERE seller_id = $1
		ORDER BY created_at DESC
	`, uid)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	out := []Bouquet{}
	for rows.Next() {
		var b Bouquet
		var createdAt string
		if err := rows.Scan(&b.ID, &b.SellerID, &b.Title, &b.Description, &b.Price, &b.City,
			pq.Array(&b.Photos), &b.Category, &b.Status, &createdAt); err != nil {
			continue
		}
		b.CreatedAt = createdAt
		out = append(out, b)
	}

	response.OK(c, out)
}
