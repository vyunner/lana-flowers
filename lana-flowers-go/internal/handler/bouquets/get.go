package bouquets

import (
	"database/sql"
	"errors"
	"net/http"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

func Get(c *gin.Context, db *sql.DB) {
	id := c.Param("id")

	currentUser, _ := c.Get("user_id")
	uid, _ := currentUser.(string)

	q := bouquetWithSellerSelect + ` WHERE b.id = $2`
	b, err := scanBouquetWithSeller(db.QueryRow(q, uid, id))

	if errors.Is(err, sql.ErrNoRows) {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "bouquet not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	response.OK(c, b)
}
