package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/auth"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/handlers"
	"github.com/olamideolayemi/framelane-api/internal/payments"
	"github.com/olamideolayemi/framelane-api/internal/storage"
	"github.com/olamideolayemi/framelane-api/ws"
)

type Deps struct {
	DB              *gorm.DB
	JWTSecret       string
	JWTHours        int
	S3              *storage.S3
	Email           *email.Sender
	FrontendBaseURL string
	Hub             *ws.Hub
	Paystack        *payments.Paystack
}

func Setup(r *gin.Engine, d Deps) {
	r.GET("/v1/health", handlers.Health)

	fh := &handlers.FrameHandler{DB: d.DB}
	ch := &handlers.CustomizationHandler{DB: d.DB}
	carth := &handlers.CartHandler{DB: d.DB}
	addh := &handlers.AddressHandler{DB: d.DB}
	oh := &handlers.OrdersHandler{DB: d.DB, Email: d.Email, Hub: d.Hub}
	ph := &handlers.PaymentsHandler{DB: d.DB, Email: d.Email, Hub: d.Hub, Paystack: d.Paystack}
	promh := &handlers.PromoHandler{DB: d.DB}
	revh := &handlers.ReviewsHandler{DB: d.DB}
	refh := &handlers.ReferralsHandler{DB: d.DB}
	statsh := &handlers.AdminStatsHandler{DB: d.DB}

	// Auth
	ah := &handlers.AuthHandler{
		DB: d.DB, JWTSecret: d.JWTSecret, JWTHours: d.JWTHours,
		Email: d.Email, FrontendBaseURL: d.FrontendBaseURL,
	}
	r.POST("/v1/auth/register", ah.Register)
	r.POST("/v1/auth/login", ah.Login)
	r.GET("/v1/auth/verify", ah.VerifyEmail)
	r.POST("/v1/auth/forgot-password", ah.ForgotPassword)
	r.POST("/v1/auth/reset-password", ah.ResetPassword)

	// Public catalog
	r.GET("/v1/frames", fh.ListFrameTypes)
	r.GET("/v1/frames/size", fh.ListFrameSizes)
	r.GET("/v1/glasses", ch.ListGlasses)
	r.GET("/v1/laminations", ch.ListLaminations)
	r.GET("/v1/finishes", ch.ListFinishes)
	r.GET("/v1/reviews/frame/:frameId", revh.ListByFrame)

	// Public order tracking + payment webhook
	r.GET("/v1/track/:orderId", oh.Track)
	r.POST("/v1/payments/paystack/webhook", ph.Webhook)

	// User (auth + verified)
	uh := &handlers.UsersHandler{DB: d.DB}
	user := r.Group("/v1")
	user.Use(auth.RequireAuth(d.JWTSecret), auth.RequireVerified(d.DB))
	{
		user.GET("/upload-url", (&handlers.UploadHandler{S3: d.S3}).GetPresignedURL)
		user.PUT("/user/profile", uh.UpdateUserProfile)

		// Cart
		user.GET("/cart", carth.List)
		user.POST("/cart", carth.Add)
		user.PATCH("/cart/:id", carth.Update)
		user.DELETE("/cart/:id", carth.Remove)
		user.DELETE("/cart", carth.Clear)

		// Address book
		user.GET("/addresses", addh.List)
		user.POST("/addresses", addh.Create)
		user.PUT("/addresses/:id", addh.Update)
		user.DELETE("/addresses/:id", addh.Delete)

		// Orders
		user.GET("/orders", oh.ListMine)
		user.GET("/orders/:id", oh.GetMine)
		user.POST("/orders", oh.Create)
		user.POST("/orders/:id/reorder", oh.Reorder)
		user.GET("/orders/:id/events", oh.ListEvents)

		// Payments
		user.POST("/payments/paystack/initialize", ph.Initialize)
		user.GET("/payments/paystack/verify", ph.Verify)

		// Promo validation
		user.POST("/promo/validate", promh.Validate)

		// Reviews
		user.POST("/reviews", revh.Create)

		// Referral self-info
		user.GET("/referrals/me", refh.Me)
	}

	// Admin
	admin := r.Group("/v1/admin")
	admin.Use(auth.RequireAuth(d.JWTSecret), auth.RequireVerified(d.DB), auth.RequireAdmin())
	{
		// Orders
		admin.GET("/orders", oh.ListAll)
		admin.PATCH("/orders/:id/status", oh.UpdateStatus)
		admin.DELETE("/orders/:id", oh.DeleteOrder)

		// Users
		admin.GET("/users", uh.ListUsers)
		admin.GET("/users/:id", uh.GetUser)
		admin.PATCH("/users/:id/suspend", uh.SuspendUser)
		admin.DELETE("/users/:id", uh.DeleteUser)

		// Frames
		admin.POST("/frames", fh.CreateFrameType)
		admin.PUT("/frames/:id", fh.UpdateFrameType)
		admin.DELETE("/frames/:id", fh.DeleteFrameType)
		admin.POST("/frames/size", fh.CreateFrameSize)
		admin.PUT("/frames/size/:id", fh.UpdateFrameSize)
		admin.DELETE("/frames/size/:id", fh.DeleteFrameSize)

		// Customization
		admin.POST("/glasses", ch.CreateGlass)
		admin.PUT("/glasses/:id", ch.UpdateGlass)
		admin.DELETE("/glasses/:id", ch.DeleteGlass)
		admin.POST("/laminations", ch.CreateLamination)
		admin.PUT("/laminations/:id", ch.UpdateLamination)
		admin.DELETE("/laminations/:id", ch.DeleteLamination)
		admin.POST("/finishes", ch.CreateFinish)
		admin.PUT("/finishes/:id", ch.UpdateFinish)
		admin.DELETE("/finishes/:id", ch.DeleteFinish)

		// Promos
		admin.GET("/promos", promh.List)
		admin.POST("/promos", promh.Create)
		admin.PUT("/promos/:id", promh.Update)
		admin.DELETE("/promos/:id", promh.Delete)

		// Reviews moderation
		admin.GET("/reviews", revh.AdminList)
		admin.PATCH("/reviews/:id/status", revh.SetStatus)
		admin.DELETE("/reviews/:id", revh.Delete)

		// Dashboard stats
		admin.GET("/stats", statsh.Dashboard)
	}
}
