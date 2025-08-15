package email

import (
	"log"
	"os"
	"strconv"
)

var SenderService *Sender

func Init() {
	port, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		log.Fatalf("Invalid SMTP_PORT: %v", err)
	}

	SenderService = New(
		os.Getenv("SMTP_HOST"),
		port,
		os.Getenv("SMTP_FROM_EMAIL"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASS"),
	)
}
