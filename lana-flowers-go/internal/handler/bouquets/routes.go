package bouquets

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	g := r.Group("/bouquets")
	{
		g.GET("", func(c *gin.Context) { List(c, db) })
		g.POST("", func(c *gin.Context) { Create(c, db) })
		g.GET("/my", func(c *gin.Context) { My(c, db) })
		g.GET("/:id", func(c *gin.Context) { Get(c, db) })
		g.DELETE("/:id", func(c *gin.Context) { Delete(c, db) })
	}
}

// Bouquet — DTO для ответа.
type Bouquet struct {
	ID          int64       `json:"id"`
	SellerID    string      `json:"seller_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Price       int64       `json:"price"`
	City        string      `json:"city"`
	Photos      []string    `json:"photos"`
	Category    string      `json:"category"`
	Status      string      `json:"status"`
	CreatedAt   string      `json:"created_at"`
	Seller      *SellerInfo `json:"seller,omitempty"`
	// MyOffer — если у текущего юзера есть pending-оффер на этот букет.
	MyOffer *MyOfferInfo `json:"my_offer,omitempty"`
}

type SellerInfo struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type MyOfferInfo struct {
	ID    int64 `json:"id"`
	Price int64 `json:"price"`
}
