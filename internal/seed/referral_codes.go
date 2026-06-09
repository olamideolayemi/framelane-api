package seed

import (
	"crypto/rand"
	"encoding/base32"
	"strings"

	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

// BackfillReferralCodes assigns a unique referral code to any user that
// doesn't have one. Must run before the uniqueIndex on referral_code
// becomes problematic (multiple empty strings collide).
func BackfillReferralCodes(db *gorm.DB) error {
	var users []models.User
	if err := db.Where("referral_code IS NULL OR referral_code = ''").Find(&users).Error; err != nil {
		return err
	}
	for i := range users {
		code, err := generateReferralCode()
		if err != nil {
			return err
		}
		if err := db.Model(&users[i]).Update("referral_code", code).Error; err != nil {
			return err
		}
	}
	return nil
}

func generateReferralCode() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimRight(base32.StdEncoding.EncodeToString(b), "=")), nil
}

// GenerateReferralCode is exported so registration can use the same routine.
func GenerateReferralCode() (string, error) { return generateReferralCode() }
