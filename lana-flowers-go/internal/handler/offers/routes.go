package offers

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	g := r.Group("/offers")
	{
		g.POST("", func(c *gin.Context) { Create(c, db) })
		g.POST("/:id/respond", func(c *gin.Context) { Respond(c, db) })
		g.GET("/sent", func(c *gin.Context) { ListSent(c, db) })
		g.GET("/received", func(c *gin.Context) { ListReceived(c, db) })
	}
}

type Offer struct {
	ID          int64  `json:"id"`
	BouquetID   int64  `json:"bouquet_id"`
	BuyerID     string `json:"buyer_id"`
	SellerID    string `json:"seller_id"`
	Price       int64  `json:"price"`
	Message     string `json:"message"`
	Status      string `json:"status"`
	ParentID    *int64 `json:"parent_id,omitempty"`
	CreatedAt   string `json:"created_at"`
	RespondedAt string `json:"responded_at,omitempty"`
}
