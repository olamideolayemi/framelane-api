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
			if err := clearExpiredPasswordResets(db); err != nil {
				log.Printf("failed to clear expired password resets: %v", err)
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

func clearExpiredPasswordResets(db *gorm.DB) error {
	return db.Model(&models.User{}).
		Where("password_reset_expires_at IS NOT NULL AND password_reset_expires_at < ?", time.Now()).
		Updates(map[string]interface{}{
			"password_reset_token_hash": "",
			"password_reset_sent_at":    nil,
			"password_reset_expires_at": nil,
		}).Error
}
