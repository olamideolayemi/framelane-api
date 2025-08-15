package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	JWTExpiresH int

	S3Endpoint  string
	S3UseSSL    bool
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3Region    string

	SMTPHost  string
	SMTPPort  int
	SMTPUser  string
	SMTPPass  string
	FromEmail string
}

func Load() *Config {
	_ = godotenv.Load()

	toInt := func(k string, def int) int {
		i, err := strconv.Atoi(os.Getenv(k))
		if err != nil {
			return def
		}
		return i
	}
	toBool := func(k string, def bool) bool {
		v := os.Getenv(k)
		if v == "" {
			return def
		}
		return v == "true" || v == "1"
	}

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTExpiresH: toInt("JWT_EXPIRES_HOURS", 720),

		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3UseSSL:    toBool("S3_USE_SSL", false),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),
		S3Bucket:    os.Getenv("S3_BUCKET"),
		S3Region:    os.Getenv("S3_REGION"),

		FromEmail: os.Getenv("SMTP_FROM_EMAIL"),
		SMTPHost:  os.Getenv("SMTP_HOST"),
		SMTPPort:  toInt("SMTP_PORT", 587),
		SMTPUser:  os.Getenv("SMTP_USER"),
		SMTPPass:  os.Getenv("SMTP_PASS"),
	}
	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" {
		log.Fatal("Missing critical env vars")
	}
	return cfg
}
