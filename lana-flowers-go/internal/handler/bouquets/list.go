package bouquets

import (
	"database/sql"
	"net/http"
	"strconv"

	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

// List возвращает активные букеты с фильтрацией по городу/категории.
// Если в контексте есть user_id — для каждого букета добавляется my_offer
// (его собственный pending-оффер если есть), чтобы UI мог показать состояние.
//
//	GET /bouquets?city=Алматы&category=roses&limit=50
func List(c *gin.Context, db *sql.DB) {
	city := c.Query("city")
	category := c.Query("category")
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	currentUser, _ := c.Get("user_id")
	uid, _ := currentUser.(string)

	q := bouquetWithSellerSelect + ` WHERE b.status = '` + offers.BouquetActive + `'`
	args := []any{uid}

	if city != "" {
		args = append(args, city)
		q += ` AND b.city = $` + strconv.Itoa(len(args))
	}
	if category != "" && category != "all" {
		args = append(args, category)
		q += ` AND b.category = $` + strconv.Itoa(len(args))
	}
	args = append(args, limit)
	q += ` ORDER BY b.created_at DESC LIMIT $` + strconv.Itoa(len(args))

	rows, err := db.Query(q, args...)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	out := []Bouquet{}
	for rows.Next() {
		b, err := scanBouquetWithSeller(rows)
		if err != nil {
			continue
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	response.OK(c, out)
}
