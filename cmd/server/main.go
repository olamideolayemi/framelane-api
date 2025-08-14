package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/config"
	"github.com/olamideolayemi/framelane-api/internal/db"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/routes"
	"github.com/olamideolayemi/framelane-api/internal/storage"
)

func main() {
	cfg := config.Load()
	d := db.Connect(cfg.DatabaseURL)

	s3, err := storage.New(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3UseSSL, cfg.S3Bucket)
	if err != nil {
		log.Fatal(err)
	}

	mailer := email.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.FromEmail)

	r := gin.Default()
	routes.Setup(r, routes.Deps{
		DB: d, JWTSecret: cfg.JWTSecret, JWTHours: cfg.JWTExpiresH,
		S3: s3, Email: mailer,
	})

	log.Println("listening on http://localhost:8080")
	_ = r.Run(":8080")
}
