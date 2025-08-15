package main

import (
	"log"
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

	// Seed frame sizes after DB connection
	if err := seed.SeedFrameSizes(d); err != nil {
		log.Fatal("failed to seed frame sizes:", err)
	}

	s3, err := storage.New(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3UseSSL, cfg.S3Bucket)
	if err != nil {
		log.Fatal(err)
	}

	mailer := email.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.FromEmail)

	// Create one router instance
	r := gin.New()
	r.Use(gin.Recovery(), cors.New(cors.Config{
		AllowOrigins:     []string{"http://framelane-framer-app-v1.2.vercel.app", "http://localhost:3000"},
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
		S3: s3, Email: mailer,
	})

	hub := ws.NewHub()
	go hub.Run()

	r.GET("/ws", func(c *gin.Context) {
		ws.ServeWS(hub, c.Writer, c.Request)
	})

	log.Println("listening on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
