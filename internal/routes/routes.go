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
	fh := &handlers.FrameHandler{DB: d.DB}

	ah := &handlers.AuthHandler{DB: d.DB, JWTSecret: d.JWTSecret, JWTHours: d.JWTHours}
	r.POST("/v1/auth/register", ah.Register)
	r.POST("/v1/auth/login", ah.Login)

	uh := &handlers.UploadHandler{S3: d.S3}
	r.GET("/v1/upload-url", auth.RequireAuth(d.JWTSecret), uh.GetPresignedURL)

	oh := &handlers.OrdersHandler{DB: d.DB, Email: d.Email}
	r.GET("/v1/track/:orderId", oh.Track)

	ph := &handlers.PaymentsHandler{DB: d.DB}
	r.POST("/v1/payments/intent", ph.CreateIntent)
	r.POST("/v1/payments/webhook", ph.Webhook)

	// Public routes
	r.GET("/v1/frames/size", fh.ListFrameSizes) // List all frame sizes
	r.GET("/v1/frames", fh.ListFrameTypes)      // List all frames

	// user
	user := r.Group("/v1")
	user.Use(auth.RequireAuth(d.JWTSecret))
	{
		user.GET("/orders", oh.ListMine)
		user.POST("/orders", oh.Create)

		uh := &handlers.UsersHandler{DB: d.DB}
		user.PUT("/user/profile", uh.UpdateUserProfile)
	}

	// admin
	admin := r.Group("/v1/admin")
	admin.Use(auth.RequireAuth(d.JWTSecret), auth.RequireAdmin())
	{
		admin.GET("/orders", oh.ListAll)
		admin.PATCH("/orders/:id/status", oh.UpdateStatus)
		admin.DELETE("/orders/:id", oh.DeleteOrder)

		// User management
		uh := &handlers.UsersHandler{DB: d.DB}
		admin.GET("/users", uh.ListUsers)
		admin.GET("/users/:id", uh.GetUser)
		admin.PATCH("/users/:id/suspend", uh.SuspendUser)
		admin.DELETE("/users/:id", uh.DeleteUser)

		// Frame sizes
		admin.POST("/frames/size", fh.CreateFrameSize)
		admin.PUT("/frames/size/:id", fh.UpdateFrameSize)
		admin.DELETE("/frames/size/:id", fh.DeleteFrameSize)

		// Frame types
		admin.POST("/frames", fh.CreateFrameType)
		admin.PUT("/frames/:id", fh.UpdateFrameType)
		admin.DELETE("/frames/:id", fh.DeleteFrameType)
	}
}
