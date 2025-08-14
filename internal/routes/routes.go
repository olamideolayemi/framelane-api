package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/auth"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/handlers"
	"github.com/olamideolayemi/framelane-api/internal/storage"
)

type Deps struct {
	DB        *gorm.DB
	JWTSecret string
	JWTHours  int
	S3        *storage.S3
	Email     *email.Sender
}

func Setup(r *gin.Engine, d Deps) {
	r.GET("/v1/health", handlers.Health)

	ah := &handlers.AuthHandler{DB: d.DB, JWTSecret: d.JWTSecret, JWTHours: d.JWTHours}
	r.POST("/v1/auth/register", ah.Register)
	r.POST("/v1/auth/login", ah.Login)

	uh := &handlers.UploadHandler{S3: d.S3}
	r.GET("/v1/upload-url", auth.RequireAuth(d.JWTSecret), uh.GetPresignedURL)

	oh := &handlers.OrdersHandler{DB: d.DB, Email: d.Email}
	r.POST("/v1/orders", oh.Create)
	r.GET("/v1/track/:orderId", oh.Track)

	// user
	user := r.Group("/v1")
	user.Use(auth.RequireAuth(d.JWTSecret))
	{
		user.GET("/orders", oh.ListMine)
	}

	// admin
	admin := r.Group("/v1/admin")
	admin.Use(auth.RequireAuth(d.JWTSecret), auth.RequireAdmin())
	{
		admin.GET("/orders", oh.ListAll)
		admin.PATCH("/orders/:id/status", oh.UpdateStatus)
	}
}
