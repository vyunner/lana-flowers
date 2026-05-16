package bouquets

import (
	"database/sql"
	"net/http"

	"lana-flowers-go/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

func Get(c *gin.Context, db *sql.DB) {
	id := c.Param("id")

	currentUser, _ := c.Get("user_id")
	uid, _ := currentUser.(string)

	var b Bouquet
	var seller SellerInfo
	var createdAt string
	var myOfferID sql.NullInt64
	var myOfferPrice sql.NullInt64
	err := db.QueryRow(`
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
		WHERE b.id = $2
	`, uid, id).Scan(&b.ID, &b.SellerID, &b.Title, &b.Description, &b.Price, &b.City,
		pq.Array(&b.Photos), &b.Category, &b.Status, &createdAt,
		&seller.DisplayName, &seller.AvatarURL,
		&myOfferID, &myOfferPrice)

	if err == sql.ErrNoRows {
		response.Err(c, http.StatusNotFound, "NOT_FOUND", "bouquet not found")
		return
	}
	if err != nil {
		response.Err(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	seller.UserID = b.SellerID
	b.Seller = &seller
	b.CreatedAt = createdAt
	if myOfferID.Valid {
		b.MyOffer = &MyOfferInfo{ID: myOfferID.Int64, Price: myOfferPrice.Int64}
	}

	response.OK(c, b)
}
