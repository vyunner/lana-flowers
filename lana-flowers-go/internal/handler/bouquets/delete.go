package bouquets

import (
	"database/sql"
	"net/http"

	"lana-flowers-go/internal/handler/offers"
	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

// Delete — мягкое снятие объявления (статус → archived). Только владелец.
//   DELETE /bouquets/:id
func Delete(c *gin.Context, db *sql.DB) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)
	id := c.Param("id")

	res, err := db.Exec(`
		UPDATE bouquets SET status = $3, updated_at = NOW()
		WHERE id = $1 AND seller_id = $2 AND status != $3
	`, id, uid, offers.BouquetArchived)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "bouquet not found or not yours")
		return
	}

	response.OK(c, gin.H{"archived": true})
}
