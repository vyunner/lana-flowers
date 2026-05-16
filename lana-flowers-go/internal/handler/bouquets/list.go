package bouquets

import (
	"database/sql"
	"net/http"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
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
		var n int
		_, _ = jsonInt(v, &n)
		if n > 0 && n <= 100 {
			limit = n
		}
	}

	currentUser, _ := c.Get("user_id")
	uid, _ := currentUser.(string)

	q := `
		SELECT b.id, b.seller_id, b.title, b.description, b.price, b.city,
		       b.photos, b.category, b.status, b.created_at,
		       COALESCE(NULLIF(u.display_name, ''), u.first_name) AS seller_name,
		       u.avatar_url,
		       o.id AS my_offer_id,
		       o.price AS my_offer_price
		FROM bouquets b
		JOIN users u ON u.user_id = b.seller_id
		LEFT JOIN offers o
		  ON o.bouquet_id = b.id
		 AND o.buyer_id = $1
		 AND o.status = 'pending'
		WHERE b.status = 'active'
	`
	args := []any{uid}
	if city != "" {
		q += ` AND b.city = $` + plus(len(args)+1)
		args = append(args, city)
	}
	if category != "" && category != "all" {
		q += ` AND b.category = $` + plus(len(args)+1)
		args = append(args, category)
	}
	q += ` ORDER BY b.created_at DESC LIMIT $` + plus(len(args)+1)
	args = append(args, limit)

	rows, err := db.Query(q, args...)
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	out := []Bouquet{}
	for rows.Next() {
		var b Bouquet
		var seller SellerInfo
		var createdAt string
		var myOfferID sql.NullInt64
		var myOfferPrice sql.NullInt64
		if err := rows.Scan(&b.ID, &b.SellerID, &b.Title, &b.Description, &b.Price, &b.City,
			pq.Array(&b.Photos), &b.Category, &b.Status, &createdAt,
			&seller.DisplayName, &seller.AvatarURL,
			&myOfferID, &myOfferPrice); err != nil {
			continue
		}
		seller.UserID = b.SellerID
		b.Seller = &seller
		b.CreatedAt = createdAt
		if myOfferID.Valid {
			b.MyOffer = &MyOfferInfo{ID: myOfferID.Int64, Price: myOfferPrice.Int64}
		}
		out = append(out, b)
	}

	response.OK(c, out)
}

func jsonInt(s string, dst *int) (int, error) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, nil
		}
		n = n*10 + int(ch-'0')
	}
	*dst = n
	return n, nil
}

func plus(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	buf := [10]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}
