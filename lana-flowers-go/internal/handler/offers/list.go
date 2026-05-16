package offers

import (
	"database/sql"
	"net/http"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
)

// ListSent — мои отправленные офферы (я как покупатель).
//   GET /offers/sent
func ListSent(c *gin.Context, db *sql.DB) {
	listByField(c, db, "buyer_id")
}

// ListReceived — входящие на мои букеты (я как продавец).
//   GET /offers/received[?status=pending]
func ListReceived(c *gin.Context, db *sql.DB) {
	listByField(c, db, "seller_id")
}

func listByField(c *gin.Context, db *sql.DB, field string) {
	uidVal, ok := c.Get("user_id")
	if !ok {
		response.Err(c, http.StatusUnauthorized, "UNAUTHORIZED", "no user")
		return
	}
	uid := uidVal.(string)
	statusFilter := c.Query("status")

	q := `
		SELECT id, bouquet_id, buyer_id, seller_id, price, message, status, parent_id,
		       created_at, COALESCE(responded_at::text, '')
		FROM offers
		WHERE ` + field + ` = $1
	`
	args := []any{uid}
	if statusFilter != "" {
		q += ` AND status = $2`
		args = append(args, statusFilter)
	}
	q += ` ORDER BY created_at DESC`

	rows, err := db.Query(q, args...)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	out := []Offer{}
	for rows.Next() {
		var o Offer
		var parent sql.NullInt64
		if err := rows.Scan(&o.ID, &o.BouquetID, &o.BuyerID, &o.SellerID, &o.Price, &o.Message,
			&o.Status, &parent, &o.CreatedAt, &o.RespondedAt); err != nil {
			continue
		}
		if parent.Valid {
			v := parent.Int64
			o.ParentID = &v
		}
		out = append(out, o)
	}

	response.OK(c, out)
}
