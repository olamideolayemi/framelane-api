package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	tbgin "github.com/didip/tollbooth_gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/olamideolayemi/framelane-api/internal/config"
	"github.com/olamideolayemi/framelane-api/internal/db"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/models"
	"github.com/olamideolayemi/framelane-api/internal/payments"
	"github.com/olamideolayemi/framelane-api/internal/routes"
	"github.com/olamideolayemi/framelane-api/internal/seed"
	"github.com/olamideolayemi/framelane-api/internal/storage"
	"github.com/olamideolayemi/framelane-api/ws"
)

func main() {
	cfg := config.Load()
	email.Init()
	d := db.Connect(cfg.DatabaseURL)

	if err := d.AutoMigrate(&models.FrameSize{}, &models.Frame{}); err != nil {
		log.Fatal("Failed to migrate FrameSize table:", err)
	}

	// Backfill referral codes BEFORE the uniqueIndex on referral_code is enforced
	// (db.Connect runs AutoMigrate; here we just ensure no collisions for users
	// that existed before this column was added).
	if err := seed.BackfillReferralCodes(d); err != nil {
		log.Fatal("failed to backfill referral codes:", err)
	}

	// Seed frame sizes after DB connection
	if err := seed.SeedFrameSizes(d); err != nil {
		log.Fatal("failed to seed frame sizes:", err)
	}
	if err := seed.SeedFrames(d); err != nil {
		log.Fatal("failed to seed frames:", err)
	}
	if err := seed.SeedGlasses(d); err != nil {
		log.Fatal("failed to seed glasses:", err)
	}
	if err := seed.SeedLaminations(d); err != nil {
		log.Fatal("failed to seed laminations:", err)
	}
	if err := seed.EnsureAdminUser(d); err != nil {
		log.Fatal("failed to ensure admin user:", err)
	}
	seed.StartUnverifiedUserCleanup(d, time.Hour)

	s3, err := storage.New(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3UseSSL, cfg.S3Bucket)
	if err != nil {
		log.Fatal(err)
	}

	mailer := email.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.FromEmail)

	// Paystack client (lazy: handler will fall back to env-built one if nil)
	paystackClient := payments.NewPaystack(os.Getenv("PAYSTACK_SECRET"))

	// WebSocket hub for live admin updates
	hub := ws.NewHub()
	go hub.Run()

	// CORS allow-list is configured via CORS_ALLOWED_ORIGINS (comma-separated).
	// Defaults to localhost for development.
	originsRaw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if strings.TrimSpace(originsRaw) == "" {
		originsRaw = "http://localhost:3000"
	}
	origins := strings.Split(originsRaw, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}

	// Create one router instance
	r := gin.New()
	r.Use(gin.Recovery(), cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiting
	lim := tollbooth.NewLimiter(10, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
	r.Use(tbgin.LimitHandler(lim))

	// Register routes
	routes.Setup(r, routes.Deps{
		DB: d, JWTSecret: cfg.JWTSecret, JWTHours: cfg.JWTExpiresH,
		S3: s3, Email: mailer, FrontendBaseURL: cfg.FrontendBaseURL,
		Hub: hub, Paystack: paystackClient,
	})

	r.GET("/ws", func(c *gin.Context) {
		ws.ServeWS(hub, c.Writer, c.Request)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
