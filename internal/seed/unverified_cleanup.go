package seed

import (
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

func StartUnverifiedUserCleanup(db *gorm.DB, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			if err := purgeUnverifiedUsers(db, 24*time.Hour); err != nil {
				log.Printf("failed to purge unverified users: %v", err)
			}
			<-ticker.C
		}
	}()
}

func purgeUnverifiedUsers(db *gorm.DB, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	return db.Where("email_verified = ? AND email_verify_sent_at IS NOT NULL AND email_verify_sent_at < ? AND is_admin = ?", false, cutoff, false).
		Delete(&models.User{}).Error
}
