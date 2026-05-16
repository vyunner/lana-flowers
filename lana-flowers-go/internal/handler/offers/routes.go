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
